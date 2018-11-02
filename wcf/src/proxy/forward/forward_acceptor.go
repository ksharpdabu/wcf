package forward

import (
	"errors"
	"fmt"
	"net"
	"proxy"
	"strconv"
)

func init() {
	proxy.Regist("forward", func(addr string, extra interface{}) (proxy.ProxyListener, error) {
		return Bind(addr, extra)
	})
}

type ForwardAcceptor struct {
	listener   net.Listener
	targetAddr string
	targetPort int
}

type ForwardConn struct {
	net.Conn
	targetAddr string
	targetPort int
}

func (this *ForwardConn) GetTargetPort() uint16 {
	return uint16(this.targetPort)
}

func (this *ForwardConn) GetTargetAddress() string {
	return fmt.Sprintf("%s:%d", this.targetAddr, this.targetPort)
}

func (this *ForwardConn) GetTargetName() string {
	return this.targetAddr
}

func (this *ForwardConn) GetTargetType() int {
	return proxy.ADDR_TYPE_DETERMING //1-ipv4, 3-domain, 4-ipv6
}

func (this *ForwardConn) GetTargetOPType() int {
	return proxy.OP_TYPE_FORWARD
}

func (this *ForwardAcceptor) AddHostHook(fun proxy.HostCheckFunc) {

}

func WrapListener(listener net.Listener, extra interface{}) (*ForwardAcceptor, error) {
	if extra == nil {
		return nil, errors.New(fmt.Sprintf("not target params found, extra:%v", extra))
	}
	host, sport, err := net.SplitHostPort(extra.(string))
	if err != nil {
		return nil, errors.New(fmt.Sprintf("split ip/port fail, err:%v, extra:%v", err, extra))
	}
	port, err := strconv.ParseUint(sport, 10, 32)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("parse port fail, err:%v, extra:%v", err, extra))
	}
	return &ForwardAcceptor{listener: listener, targetAddr: host, targetPort: int(port)}, nil
}

func Bind(addr string, extra interface{}) (*ForwardAcceptor, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return WrapListener(listener, extra)
}

func (this *ForwardAcceptor) Handshake(conn net.Conn) (proxy.ProxyConn, error) {
	return &ForwardConn{Conn: conn, targetAddr: this.targetAddr, targetPort: this.targetPort}, nil
}

func (this *ForwardAcceptor) Start() error {
	if this.listener == nil {
		return errors.New("no listener")
	}
	return nil
}

func (this *ForwardAcceptor) Accept() (proxy.ProxyConn, error) {
	conn, err := this.listener.Accept()
	if err != nil {
		return nil, err
	}
	return this.Handshake(conn)
}
