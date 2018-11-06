package none

import (
	"errors"
	"fmt"
	"mix_layer"
	"net"
)

func init() {
	mix_layer.Regist("none", func(key string, conn net.Conn) (mix_layer.MixConn, error) {
		return mix_layer.CryptWrap(key, conn, NewNone())
	})
	mix_layer.Regist("", func(key string, conn net.Conn) (mix_layer.MixConn, error) {
		return mix_layer.CryptWrap(key, conn, NewNone())
	})
}

type None struct {
}

func NewNone() *None {
	return &None{}
}

func (this *None) IVLen() int {
	return 19
}

func (this *None) InitWrite(key []byte, iv []byte) error {
	return nil
}

func (this *None) InitRead(key []byte, iv []byte) error {
	return nil
}

func (this *None) Name() string {
	return "none"
}

//多了一遍複製, 但是整體上統一了。。
func (this *None) Encode(input []byte, output []byte) (int, error) {
	if len(output) < len(input) {
		return 0, errors.New(fmt.Sprintf("output buffer too small, acquire:%d, get:%d", len(input), len(output)))
	}
	return copy(output, input), nil
}

func (this *None) Decode(input []byte, output []byte) (int, error) {
	return this.Encode(input, output)
}
