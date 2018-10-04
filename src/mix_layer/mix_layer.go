package mix_layer

import (
	"net"
	"sync"
	"errors"
	"fmt"
	"strings"
)

type MixLayer struct {
	mp map[string]WrapFunc
	mu sync.Mutex
}

type WrapFunc func(key string, conn net.Conn) (MixConn, error)

type MixConn interface {
	net.Conn
	SetKey(string)
}

var layer *MixLayer

func init() {
	layer = &MixLayer{}
	layer.mp = make(map[string]WrapFunc)
}

func Regist(name string, fun WrapFunc) error {
	layer.mu.Lock()
	defer layer.mu.Unlock()
	if _, ok := layer.mp[name]; ok {
		return errors.New(fmt.Sprintf("name:%s conn already exists", name))
	}
	layer.mp[name] = fun
	return nil
}

type DefaultMixConn struct {
	net.Conn
}

func(this *DefaultMixConn) SetKey(string) {

}

func Wrap(name string, key string, conn net.Conn) (MixConn, error) {
	if len(name) == 0 || strings.ToLower(name) == "none" {
		return &DefaultMixConn{conn}, nil
	}
	layer.mu.Lock()
	defer layer.mu.Unlock()
	if v, ok := layer.mp[name]; ok {
		return v(key, conn)
	}
	return nil, errors.New(fmt.Sprintf("mix name:%s not regist", name))
}

