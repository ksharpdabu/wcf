package aes_gcm

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"github.com/pkg/errors"
	"mix_layer"
	"mix_layer/aes"
	"net"
)

var nonce = []byte("aefkjglekgk2")
var keypad = []byte("sdjasiku8f2839hy423hddjuaduh2e12")

func init() {
	mix_layer.Regist("aes-256-gcm", func(key string, conn net.Conn) (mix_layer.MixConn, error) {
		return aes_layer.Wrap(key, conn, NewAesGCM(32))
	})
	mix_layer.Regist("aes-192-gcm", func(key string, conn net.Conn) (mix_layer.MixConn, error) {
		return aes_layer.Wrap(key, conn, NewAesGCM(24))
	})
	mix_layer.Regist("aes-128-gcm", func(key string, conn net.Conn) (mix_layer.MixConn, error) {
		return aes_layer.Wrap(key, conn, NewAesGCM(16))
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

func (this *AesGCM) Encode(src []byte) ([]byte, error) {
	return this.encAead.Seal(nil, this.nonce, src, nil), nil
}

func (this *AesGCM) Decode(dst []byte) ([]byte, error) {
	return this.decAead.Open(nil, this.nonce, dst, nil)
}

func (this *AesGCM) Close() error {
	return nil
}
