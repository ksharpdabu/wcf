package http

import (
	"net"
	"errors"
	"fmt"
	"bytes"
	"strings"
	"strconv"
	"net_utils"
	"proxy"
	"check"
)

var HTTP_END = "\r\n\r\n"
var PROXY_REQ_NAME = "CONNECT"
var PROXY_RSP_STR = "%s 200 connection established\r\n\r\n"

func init() {
	proxy.Regist("http", func(addr string) (proxy.ProxyListener, error) {
		return Bind(addr)
	})
}

type HttpAddress struct {
	Name string
	Port uint16
	AddrType int
	Address string
	HttpType string
}

type HttpConn struct {
	address *HttpAddress
	net.Conn
	buffer []byte
}

func(this *HttpConn) Read(b []byte) (n int, err error) {
	if this.buffer == nil || len(this.buffer) == 0 {
		return this.Conn.Read(b)
	}
	cnt := copy(b, this.buffer)
	if cnt == len(this.buffer) {
		this.buffer = nil
	} else {
		this.buffer = this.buffer[cnt:]
	}
	return cnt, nil
}

func(this *HttpConn) GetTargetPort() uint16 {
	return this.address.Port
}

func(this *HttpConn) GetTargetAddress() string {
	return this.address.Address
}

func(this *HttpConn) GetTargetName() string {
	return this.address.Name
}

func(this *HttpConn) GetTargetType() int {
	return this.address.AddrType
}

func(this *HttpConn) GetHttpType() string {
	return this.address.HttpType
}

type HttpAcceptor struct {
	listener net.Listener
	connectionList chan *proxy.ConnRecv
	onHostCheck proxy.HostCheckFunc
}

func newAcceptorCtx() *HttpAcceptor {
	sa := &HttpAcceptor{}
	sa.connectionList = make(chan *proxy.ConnRecv, 5)
	return sa;
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

func(this *HttpAcceptor) AddHostHook(fun proxy.HostCheckFunc) {
	this.onHostCheck = fun
}

func(this *HttpAcceptor) Handshake(conn net.Conn) (proxy.ProxyConn, error) {
	buffer := make([]byte, 2048)
	index := 0
	total := len(buffer)
	for ;index < total; {
		cnt, err := conn.Read(buffer[index:])
		if err != nil || cnt <= 0{
			return nil, errors.New(fmt.Sprintf("read connect req from browser fail, err:%v, conn:%s", err, conn.RemoteAddr()))
		}
		index += cnt
		if index > 4 {
			if string(buffer[index - 4:index]) == HTTP_END {
				break
			}
		}
	}
	if string(buffer[index - 4:index]) != HTTP_END {
		return nil, errors.New(fmt.Sprintf("invalid proxy req, not end with \r\n, conn:%s", conn.RemoteAddr()))
	}
	arr := bytes.SplitN(buffer[:index], []byte("\r\n"), 2)
	line1 := string(arr[0])
	params := strings.Split(line1, " ")
	if len(params) != 3 {
		return nil, errors.New(fmt.Sprintf("invalid proxy params, line:%s, conn:%s", line1, conn.RemoteAddr()))
	}
	httpType := params[0]
	var url string
	var port uint16
	var spare []byte
	if httpType == PROXY_REQ_NAME {
		urlPort := strings.Split(params[1], ":")
		if len(urlPort) != 2 {
			return nil, errors.New(fmt.Sprintf("parse proxy url/port fail, param:%s, conn:%s", params[1], conn.RemoteAddr()))
		}
		url = urlPort[0]
		tmpPort, err := strconv.ParseUint(urlPort[1], 10, 16)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("parse proxy url/port fail, url:%s, param:%s, conn:%s", url, params[1], conn.RemoteAddr()))
		}
		port = uint16(tmpPort)
		rsp := fmt.Sprintf(PROXY_RSP_STR, params[2])
		err = net_utils.SendSpecLen(conn, []byte(rsp))
		if err != nil {
			return nil,errors.New(fmt.Sprintf("send proxy rsp to browser fail, err:%v, conn:%s", err, conn))
		}
	} else {
		spare = buffer[:index]
		err, _, turl, tport := check.GetUrlInfo(params[1])
		if err != nil {
			return nil, errors.New(fmt.Sprintf("parse url, port fail, err:%v, params:%s, conn:%s", err, params[1], conn.RemoteAddr()))
		}
		url = turl
		port = uint16(tport)
	}
	ip := net.ParseIP(url)
	addrType := 0
	if ip == nil {
		addrType = 0x3
	} else if ip.To4() != nil {
		addrType = 0x1
	} else {
		addrType = 0x4
	}
	if this.onHostCheck != nil {
		var ck bool
		ck, url, port, addrType = this.onHostCheck(url, port, addrType)
		if !ck {
			return nil, errors.New(fmt.Sprintf("host check fail, host:%s, conn:%s", url, conn.RemoteAddr()))
		}
	}
	return &HttpConn{Conn:conn, address:&HttpAddress{Name:url, Port:uint16(port), AddrType:addrType, Address:fmt.Sprintf("%s:%d", url, port), HttpType:httpType}, buffer:spare}, nil
}

func(this *HttpConn) GetTargetOPType() int {
	return proxy.OP_TYPE_PROXY
}

func(this *HttpAcceptor) Start() error {
	if this.listener == nil {
		return errors.New("no listener")
	}
	go func() {
		for {
			conn, err := this.listener.Accept()
			if err != nil {
				this.connectionList <- &proxy.ConnRecv{ nil, err }
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

func(this *HttpAcceptor) Accept() (proxy.ProxyConn, error) {
	if this.listener == nil {
		return nil, errors.New("no listener")
	}
	cli := <-this.connectionList
	return cli.Conn, cli.Err
}