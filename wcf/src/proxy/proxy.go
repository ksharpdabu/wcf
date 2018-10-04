package proxy

import (
	"net"
	"errors"
	"fmt"
	"sync"
)

const (
	OP_TYPE_PROXY = 0          //proxy request
	OP_TYPE_FORWARD = 1        //forward request
)

const (
	ADDR_TYPE_IPV4 = 1
	ADDR_TYPE_IPV6 = 4
	ADDR_TYPE_DOMAIN = 3
)

type ConnRecv struct {
	Conn ProxyConn
	Err error
}

//
type HostCheckFunc func(host string, port uint16, hostType int) (bool, string, uint16, int)

type ProxyConn interface {
	net.Conn
	GetTargetName() string
	GetTargetType() int
	GetTargetPort() uint16
	GetTargetAddress() string
	GetTargetOPType() int
}

type ProxyListener interface {
	Handshake(conn net.Conn) (ProxyConn, error)
	Start() error
	Accept() (ProxyConn, error)
	AddHostHook(HostCheckFunc)
}

type BindFunc func(string) (ProxyListener, error)

type bindst struct {
	mu sync.RWMutex
	mp map[string]BindFunc
}

var bt *bindst

func init() {
	bt = &bindst{}
	bt.mp = make(map[string]BindFunc)
}

func Regist(network string, fun BindFunc) {
	bt.mu.Lock()
	defer bt.mu.Unlock()
	bt.mp[network] = fun
}

func Bind(network string, addr string) (ProxyListener, error) {
	bt.mu.RLock()
	defer bt.mu.RUnlock()
	if v, ok := bt.mp[network]; ok {
		return v(addr)
	}
	return nil, errors.New(fmt.Sprintf("unsupport protocol:%s", network))
}