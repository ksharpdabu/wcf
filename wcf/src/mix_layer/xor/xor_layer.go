package xor

import (
	"net"
	"mix_layer"
	"crypto/sha1"
	"net_utils"
)

func init() {
	mix_layer.Regist("xor", func(key string, conn net.Conn) (mix_layer.MixConn, error) {
		return Wrap(key, conn)
	})
}

type Xor struct {
	net.Conn
	key string
	kIndex int
}

func(this *Xor) SetKey(key string) {
	v := sha1.Sum([]byte(key))
	this.key = string(v[:])
	this.kIndex = len(key) * 13 % len(this.key)
}

func(this *Xor) xor(b []byte) {
	for i := 0; i < len(b); i++ {
		b[i] = b[i] ^ this.key[this.kIndex % len(this.key)]
		//b[i] = b[i] ^ 0xff
		this.kIndex++
	}
}

func(this *Xor) Read(b []byte) (n int, err error) {
	cnt, err := this.Conn.Read(b)
	if err != nil {
		return cnt, err
	}
	this.xor(b[:cnt])
	return cnt, err
}

func(this *Xor) Write(b []byte) (int, error) {
	bf := make([]byte, len(b))
	copy(bf, b)
	this.xor(bf)
	err := net_utils.SendSpecLen(this.Conn, bf)
	if err != nil {
		return 0, err
	}
	return len(b), nil
}

func Wrap(key string, conn net.Conn) (*Xor, error) {
	xor := &Xor{Conn:conn}
	xor.SetKey(key)
	return xor, nil
}