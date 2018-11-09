package chacha20

import (
	"crypto/cipher"
	"fmt"
	"golang.org/x/crypto/chacha20poly1305"
	"mix_layer"
	"net"
)

func init() {
	mix_layer.Regist("chacha20", func(key string, conn net.Conn) (mix_layer.MixConn, error) {
		return mix_layer.CryptWrap(key, conn, NewChacha20())
	})
}

//XChaCha20-Poly1305
type Chacha20 struct {
	enc    cipher.AEAD
	dec    cipher.AEAD
	rnonce []byte
	wnonce []byte
}

func NewChacha20() *Chacha20 {
	return &Chacha20{}
}

func (this *Chacha20) IVLen() int {
	return chacha20poly1305.NonceSizeX
}

func (this *Chacha20) InitWrite(key []byte, iv []byte) error {
	key = mix_layer.GenKeyWithPad(key, chacha20poly1305.KeySize)
	this.wnonce = mix_layer.GenKeyWithPad(iv, this.IVLen())
	enc, err := chacha20poly1305.NewX(key)
	if err != nil {
		return fmt.Errorf("create enc fail, err:%v", err)
	}
	this.enc = enc
	return nil
}

func (this *Chacha20) InitRead(key []byte, iv []byte) error {
	key = mix_layer.GenKeyWithPad(key, chacha20poly1305.KeySize)
	this.rnonce = mix_layer.GenKeyWithPad(iv, this.IVLen())
	dec, err := chacha20poly1305.NewX(key)
	if err != nil {
		return fmt.Errorf("create dec fail, err:%v", err)
	}
	this.dec = dec
	return nil
}

func (this *Chacha20) Name() string {
	return "chacha20"
}

func (this *Chacha20) Encode(input []byte, output []byte) (int, error) {
	out := this.enc.Seal(nil, this.wnonce, input, nil)
	if len(out) > len(output) {
		return 0, fmt.Errorf("output too small, skip, raw:%d, output buffer:%d", len(out), len(output))
	}
	return copy(output, out), nil
}

func (this *Chacha20) Decode(input []byte, output []byte) (int, error) {
	out, err := this.dec.Open(nil, this.rnonce, input, nil)
	if err != nil {
		return 0, fmt.Errorf("decode fail, err:%v", err)
	}
	if len(out) > len(output) {
		return 0, fmt.Errorf("output too small, skip, raw:%d, output buffer:%d", len(out), len(output))
	}
	return copy(output, out), nil
}
