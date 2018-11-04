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

var keypad = []byte("54ug8hdf3287hrh32r87ywh3ru73rwje")
var ivpad = []byte("sdrj3825uy849hfj9843h523rj3ir3rd")

type AesCTR struct {
	key []byte
	iv  []byte
	enc cipher.Stream
	dec cipher.Stream
}

func NewAesCTR(keylen int) *AesCTR {
	ctr := &AesCTR{key: make([]byte, keylen), iv: make([]byte, aes.BlockSize)}
	copy(ctr.key, keypad)
	copy(ctr.iv, ivpad)
	return ctr
}

func (this *AesCTR) Init(key []byte, iv []byte) error {
	copy(this.key, key)
	copy(this.iv, iv)
	key = this.key
	iv = this.iv
	encBlock, err := aes.NewCipher(key)
	if err != nil {
		return errors.New(fmt.Sprintf("create block fail, err:%v", err))
	}
	decBlock, _ := aes.NewCipher(key)
	this.enc = cipher.NewCTR(encBlock, iv)
	this.dec = cipher.NewCTR(decBlock, iv)
	return nil
}

func (this *AesCTR) Name() string {
	return "aes-ctr"
}

func (this *AesCTR) Encode(input []byte, output []byte) (int, error) {
	if len(output) < len(input) {
		return 0, errors.New(fmt.Sprintf("output buffer too small, input len:%d, output len:%d", len(input), len(output)))
	}
	this.enc.XORKeyStream(output, input)
	return len(input), nil
}

func (this *AesCTR) Decode(input []byte, output []byte) (int, error) {
	if len(output) < len(input) {
		return 0, errors.New(fmt.Sprintf("output buffer too small, input len:%d, output len:%d", len(input), len(output)))
	}
	this.dec.XORKeyStream(output, input)
	return len(input), nil
}
