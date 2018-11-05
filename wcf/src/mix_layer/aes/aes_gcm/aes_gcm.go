package aes_gcm

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"mix_layer"
	"net"
)

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
	wnonce  []byte
	rnonce  []byte
	keylen  int
}

func NewAesGCM(keylen int) *AesGCM {
	gcm := &AesGCM{}
	gcm.keylen = keylen
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

func (this *AesGCM) IVLen() int {
	return 12
}

func (this *AesGCM) InitWrite(key []byte, iv []byte) error {
	key = mix_layer.GenKeyWithPad(key, this.keylen)
	iv = mix_layer.GenKeyWithPad(key, this.IVLen())
	if enc, err := this.newAead(key); err != nil {
		return errors.New(fmt.Sprintf("create enc aead fail, err:%v", err))
	} else {
		this.encAead = enc
	}
	this.wnonce = iv
	return nil
}

func (this *AesGCM) InitRead(key []byte, iv []byte) error {
	key = mix_layer.GenKeyWithPad(key, this.keylen)
	iv = mix_layer.GenKeyWithPad(key, this.IVLen())
	if dec, err := this.newAead(key); err != nil {
		return errors.New(fmt.Sprintf("create dec aead fail, err:%v", err))
	} else {
		this.decAead = dec
	}
	this.rnonce = iv
	return nil
}

func (this *AesGCM) Name() string {
	return "aes-gcm"
}

func (this *AesGCM) Encode(input []byte, output []byte) (int, error) {
	out := this.encAead.Seal(nil, this.wnonce, input, nil)
	if len(out) > len(output) {
		return 0, errors.New(fmt.Sprintf("buffer too small, need:%d, output:%d, input:%d", len(out), len(output), len(input)))
	}
	cnt := copy(output, out)
	return cnt, nil
}

func (this *AesGCM) Decode(input []byte, output []byte) (int, error) {
	out, err := this.decAead.Open(nil, this.rnonce, input, nil)
	if err != nil {
		return 0, errors.New(fmt.Sprintf("decode fail, err:%v, in len:%d, out buffer len:%d", err, len(input), len(output)))
	}
	if len(out) > len(output) {
		return 0, errors.New(fmt.Sprintf("buffer too small, need:%d, output:%d, input:%d", len(out), len(output), len(input)))
	}
	cnt := copy(output, out)
	return cnt, nil
}
