package aes_cfb

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"mix_layer"
	"net"
)

func init() {
	mix_layer.Regist("aes-256-cfb", func(key string, conn net.Conn) (mix_layer.MixConn, error) {
		return mix_layer.CryptWrap(key, conn, NewAesCFB(32))
	})
	mix_layer.Regist("aes-192-cfb", func(key string, conn net.Conn) (mix_layer.MixConn, error) {
		return mix_layer.CryptWrap(key, conn, NewAesCFB(24))
	})
	mix_layer.Regist("aes-128-cfb", func(key string, conn net.Conn) (mix_layer.MixConn, error) {
		return mix_layer.CryptWrap(key, conn, NewAesCFB(16))
	})
}

type AesCFB struct {
	enc    cipher.Stream
	dec    cipher.Stream
	keylen int
}

func NewAesCFB(keylen int) *AesCFB {
	cfb := &AesCFB{}
	cfb.keylen = keylen
	return cfb
}

func (this *AesCFB) IVLen() int {
	return aes.BlockSize
}

func (this *AesCFB) InitRead(key []byte, iv []byte) error {
	key = mix_layer.GenKeyWithPad(key, this.keylen)
	iv = mix_layer.GenKeyWithPad(iv, this.IVLen())
	decBlock, err := aes.NewCipher(key)
	if err != nil {
		return errors.New(fmt.Sprintf("create dec block fail, err:%v", err))
	}
	this.dec = cipher.NewCFBDecrypter(decBlock, iv)
	return nil
}

func (this *AesCFB) InitWrite(key []byte, iv []byte) error {
	key = mix_layer.GenKeyWithPad(key, this.keylen)
	iv = mix_layer.GenKeyWithPad(iv, this.IVLen())
	encBlock, err := aes.NewCipher(key)
	if err != nil {
		return errors.New(fmt.Sprintf("create enc block fail, err:%v", err))
	}
	this.enc = cipher.NewCFBEncrypter(encBlock, iv)
	return nil
}

func (this *AesCFB) Name() string {
	return "aes-cfb"
}

func (this *AesCFB) Encode(input []byte, output []byte) (int, error) {
	if len(output) < len(input) {
		return 0, errors.New(fmt.Sprintf("output buffer too small, input len:%d, output len:%d", len(input), len(output)))
	}
	this.enc.XORKeyStream(output, input)
	return len(input), nil
}

func (this *AesCFB) Decode(input []byte, output []byte) (int, error) {
	if len(output) < len(input) {
		return 0, errors.New(fmt.Sprintf("output buffer too small, input len:%d, output len:%d", len(input), len(output)))
	}
	this.dec.XORKeyStream(output, input)
	return len(input), nil
}
