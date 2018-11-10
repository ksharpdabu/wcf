package xor

import (
	"crypto/sha1"
	"mix_layer"
	"net"
)

func init() {
	mix_layer.Regist("xor", func(key string, conn net.Conn) (mix_layer.MixConn, error) {
		return mix_layer.CryptWrap(key, conn, NewXor())
	})
}

type Xor struct {
	wKey   []byte
	rKey   []byte
	wIndex int
	rIndex int
}

func NewXor() *Xor {
	return &Xor{}
}

func (this *Xor) IVLen() int {
	return 13
}

func genKey(key []byte, iv []byte) []byte {
	newKey := make([]byte, len(key)+len(iv))
	copy(newKey, key)
	copy(newKey[len(key):], iv)
	shaSum := sha1.Sum(newKey)
	return shaSum[:]
}

func (this *Xor) InitWrite(key []byte, iv []byte) error {
	this.wKey = genKey(key, iv)
	this.wIndex = 0
	return nil
}

func (this *Xor) InitRead(key []byte, iv []byte) error {
	this.rKey = genKey(key, iv)
	this.rIndex = 0
	return nil
}

func (this *Xor) Name() string {
	return "xor"
}

func xor(in []byte, out []byte, key []byte, loc int) int {
	for i := 0; i < len(in); i++ {
		out[i] = in[i] ^ key[loc%len(key)]
		loc++
	}
	if loc > 100 {
		loc = 0
	}
	return loc
}

func (this *Xor) Encode(input []byte) ([]byte, error) {
	output := make([]byte, len(input))
	this.wIndex = xor(input, output, this.wKey, this.wIndex)
	return output, nil
}

func (this *Xor) Decode(input []byte) ([]byte, error) {
	output := make([]byte, len(input))
	this.rIndex = xor(input, output, this.rKey, this.rIndex)
	return output, nil
}
