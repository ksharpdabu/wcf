package aes_layer

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	"net"
	"net_utils"
)

type Aes struct {
	net.Conn
	key    []byte
	ivKey  []byte
	coder  EnDec
	rbuf   bytes.Buffer
	decbuf bytes.Buffer
}

const MAX_AES_PACKET_LEN = 64 * 1024

func EncodeHeadFrame(src []byte, dst []byte) (int, error) {
	if len(dst) < len(src)+4 {
		return 0, errors.New(fmt.Sprintf("buffer too small, src len:%d, buf len:%d", len(src), len(dst)))
	}
	binary.BigEndian.PutUint32(dst, uint32(len(src)+4))
	copy(dst[4:], src)
	return 4 + len(src), nil
}

func CheckHeadFrame(src []byte, maxData int) (int, error) {
	if len(src) <= 4 {
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

func DecodeHeadFrame(src []byte, dst []byte) (int, error) {
	sz, err := CheckHeadFrame(src, 64*1024)
	if sz <= 0 {
		return 0, errors.New(fmt.Sprintf("package check fail, sz:%d, err:%v", sz, err))
	}
	return copy(dst, src[4:sz]), nil
}

func (this *Aes) SetKey(key string) {
	iv := md5.Sum([]byte(key))
	k := sha1.Sum([]byte(key))
	this.ivKey = []byte(iv[:])
	this.key = []byte(k[:])
}

func (this *Aes) encrypt(src []byte) ([]byte, error) {
	return this.coder.Encode(src)
}

func (this *Aes) decrypt(dst []byte) ([]byte, error) {
	return this.coder.Decode(dst)
}

func (this *Aes) Read(b []byte) (int, error) {
	if this.decbuf.Len() != 0 {
		n := copy(b, this.decbuf.Bytes())
		this.decbuf.Next(n)
		return n, nil
	}
	tmp := make([]byte, 16*1024)
	index := 0
	var err error
	for {
		index, err = this.Conn.Read(tmp)
		if err != nil {
			return 0, err
		}
		this.rbuf.Write(tmp[:index])
		cnt, err := CheckHeadFrame(this.rbuf.Bytes(), MAX_AES_PACKET_LEN)
		if cnt != 0 || err != nil {
			break
		}
	}
	buf := make([]byte, MAX_AES_PACKET_LEN)
	for {
		cnt, err := CheckHeadFrame(this.rbuf.Bytes(), MAX_AES_PACKET_LEN)
		if err != nil || cnt < 0 {
			return 0, errors.New(fmt.Sprintf("check frame data fail, err:%v, cnt:%d", err, cnt))
		}
		if cnt == 0 {
			break
		}
		total, err := DecodeHeadFrame(this.rbuf.Bytes()[:cnt], buf)
		if err != nil {
			return 0, errors.New(fmt.Sprintf("decode head frame data fail, err:%v", err))
		}
		this.rbuf.Next(cnt)
		raw, err := this.decrypt(buf[:total])
		if err != nil {
			return 0, errors.New(fmt.Sprintf("decode aes data fail, err:%v, data len:%d, coder:%s", err, total, this.coder.Name()))
		}
		this.decbuf.Write(raw)
	}
	if this.decbuf.Len() <= 0 {
		return 0, errors.New(fmt.Sprintf("no more data, may has err"))
	}
	cnt := copy(b, this.decbuf.Bytes())
	this.decbuf.Next(cnt)
	return cnt, nil
}

func (this *Aes) Write(b []byte) (int, error) {
	if len(b) >= 1*MAX_AES_PACKET_LEN/3 {
		b = b[:1*MAX_AES_PACKET_LEN/3]
	}
	enc, err := this.coder.Encode(b)
	if err != nil {
		return 0, err
	}
	buf := make([]byte, MAX_AES_PACKET_LEN)
	cnt, err := EncodeHeadFrame(enc, buf)
	if err != nil {
		return 0, err
	}
	if cnt >= MAX_AES_PACKET_LEN {
		return 0, errors.New(fmt.Sprintf("encode packet too long, skip, cnt:%d", cnt))
	}
	if err = net_utils.SendSpecLen(this.Conn, buf[:cnt]); err != nil {
		return 0, err
	}
	return len(b), nil
}

func Wrap(key string, conn net.Conn, coder EnDec) (*Aes, error) {
	as := &Aes{Conn: conn}
	as.SetKey(key)
	as.coder = coder
	err := as.coder.Init(as.key, as.ivKey)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("init coder fail, name:%s, err:%v", coder.Name(), err))
	}
	return as, nil
}