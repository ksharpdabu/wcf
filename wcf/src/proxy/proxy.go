package proxy

import (
	"net"
	"errors"
	"fmt"
	"sync"
	"time"
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
type DialFunc func(addr string, proxy string, timeout time.Duration) (net.Conn, error)

type bindst struct {
	mu sync.RWMutex
	mp map[string]BindFunc
}

var bt *bindst
var dt *dialst

func init() {
	bt = &bindst{}
	bt.mp = make(map[string]BindFunc)
	dt = &dialst{}
	dt.mp = make(map[string]DialFunc)
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
	return nil, errors.New(fmt.Sprintf("bind unsupport protocol:%s", network))
}

type dialst struct {
	mu sync.RWMutex
	mp map[string]DialFunc
}

func RegistClient(network string, fun DialFunc) {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.mp[network] = fun
}

func DialTimeout(network string, addr string, proxy string, timeout time.Duration) (net.Conn, error) {
	dt.mu.RLock()
	defer dt.mu.RUnlock()
	if v, ok := dt.mp[network]; ok {
		return v(addr, proxy, timeout)
	}
	return nil, errors.New(fmt.Sprintf("dial unsupport protocol:%s", network))
}