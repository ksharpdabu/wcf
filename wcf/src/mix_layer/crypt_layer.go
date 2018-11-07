package mix_layer

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net_utils"
	"sync"
)

type MixLayerAdaptor struct {
	net.Conn
	key          []byte
	ivKey        []byte
	coder        EnDec
	rbuf         bytes.Buffer
	decbuf       bytes.Buffer
	initRead     bool
	enableEncode bool
	enableDecode bool
}

const MAX_CRYPT_PACKET_LEN = 64 * 1024
const HMAC_LENGTH = 20

var cryptMemPool = &sync.Pool{
	New: func() interface{} {
		return make([]byte, MAX_CRYPT_PACKET_LEN)
	},
}

func (this *MixLayerAdaptor) SetKey(key string) {
	k := sha1.Sum([]byte(key))
	this.key = []byte(k[:])
}

func (this *MixLayerAdaptor) DisableEncode() {
	this.enableEncode = false
}

func (this *MixLayerAdaptor) DisableDecode() {
	this.enableDecode = false
}

func (this *MixLayerAdaptor) Read(b []byte) (int, error) {
	//禁用了解碼后, 原先的數據應該是在rbuf中
	if !this.enableDecode {
		if this.rbuf.Len() != 0 {
			n := copy(b, this.rbuf.Bytes())
			this.rbuf.Next(n)
			return n, nil
		}
		return this.Conn.Read(b)
	}
	//剩下的走正常的解碼流程
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
	//如果禁用了編碼, 那麽直接寫數據就好了
	if !this.enableEncode {
		return this.Conn.Write(b)
	}
	//否則的話, 需要進行編碼再寫
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

func CryptWrap(key string, conn net.Conn, coder EnDec) (*MixLayerAdaptor, error) {
	as := &MixLayerAdaptor{Conn: conn, enableEncode: true, enableDecode: true}
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
