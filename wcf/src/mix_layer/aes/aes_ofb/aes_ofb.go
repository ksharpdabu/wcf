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

var keypad = []byte("dfji923ur823hr23iojero2i3hry3dsd")
var ivpad = []byte("3rm983t9j843u843rj23jr4hseuj83ds")

type AesOFB struct {
	key []byte
	iv  []byte
	enc cipher.Stream
	dec cipher.Stream
}

func NewAesOFB(keylen int) *AesOFB {
	ofb := &AesOFB{key: make([]byte, keylen), iv: make([]byte, aes.BlockSize)}
	copy(ofb.key, keypad)
	copy(ofb.iv, ivpad)
	return ofb
}

func (this *AesOFB) Init(key []byte, iv []byte) error {
	copy(this.key, key)
	copy(this.iv, iv)
	key = this.key
	iv = this.iv
	encBlock, err := aes.NewCipher(key)
	if err != nil {
		return errors.New(fmt.Sprintf("create block fail, err:%v", err))
	}
	decBlock, _ := aes.NewCipher(key)
	this.enc = cipher.NewOFB(encBlock, iv)
	this.dec = cipher.NewOFB(decBlock, iv)
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
