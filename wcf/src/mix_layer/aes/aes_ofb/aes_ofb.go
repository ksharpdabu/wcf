package aes_ofb

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"mix_layer"
	"net"
)

func init() {
	mix_layer.Regist("aes-256-ofb", func(key string, conn net.Conn) (mix_layer.MixConn, error) {
		return mix_layer.CryptWrap(key, conn, NewAesOFB(32))
	})
	mix_layer.Regist("aes-192-ofb", func(key string, conn net.Conn) (mix_layer.MixConn, error) {
		return mix_layer.CryptWrap(key, conn, NewAesOFB(24))
	})
	mix_layer.Regist("aes-128-ofb", func(key string, conn net.Conn) (mix_layer.MixConn, error) {
		return mix_layer.CryptWrap(key, conn, NewAesOFB(16))
	})
}

type AesOFB struct {
	enc    cipher.Stream
	dec    cipher.Stream
	keylen int
}

func NewAesOFB(keylen int) *AesOFB {
	ofb := &AesOFB{}
	ofb.keylen = keylen
	return ofb
}

func (this *AesOFB) IVLen() int {
	return aes.BlockSize
}

func (this *AesOFB) InitWrite(key []byte, iv []byte) error {
	key = mix_layer.GenKeyWithPad(key, this.keylen)
	iv = mix_layer.GenKeyWithPad(iv, this.IVLen())
	enc, err := aes.NewCipher(key)
	if err != nil {
		return errors.New(fmt.Sprintf("create write block fail, err:%v", err))
	}
	this.enc = cipher.NewOFB(enc, iv)
	if this.enc == nil {
		return errors.New(fmt.Sprintf("create write ofb fail, iv len:%d", len(iv)))
	}
	return nil
}

func (this *AesOFB) InitRead(key []byte, iv []byte) error {
	key = mix_layer.GenKeyWithPad(key, this.keylen)
	iv = mix_layer.GenKeyWithPad(iv, this.IVLen())
	dec, err := aes.NewCipher(key)
	if err != nil {
		return errors.New(fmt.Sprintf("create read block fail, err:%v", err))
	}
	this.dec = cipher.NewOFB(dec, iv)
	if this.dec == nil {
		return errors.New(fmt.Sprintf("create read ofb fail, iv len:%d", len(iv)))
	}
	return nil
}

func (this *AesOFB) Name() string {
	return "aes-ofb"
}

func (this *AesOFB) Encode(input []byte, output []byte) (int, error) {
	if len(output) < len(input) {
		return 0, errors.New(fmt.Sprintf("output buffer too small, input len:%d, output len:%d", len(input), len(output)))
	}
	this.enc.XORKeyStream(output, input)
	return len(input), nil
}

func (this *AesOFB) Decode(input []byte, output []byte) (int, error) {
	if len(output) < len(input) {
		return 0, errors.New(fmt.Sprintf("output buffer too small, input len:%d, output len:%d", len(input), len(output)))
	}
	this.dec.XORKeyStream(output, input)
	return len(input), nil
}
