package mix_layer

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net_utils"
	"sync"
)

type MixLayerAdaptor struct {
	net.Conn
	key      []byte
	ivKey    []byte
	coder    EnDec
	rbuf     bytes.Buffer
	decbuf   bytes.Buffer
	initRead bool
}

const MAX_CRYPT_PACKET_LEN = 64 * 1024
const HMAC_LENGTH = 20

var cryptMemPool = &sync.Pool{
	New: func() interface{} {
		return make([]byte, MAX_CRYPT_PACKET_LEN)
	},
}

func PKCS5Padding(src []byte, blockSize int) []byte {
	padding := blockSize - len(src)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	totalData := make([]byte, len(src)+len(padtext))
	copy(totalData, src)
	copy(totalData[len(src):], padtext)
	return totalData
}

func PKCS5UnPadding(src []byte) ([]byte, error) {
	length := len(src)
	unpadding := int(src[length-1])
	if length <= unpadding {
		return nil, errors.New(fmt.Sprintf("invalid pad data, length:%d, unpadding:%d", length, unpadding))
	}
	return src[:(length - unpadding)], nil
}

//datalen(4) + iv(variable) + data(variable) + hmac-sha1(20)
func EncodeHeadFrame(src []byte, dst []byte, ivin []byte, key []byte) (int, error) {
	if len(dst) < len(src)+len(ivin)+4+HMAC_LENGTH {
		return 0, errors.New(fmt.Sprintf("buffer too small, src len:%d, buf len:%d", len(src), len(dst)))
	}
	binary.BigEndian.PutUint32(dst, uint32(len(src)+len(ivin)+4+HMAC_LENGTH))
	copy(dst[4:], ivin)
	copy(dst[4+len(ivin):], src)
	hasher := hmac.New(sha1.New, key)
	hasher.Write(dst[4 : 4+len(ivin)+len(src)])
	hmacSum := hasher.Sum(nil)
	copy(dst[4+len(ivin)+len(src):], hmacSum)
	return 4 + len(src) + len(ivin) + len(hmacSum), nil
}

func CheckHeadFrame(src []byte, ivlen int, maxData int) (int, error) {
	if len(src) <= 4+HMAC_LENGTH+ivlen {
		return 0, nil
	}
	total := int(binary.BigEndian.Uint32(src))
	if total <= 0 {
		return -2, errors.New(fmt.Sprintf("invalid data frame len:%d", total))
	}
	if total > maxData {
		return -1, errors.New(fmt.Sprintf("data too long, skip, data len:%d, max len:%d", total, maxData))
	}
	if total > len(src) {
		return 0, nil
	}
	return total, nil
}

//return decode data len, iv, error
func DecodeHeadFrame(src []byte, dst []byte, ivout []byte, key []byte) (int, error) {
	sz, err := CheckHeadFrame(src, len(ivout), MAX_CRYPT_PACKET_LEN)
	if sz <= 0 {
		return 0, errors.New(fmt.Sprintf("package check fail, sz:%d, err:%v", sz, err))
	}
	hasher := hmac.New(sha1.New, key)
	hasher.Write(src[4 : len(src)-HMAC_LENGTH])
	hmacSum := hasher.Sum(nil)
	if !bytes.Equal(hmacSum, src[len(src)-HMAC_LENGTH:]) {
		return 0, errors.New(fmt.Sprintf("hmac check fail, acquire hmac:%s, but get hmac:%s",
			hex.EncodeToString(hmacSum), hex.EncodeToString(src[len(src)-HMAC_LENGTH:])))
	}
	ivlen := copy(ivout, src[4:])
	return copy(dst, src[4+ivlen:sz-HMAC_LENGTH]), nil
}

func (this *MixLayerAdaptor) SetKey(key string) {
	k := sha1.Sum([]byte(key))
	this.key = []byte(k[:])
}

