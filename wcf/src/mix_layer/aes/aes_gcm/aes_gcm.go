package aes_gcm

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"mix_layer"
	"net"
)

var nonce = []byte("aefkjglekgk2")
var keypad = []byte("sdjasiku8f2839hy423hddjuaduh2e12")

func init() {
	mix_layer.Regist("aes-256-gcm", func(key string, conn net.Conn) (mix_layer.MixConn, error) {
		return mix_layer.CryptWrap(key, conn, NewAesGCM(32))
	})
	mix_layer.Regist("aes-192-gcm", func(key string, conn net.Conn) (mix_layer.MixConn, error) {
		return mix_layer.CryptWrap(key, conn, NewAesGCM(24))
	})
	mix_layer.Regist("aes-128-gcm", func(key string, conn net.Conn) (mix_layer.MixConn, error) {
		return mix_layer.CryptWrap(key, conn, NewAesGCM(16))
	})
}

type AesGCM struct {
	encAead cipher.AEAD
	decAead cipher.AEAD
	nonce   []byte
	key     []byte
}

func NewAesGCM(keylen int) *AesGCM {
	gcm := &AesGCM{key: make([]byte, keylen)}
	copy(gcm.key, keypad)
	return gcm
}

func (this *AesGCM) newAead(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return aead, nil
}

func (this *AesGCM) Init(key []byte, iv []byte) error {
	copy(this.key, key)
	key = this.key
	if enc, err := this.newAead(key); err != nil {
		return errors.New(fmt.Sprintf("create enc aead fail, err:%v", err))
	} else {
		this.encAead = enc
	}
	if dec, err := this.newAead(key); err != nil {
		return errors.New(fmt.Sprintf("create dec aead fail, err:%v", err))
	} else {
		this.decAead = dec
	}
	this.nonce = nonce
	return nil
}

func (this *AesGCM) Name() string {
	return "aes-gcm"
}

func (this *AesGCM) Encode(input []byte, output []byte) (int, error) {
	out := this.encAead.Seal(nil, this.nonce, input, nil)
	if len(out) > len(output) {
		return 0, errors.New(fmt.Sprintf("buffer too small, need:%d, output:%d, input:%d", len(out), len(output), len(input)))
	}
	cnt := copy(output, out)
	return cnt, nil
}

func (this *AesGCM) Decode(input []byte, output []byte) (int, error) {
	out, err := this.decAead.Open(nil, this.nonce, input, nil)
	if err != nil {
		return 0, errors.New(fmt.Sprintf("decode fail, err:%v, in len:%d, out buffer len:%d", err, len(input), len(output)))
	}
	if len(out) > len(output) {
		return 0, errors.New(fmt.Sprintf("buffer too small, need:%d, output:%d, input:%d", len(out), len(output), len(input)))
	}
	cnt := copy(output, out)
	return cnt, nil
}
