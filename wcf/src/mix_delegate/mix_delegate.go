package mix_delegate

import (
	"mix_layer"
	_ "mix_layer/xor"
	_ "mix_layer/comp"
	"net"
)

func Wrap(name string, key string, conn net.Conn) (mix_layer.MixConn, error) {
	return mix_layer.Wrap(name, key, conn)
}