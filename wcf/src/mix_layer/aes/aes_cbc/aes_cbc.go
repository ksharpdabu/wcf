package aes_cbc

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"mix_layer"
	"net"
)

func init() {
	mix_layer.Regist("aes-256-cbc", func(key string, conn net.Conn) (mix_layer.MixConn, error) {
		return mix_layer.CryptWrap(key, conn, NewAesCBC(32))
	})
	mix_layer.Regist("aes-192-cbc", func(key string, conn net.Conn) (mix_layer.MixConn, error) {
		return mix_layer.CryptWrap(key, conn, NewAesCBC(24))
	})
	mix_layer.Regist("aes-128-cbc", func(key string, conn net.Conn) (mix_layer.MixConn, error) {
		return mix_layer.CryptWrap(key, conn, NewAesCBC(16))
	})
}

var keypad = []byte("3rj293hru2i3hr4g32r98fhu7324rf46")
var ivpad = []byte("fejnnu23h4g3r2n4rsah39r5j21h9r0-")

type AesCBC struct {
	key []byte
	iv  []byte
	enc cipher.BlockMode
	dec cipher.BlockMode
}

func NewAesCBC(keylen int) *AesCBC {
	ofb := &AesCBC{key: make([]byte, keylen), iv: make([]byte, aes.BlockSize)}
	copy(ofb.key, keypad)
	copy(ofb.iv, ivpad)
	return ofb
}

func (this *AesCBC) Init(key []byte, iv []byte) error {
	copy(this.key, key)
	copy(this.iv, iv)
	key = this.key
	iv = this.iv
	encBlock, err := aes.NewCipher(key)
	if err != nil {
		return errors.New(fmt.Sprintf("create block fail, err:%v", err))
	}
	decBlock, _ := aes.NewCipher(key)
	this.enc = cipher.NewCBCEncrypter(encBlock, iv)
	this.dec = cipher.NewCBCDecrypter(decBlock, iv)
	return nil
}

func (this *AesCBC) Name() string {
	return "aes-cbc"
}

func PKCS5Padding(src []byte, blockSize int) []byte {
	padding := blockSize - len(src)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	totalData := make([]byte, len(src)+len(padtext))
	copy(totalData, src)
	copy(totalData[len(src):], padtext)
	return totalData
}

func PKCS5UnPadding(src []byte) []byte {
	length := len(src)
	unpadding := int(src[length-1])
	return src[:(length - unpadding)]
}

func (this *AesCBC) Encode(input []byte, output []byte) (int, error) {
	in := PKCS5Padding(input, this.enc.BlockSize())
	this.enc.CryptBlocks(output, in)
	return len(in), nil
}

func (this *AesCBC) Decode(input []byte, output []byte) (int, error) {
	this.dec.CryptBlocks(output, input)
	out := PKCS5UnPadding(output[:len(input)])
	cnt := copy(output, out)
	return cnt, nil
}
