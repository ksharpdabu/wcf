package forward

import (
	"net"
	"errors"
	"proxy"
)

func init() {
	proxy.Regist("forward", func(addr string) (proxy.ProxyListener, error) {
		return Bind(addr)
	})
}

type ForwardAcceptor struct {
	listener net.Listener
}

type ForwardConn struct {
	net.Conn
}

func(this *ForwardConn) GetTargetPort() uint16 {
	return 0
}

func(this *ForwardConn) GetTargetAddress() string {
	return "0.0.0.0:0"
}

func(this *ForwardConn) GetTargetName() string {
	return "0.0.0.0"
}

func(this *ForwardConn) GetTargetType() int {
	return proxy.ADDR_TYPE_IPV4   //1-ipv4, 3-domain, 4-ipv6
}

func(this *ForwardConn) GetTargetOPType() int {
	return proxy.OP_TYPE_FORWARD
}

func(this *ForwardAcceptor) AddHostHook(fun proxy.HostCheckFunc) {

}

func WrapListener(listener net.Listener) (*ForwardAcceptor, error) {
	return &ForwardAcceptor{listener:listener}, nil
}

func Bind(addr string) (*ForwardAcceptor, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return WrapListener(listener)
}

func(this *ForwardAcceptor) Handshake(conn net.Conn) (proxy.ProxyConn, error) {
	return &ForwardConn{conn}, nil
}

func(this *ForwardAcceptor) Start() error {
	if this.listener == nil {
		return errors.New("no listener")
	}
	return nil
}

func(this *ForwardAcceptor) Accept() (proxy.ProxyConn, error) {
	conn, err := this.listener.Accept()
	if err != nil {
		return nil, err
	}
	return &ForwardConn{conn}, nil
}


