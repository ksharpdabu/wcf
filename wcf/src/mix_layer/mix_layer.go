package mix_layer

import (
	"errors"
	"fmt"
	"net"
)

type MixLayer struct {
	mp map[string]WrapFunc
}

type WrapFunc func(key string, conn net.Conn) (MixConn, error)

type MixConn interface {
	net.Conn
	SetKey(string)
	DisableEncode()
	DisableDecode()
}

var layer *MixLayer

func init() {
	layer = &MixLayer{}
	layer.mp = make(map[string]WrapFunc)
}

func CheckMixName(name string) bool {
	_, ok := layer.mp[name]
	return ok
}

func GetAllMixName() []string {
	var result []string
	for k, _ := range layer.mp {
		result = append(result, k)
	}
	return result
}

func Regist(name string, fun WrapFunc) error {
	if _, ok := layer.mp[name]; ok {
		return errors.New(fmt.Sprintf("name:%s conn already exists", name))
	}
	layer.mp[name] = fun
	return nil
}

func Wrap(name string, key string, conn net.Conn) (MixConn, error) {
	if v, ok := layer.mp[name]; ok {
		return v(key, conn)
	}
	return nil, errors.New(fmt.Sprintf("mix name:%s not regist", name))
}
