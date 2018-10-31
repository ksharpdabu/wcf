package http

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net_utils"
	"proxy"
)

var PROXY_RSP_SUCC = []byte("HTTP/1.1 200 connection established\r\n\r\n")

func init() {
	proxy.Regist("http", func(addr string) (proxy.ProxyListener, error) {
		return Bind(addr)
	})
}

type HttpAddress struct {
	Name     string
	Port     uint16
	AddrType int
	Address  string
	HttpType string
}

type HttpConn struct {
	address *HttpAddress
	net.Conn
	bReader *bufio.Reader
	buffer  []byte
}

func (this *HttpConn) Read(b []byte) (n int, err error) {
	if this.buffer == nil || len(this.buffer) == 0 {
		return this.bReader.Read(b)
	}
	cnt := copy(b, this.buffer)
	if cnt == len(this.buffer) {
		this.buffer = nil
	} else {
		this.buffer = this.buffer[cnt:]
	}
	return cnt, nil
}

func (this *HttpConn) GetTargetPort() uint16 {
	return this.address.Port
}

func (this *HttpConn) GetTargetAddress() string {
	return this.address.Address
}

func (this *HttpConn) GetTargetName() string {
	return this.address.Name
}

func (this *HttpConn) GetTargetType() int {
	return this.address.AddrType
}

func (this *HttpConn) GetHttpType() string {
	return this.address.HttpType
}

type HttpAcceptor struct {
	listener       net.Listener
	connectionList chan *proxy.ConnRecv
	onHostCheck    proxy.HostCheckFunc
}

func newAcceptorCtx() *HttpAcceptor {
	sa := &HttpAcceptor{}
	sa.connectionList = make(chan *proxy.ConnRecv, 5)
	return sa
}

func Bind(addr string) (proxy.ProxyListener, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return WrapListener(listener)
}

func WrapListener(listener net.Listener) (*HttpAcceptor, error) {
	sa := newAcceptorCtx()
	sa.listener = listener
	return sa, nil
}

func (this *HttpAcceptor) AddHostHook(fun proxy.HostCheckFunc) {
	this.onHostCheck = fun
}

func (this *HttpAcceptor) reqToBytes(r *http.Request) []byte {
	var writer bytes.Buffer
	r.Write(&writer)
	return writer.Bytes()
}

func (this *HttpAcceptor) parseAddrPort(host string) (error, string, uint16) {
	err, _, addr, port := net_utils.GetUrlInfo(host)
	return err, addr, uint16(port)
}

//refer taosocks
func (this *HttpAcceptor) Handshake(conn net.Conn) (proxy.ProxyConn, error) {
	bReader := bufio.NewReader(conn)
	req, err := http.ReadRequest(bReader)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("read http request fail, err:%v, conn:%s", err, conn.RemoteAddr()))
	}
	var waitToSend []byte
	if req.Method == http.MethodConnect {
		if err = net_utils.SendSpecLen(conn, PROXY_RSP_SUCC); err != nil {
			return nil, errors.New(fmt.Sprintf("send connect method reply to browser fail, err:%v, conn:%s", err, conn.RemoteAddr()))
		}
	} else {
		waitToSend = this.reqToBytes(req)
	}
	err, addr, port := this.parseAddrPort(req.Host)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("parse http addr/port fail, err:%v, host:%s", err, req.Host))
	}
	return &HttpConn{
		Conn:    conn,
		address: &HttpAddress{Name: addr, Port: port, AddrType: proxy.ADDR_TYPE_DETERMING, Address: fmt.Sprintf("%s:%d", addr, port), HttpType: req.Method},
		buffer:  waitToSend, bReader: bReader,
	}, nil
}

func (this *HttpConn) GetTargetOPType() int {
	return proxy.OP_TYPE_PROXY
}

func (this *HttpAcceptor) Start() error {
	if this.listener == nil {
		return errors.New("no listener")
	}
	go func() {
		for {
			conn, err := this.listener.Accept()
			if err != nil {
				this.connectionList <- &proxy.ConnRecv{nil, err}
				continue
			}
			go func() {
				client, err := this.Handshake(conn)
				if err != nil {
					this.connectionList <- &proxy.ConnRecv{nil, err}
					conn.Close()
				} else {
					this.connectionList <- &proxy.ConnRecv{client, nil}
				}
			}()
		}
	}()
	return nil
}

func (this *HttpAcceptor) Accept() (proxy.ProxyConn, error) {
	if this.listener == nil {
		return nil, errors.New("no listener")
	}
	cli := <-this.connectionList
	return cli.Conn, cli.Err
}
