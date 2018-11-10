package salsa20

import (
	"crypto/cipher"
	"golang.org/x/crypto/salsa20"
	"mix_layer"
	"net"
)

func init() {
	mix_layer.Regist("salsa20", func(key string, conn net.Conn) (mix_layer.MixConn, error) {
		return mix_layer.CryptWrap(key, conn, NewSalsa20())
	})
}

const SALSA20_IVLEN = 24
const SALSA20_KEYLEN = 32

type Salsa20 struct {
	enc    cipher.AEAD
	dec    cipher.AEAD
	rkey   [32]byte
	wkey   [32]byte
	rnonce []byte
	wnonce []byte
}

func NewSalsa20() *Salsa20 {
	return &Salsa20{}
}

func (this *Salsa20) IVLen() int {
	return SALSA20_IVLEN
}

func (this *Salsa20) InitWrite(key []byte, iv []byte) error {
	key = mix_layer.GenKeyWithPad(key, SALSA20_KEYLEN)
	copy(this.wkey[:], key)
	this.wnonce = mix_layer.GenKeyWithPad(iv, this.IVLen())
	return nil
}

func (this *Salsa20) InitRead(key []byte, iv []byte) error {
	key = mix_layer.GenKeyWithPad(key, SALSA20_KEYLEN)
	copy(this.rkey[:], key)
	this.rnonce = mix_layer.GenKeyWithPad(iv, this.IVLen())
	return nil
}

func (this *Salsa20) Name() string {
	return "salsa20"
}

func (this *Salsa20) Encode(input []byte) ([]byte, error) {
	output := make([]byte, len(input))
	salsa20.XORKeyStream(output, input, this.wnonce, &this.wkey)
	return output, nil
}

func (this *Salsa20) Decode(input []byte) ([]byte, error) {
	output := make([]byte, len(input))
	salsa20.XORKeyStream(output, input, this.rnonce, &this.rkey)
	return output, nil
}
