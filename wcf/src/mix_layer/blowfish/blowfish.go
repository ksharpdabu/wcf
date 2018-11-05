package blowfish

import (
	"errors"
	"fmt"
	"golang.org/x/crypto/blowfish"
	"mix_layer"
	"net"
)

func init() {
	mix_layer.Regist("blowfish", func(key string, conn net.Conn) (mix_layer.MixConn, error) {
		return mix_layer.CryptWrap(key, conn, NewBlowFish())
	})
}

type BlowFish struct {
	enc *blowfish.Cipher
	dec *blowfish.Cipher
}

func NewBlowFish() *BlowFish {
	return &BlowFish{}
}

func (this *BlowFish) IVLen() int {
	return 8
}

func genKey(key []byte, iv []byte) []byte {
	finalKey := make([]byte, len(key)+len(iv))
	copy(finalKey, key)
	copy(finalKey[len(key):], iv)
	return finalKey
}

func (this *BlowFish) InitWrite(key []byte, iv []byte) error {
	enc, err := blowfish.NewCipher(genKey(key, iv))
	if err != nil {
		return errors.New(fmt.Sprintf("create blow fish write coder fail, err:%v", err))
	}
	this.enc = enc
	return nil
}

func (this *BlowFish) InitRead(key []byte, iv []byte) error {
	dec, err := blowfish.NewCipher(genKey(key, iv))
	if err != nil {
		return errors.New(fmt.Sprintf("create blow fish read coder fail, err:%v", err))
	}
	this.dec = dec
	return nil
}

func (this *BlowFish) Name() string {
	return "blowfish"
}

func (this *BlowFish) Encode(input []byte, output []byte) (int, error) {
	in := mix_layer.PKCS5Padding(input, this.enc.BlockSize())
	if len(output) < len(in) {
		return 0, errors.New(fmt.Sprintf("output buffer too small, input len:%d, output len:%d", len(in), len(output)))
	}
	for i := 0; i < len(in); i += this.enc.BlockSize() {
		this.enc.Encrypt(output[i:], in[i:])
	}
	return len(in), nil
}

func (this *BlowFish) Decode(input []byte, output []byte) (int, error) {
	if len(output) < len(input) {
		return 0, errors.New(fmt.Sprintf("output buffer too small, input len:%d, output len:%d", len(input), len(output)))
	}
	if len(input)%this.dec.BlockSize() != 0 {
		return 0, errors.New(fmt.Sprintf("decode data invalid data len:%d, block size:%d", len(input), this.dec.BlockSize()))
	}
	for i := 0; i < len(input); i += this.dec.BlockSize() {
		this.dec.Decrypt(output[i:], input[i:])
	}
	out, err := mix_layer.PKCS5UnPadding(output[:len(input)])
	if err != nil {
		return 0, err
	}
	cnt := copy(output, out)
	return cnt, nil
}
