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
}

var layer *MixLayer

func none(key string, conn net.Conn) (MixConn, error) {
	cn := &DefaultMixConn{conn}
	cn.SetKey(key)
	return cn, nil
}

func init() {
	layer = &MixLayer{}
	layer.mp = make(map[string]WrapFunc)
	Regist("none", none)
	Regist("", none)
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

type DefaultMixConn struct {
	net.Conn
}

func (this *DefaultMixConn) SetKey(string) {

}

func Wrap(name string, key string, conn net.Conn) (MixConn, error) {
	if v, ok := layer.mp[name]; ok {
		return v(key, conn)
	}
	return nil, errors.New(fmt.Sprintf("mix name:%s not regist", name))
}
