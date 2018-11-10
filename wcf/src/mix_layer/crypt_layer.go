package mix_layer

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
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
	rtmp         []byte
}

const MAX_CRYPT_PACKET_LEN = 32 * 1024
const HMAC_LENGTH = 20

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
	index := 0
	var err error
	var ivout []byte
	//首次读需要获取到iv并初始化解码器
	if !this.initRead {
		ivout = make([]byte, this.coder.IVLen())
	}
	for {
		index, err = this.Conn.Read(this.rtmp)
		if err != nil {
			return 0, err
		}
		this.rbuf.Write(this.rtmp[:index])
		cnt, err := CheckHeadFrame(this.rbuf.Bytes(), len(ivout), MAX_CRYPT_PACKET_LEN)
		if cnt != 0 || err != nil {
			break
		}
	}
	for {
		frameLen, err := CheckHeadFrame(this.rbuf.Bytes(), len(ivout), MAX_CRYPT_PACKET_LEN)
		if err != nil || frameLen < 0 {
			return 0, fmt.Errorf("check frame data fail, err:%v, cnt:%d", err, frameLen)
		}
		if frameLen == 0 {
			break
		}
		encRaw, err := DecodeHeadFrame(this.rbuf.Bytes()[:frameLen], ivout, this.key)
		if err != nil {
			return 0, fmt.Errorf("decode head frame data fail, err:%v", err)
		}
		if !this.initRead {
			ierr := this.coder.InitRead(this.key, ivout)
			if err != nil {
				return 0, fmt.Errorf("init read coder fail, err:%v, key hex:%s, iv hex:%s",
					ierr, hex.EncodeToString(this.key), hex.EncodeToString(ivout))
			}
			this.initRead = true
			ivout = nil
		}
		this.rbuf.Next(frameLen)
		rawWrite, err := this.coder.Decode(encRaw)
		if err != nil {
			return 0, fmt.Errorf("decode aes data fail, err:%v, data len:%d, coder:%s", err, len(encRaw), this.coder.Name())
		}
		this.decbuf.Write(rawWrite)
	}
	if this.decbuf.Len() <= 0 {
		return 0, fmt.Errorf("no more data, may has err")
	}
	cnt := copy(b, this.decbuf.Bytes())
	this.decbuf.Next(cnt)
	return cnt, nil
}

func (this *MixLayerAdaptor) Write(b []byte) (int, error) {
	if len(b) > 4*MAX_CRYPT_PACKET_LEN/5 {
		b = b[:4*MAX_CRYPT_PACKET_LEN/5]
	}
	//如果禁用了編碼, 那麽直接寫數據就好了
	if !this.enableEncode {
		return this.Conn.Write(b)
	}
	//否則的話, 需要進行編碼再寫
	encData, err := this.coder.Encode(b)
	if err != nil {
		return 0, err
	}
	frameData, err := EncodeHeadFrame(encData, this.ivKey, this.key)
	if this.ivKey != nil {
		this.ivKey = nil
	}
	if err != nil {
		return 0, err
	}
	if err = net_utils.SendSpecLen(this.Conn, frameData); err != nil {
		return 0, err
	}
	return len(b), nil
}

func CryptWrap(key string, conn net.Conn, coder EnDec) (*MixLayerAdaptor, error) {
	as := &MixLayerAdaptor{Conn: conn,
		enableEncode: true, enableDecode: true,
		rtmp: make([]byte, MAX_CRYPT_PACKET_LEN),
	}
	as.SetKey(key)
	as.coder = coder
	as.ivKey = RandIV(coder.IVLen())
	err := as.coder.InitWrite(as.key, as.ivKey)
	if err != nil {
		return nil, fmt.Errorf("init write fail, err:%v, key hex:%s, iv hex:%s",
			err, hex.EncodeToString(as.key), hex.EncodeToString(as.ivKey))
	}
	return as, nil
}
