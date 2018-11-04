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

var keypad = []byte("j823j42384uj23jtekjinr8235n34243")
var ivpad = []byte("4835y372383768438248y473483748f3")

type AesCFB struct {
	key []byte
	iv  []byte
	enc cipher.Stream
	dec cipher.Stream
}

func NewAesCFB(keylen int) *AesCFB {
	cfb := &AesCFB{key: make([]byte, keylen), iv: make([]byte, aes.BlockSize)}
	copy(cfb.key, keypad)
	copy(cfb.iv, ivpad)
	return cfb
}

func (this *AesCFB) Init(key []byte, iv []byte) error {
	copy(this.key, key)
	copy(this.iv, iv)
	key = this.key
	iv = this.iv
	encBlock, err := aes.NewCipher(key)
	if err != nil {
		return errors.New(fmt.Sprintf("create block fail, err:%v", err))
	}
	decBlock, _ := aes.NewCipher(key)
	this.enc = cipher.NewCFBEncrypter(encBlock, iv)
	this.dec = cipher.NewCFBDecrypter(decBlock, iv)
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