func (this *MixLayerAdaptor) Read(b []byte) (int, error) {
	if this.decbuf.Len() != 0 {
		n := copy(b, this.decbuf.Bytes())
		this.decbuf.Next(n)
		return n, nil
	}
	tmp := cryptMemPool.Get().([]byte)
	defer cryptMemPool.Put(tmp)
	index := 0
	var err error
	var ivout []byte
	//首次读需要获取到iv并初始化解码器
	if !this.initRead {
		ivout = make([]byte, this.coder.IVLen())
	}
	for {
		index, err = this.Conn.Read(tmp)
		if err != nil {
			return 0, err
		}
		this.rbuf.Write(tmp[:index])
		cnt, err := CheckHeadFrame(this.rbuf.Bytes(), len(ivout), MAX_CRYPT_PACKET_LEN)
		if cnt != 0 || err != nil {
			break
		}
	}
	enc := tmp
	raw := cryptMemPool.Get().([]byte)
	defer cryptMemPool.Put(raw)
	for {
		frameLen, err := CheckHeadFrame(this.rbuf.Bytes(), len(ivout), MAX_CRYPT_PACKET_LEN)
		if err != nil || frameLen < 0 {
			return 0, errors.New(fmt.Sprintf("check frame data fail, err:%v, cnt:%d", err, frameLen))
		}
		if frameLen == 0 {
			break
		}
		encLen, err := DecodeHeadFrame(this.rbuf.Bytes()[:frameLen], enc, ivout, this.key)
		if err != nil {
			return 0, errors.New(fmt.Sprintf("decode head frame data fail, err:%v", err))
		}
		if !this.initRead {
			ierr := this.coder.InitRead(this.key, ivout)
			if err != nil {
				return 0, errors.New(fmt.Sprintf("init read coder fail, err:%v, key hex:%s, iv hex:%s",
					ierr, hex.EncodeToString(this.key), hex.EncodeToString(ivout)))
			}
			this.initRead = true
			ivout = nil
		}
		this.rbuf.Next(frameLen)
		rawLen, err := this.coder.Decode(enc[:encLen], raw)
		if err != nil {
			return 0, errors.New(fmt.Sprintf("decode aes data fail, err:%v, data len:%d, coder:%s", err, encLen, this.coder.Name()))
		}
		rawWrite := raw[:rawLen]
		this.decbuf.Write(rawWrite)
	}
	if this.decbuf.Len() <= 0 {
		return 0, errors.New(fmt.Sprintf("no more data, may has err"))
	}
	cnt := copy(b, this.decbuf.Bytes())
	this.decbuf.Next(cnt)
	return cnt, nil
}

func (this *MixLayerAdaptor) Write(b []byte) (int, error) {
	if len(b) >= 2*MAX_CRYPT_PACKET_LEN/3 {
		b = b[:2*MAX_CRYPT_PACKET_LEN/3]
	}
	enc := cryptMemPool.Get().([]byte)
	defer cryptMemPool.Put(enc)
	encLen, err := this.coder.Encode(b, enc)
	if err != nil {
		return 0, err
	}
	encData := enc[:encLen]
	frame := cryptMemPool.Get().([]byte)
	defer cryptMemPool.Put(frame)
	frameLen, err := EncodeHeadFrame(encData, frame, this.ivKey, this.key)
	if this.ivKey != nil {
		this.ivKey = nil
	}
	if err != nil {
		return 0, err
	}
	frameData := frame[:frameLen]
	if err = net_utils.SendSpecLen(this.Conn, frameData); err != nil {
		return 0, err
	}
	return len(b), nil
}

func RandIV(sz int) []byte {
	if sz <= 0 {
		return nil
	}
	iv := make([]byte, sz)
	rand.Reader.Read(iv)
	return iv
}

func GenKeyWithPad(key []byte, padto int) []byte {
	if len(key) == padto {
		return key
	}
	finalKey := make([]byte, padto)
	copy(finalKey, key)
	return finalKey
}

func CryptWrap(key string, conn net.Conn, coder EnDec) (*MixLayerAdaptor, error) {
	as := &MixLayerAdaptor{Conn: conn}
	as.SetKey(key)
	as.coder = coder
	as.ivKey = RandIV(coder.IVLen())
	err := as.coder.InitWrite(as.key, as.ivKey)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("init write fail, err:%v, key hex:%s, iv hex:%s",
			err, hex.EncodeToString(as.key), hex.EncodeToString(as.ivKey)))
	}
	return as, nil
}
