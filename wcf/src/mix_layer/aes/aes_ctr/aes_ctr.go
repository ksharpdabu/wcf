package aes_ctr

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"mix_layer"
	"net"
)

func init() {
	mix_layer.Regist("aes-256-ctr", func(key string, conn net.Conn) (mix_layer.MixConn, error) {
		return mix_layer.CryptWrap(key, conn, NewAesCTR(32))
	})
	mix_layer.Regist("aes-192-ctr", func(key string, conn net.Conn) (mix_layer.MixConn, error) {
		return mix_layer.CryptWrap(key, conn, NewAesCTR(24))
	})
	mix_layer.Regist("aes-128-ctr", func(key string, conn net.Conn) (mix_layer.MixConn, error) {
		return mix_layer.CryptWrap(key, conn, NewAesCTR(16))
	})
}

type AesCTR struct {
	enc    cipher.Stream
	dec    cipher.Stream
	keylen int
}

func NewAesCTR(keylen int) *AesCTR {
	ctr := &AesCTR{}
	ctr.keylen = keylen
	return ctr
}

func (this *AesCTR) IVLen() int {
	return aes.BlockSize
}

func (this *AesCTR) InitWrite(key []byte, iv []byte) error {
	key = mix_layer.GenKeyWithPad(key, this.keylen)
	iv = mix_layer.GenKeyWithPad(key, this.IVLen())
	encBlock, err := aes.NewCipher(key)
	if err != nil {
		return errors.New(fmt.Sprintf("create enc block fail, err:%v", err))
	}
	this.enc = cipher.NewCTR(encBlock, iv)
	if this.enc == nil {
		return errors.New(fmt.Sprintf("create enc stream fail, iv len:%d", len(iv)))
	}
	return nil
}

func (this *AesCTR) InitRead(key []byte, iv []byte) error {
	key = mix_layer.GenKeyWithPad(key, this.keylen)
	iv = mix_layer.GenKeyWithPad(key, this.IVLen())
	decBlock, err := aes.NewCipher(key)
	if err != nil {
		return errors.New(fmt.Sprintf("create dec block fail, err:%v", err))
	}
	this.dec = cipher.NewCTR(decBlock, iv)
	if this.dec == nil {
		return errors.New(fmt.Sprintf("create dec stream fail, iv len:%d", len(iv)))
	}
	return nil
}

func (this *AesCTR) Name() string {
	return "aes-ctr"
}

func (this *AesCTR) Encode(input []byte) ([]byte, error) {
	output := make([]byte, len(input))
	this.enc.XORKeyStream(output, input)
	return output, nil
}

func (this *AesCTR) Decode(input []byte) ([]byte, error) {
	output := make([]byte, len(input))
	this.dec.XORKeyStream(output, input)
	return output, nil
}
