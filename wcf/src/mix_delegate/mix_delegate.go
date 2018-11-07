package mix_delegate

import (
	"mix_layer"
	_ "mix_layer/aes/aes_cbc"
	_ "mix_layer/aes/aes_cfb"
	_ "mix_layer/aes/aes_ctr"
	_ "mix_layer/aes/aes_gcm"
	_ "mix_layer/aes/aes_ofb"
	_ "mix_layer/blowfish"
	_ "mix_layer/none"
	_ "mix_layer/xor"
	"net"
)

func Wrap(name string, key string, conn net.Conn) (mix_layer.MixConn, error) {
	return mix_layer.Wrap(name, key, conn)
}

func GetAllMixName() []string {
	return mix_layer.GetAllMixName()
}

func CheckMixName(name string) bool {
	return mix_layer.CheckMixName(name)
}
