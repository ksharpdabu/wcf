package aes_cbc

import (
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

type AesCBC struct {
	enc    cipher.BlockMode
	dec    cipher.BlockMode
	keylen int
}

func NewAesCBC(keylen int) *AesCBC {
	cbc := &AesCBC{}
	cbc.keylen = keylen
	return cbc
}

func (this *AesCBC) IVLen() int {
	return aes.BlockSize
}

func (this *AesCBC) InitRead(key []byte, iv []byte) error {
	key = mix_layer.GenKeyWithPad(key, this.keylen)
	iv = mix_layer.GenKeyWithPad(iv, this.IVLen())
	decBlock, err := aes.NewCipher(key)
	if err != nil {
		return errors.New(fmt.Sprintf("create enc block fail, err:%v", err))
	}
	this.dec = cipher.NewCBCDecrypter(decBlock, iv)
	return nil
}

func (this *AesCBC) InitWrite(key []byte, iv []byte) error {
	key = mix_layer.GenKeyWithPad(key, this.keylen)
	iv = mix_layer.GenKeyWithPad(iv, this.IVLen())
	encBlock, err := aes.NewCipher(key)
	if err != nil {
		return errors.New(fmt.Sprintf("create enc block fail, err:%v", err))
	}
	this.enc = cipher.NewCBCEncrypter(encBlock, iv)
	return nil
}

func (this *AesCBC) Name() string {
	return "aes-cbc"
}

func (this *AesCBC) Encode(input []byte) ([]byte, error) {
	in := mix_layer.PKCS5Padding(input, this.enc.BlockSize())
	output := make([]byte, len(in))
	this.enc.CryptBlocks(output, in)
	return output, nil
}

func (this *AesCBC) Decode(input []byte) ([]byte, error) {
	if len(input)%this.dec.BlockSize() != 0 {
		return nil, errors.New(fmt.Sprintf("decode data invalid data len:%d, block size:%d", len(input), this.dec.BlockSize()))
	}
	output := make([]byte, len(input))
	this.dec.CryptBlocks(output, input)
	out, err := mix_layer.PKCS5UnPadding(output[:len(input)])
	if err != nil {
		return nil, err
	}
	return out, nil
}
