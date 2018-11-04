package mix_delegate

import (
	"mix_layer"
	_ "mix_layer/aes/aes_cfb"
	_ "mix_layer/aes/aes_gcm"
	_ "mix_layer/aes/aes_ofb"
	_ "mix_layer/comp"
	_ "mix_layer/xor"
	"net"
)

func Wrap(name string, key string, conn net.Conn) (mix_layer.MixConn, error) {
	return mix_layer.Wrap(name, key, conn)
}
