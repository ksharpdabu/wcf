package socks

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net_utils"
	"proxy"
	//log "github.com/sirupsen/logrus"
)

func init() {
	proxy.Regist("socks", func(addr string, extra interface{}) (proxy.ProxyListener, error) {
		return Bind(addr)
	})
}

type SocksAcceptor struct {
	listener       net.Listener
	connectionList chan *proxy.ConnRecv
	onHostCheck    proxy.HostCheckFunc
}

func newAcceptorCtx() *SocksAcceptor {
	sa := &SocksAcceptor{}
	sa.connectionList = make(chan *proxy.ConnRecv, 5)
	return sa
}

type SocksAddress struct {
	Address  string
	AddrType int
	Name     string
	Port     uint16
}

type SocksConn struct {
	address *SocksAddress
	net.Conn
	rbuf []byte
}

func (this *SocksConn) Read(b []byte) (int, error) {
	if len(this.rbuf) != 0 {
		cnt := copy(b, this.rbuf)
		if cnt == len(this.rbuf) {
			this.rbuf = nil
		} else {
			this.rbuf = this.rbuf[cnt:]
		}
		return cnt, nil
	}
	return this.Conn.Read(b)
}

func (this *SocksConn) GetTargetName() string {
	return this.address.Name
}

func (this *SocksConn) GetTargetPort() uint16 {
	return this.address.Port
}

func (this *SocksConn) GetTargetType() int {
	return this.address.AddrType
}

func (this *SocksConn) GetTargetOPType() int {
	return proxy.OP_TYPE_PROXY
}

func (this *SocksConn) GetTargetAddress() string {
	return this.address.Address
}

func Bind(addr string) (*SocksAcceptor, error) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return WrapListener(listener)
}

func WrapListener(listener net.Listener) (*SocksAcceptor, error) {
	sa := newAcceptorCtx()
	sa.listener = listener
	return sa, nil
}

func (this *SocksAcceptor) AddHostHook(fun proxy.HostCheckFunc) {
	this.onHostCheck = fun
}

func (this *SocksAcceptor) Handshake(conn net.Conn) (proxy.ProxyConn, error) {
	buf := make([]byte, 512)
	index := 0
	hasSendHandshake := false
	var addr string
	var port uint16
	var atyp int
	var total int
	for {
		cnt, err := conn.Read(buf[index:])
		if err != nil {
			return nil, errors.New(fmt.Sprintf("read data from remote fail, err:%v, conn:%s", err, conn.RemoteAddr()))
		}
		index += cnt
		if !hasSendHandshake {
			if len(buf) < 3 {
				continue
			}
			ms := int(buf[1])
			if len(buf) < ms+1+1 {
				continue
			}
			if buf[0] != 0x5 {
				conn.Write([]byte{0x5, 0xFF})
				return nil, errors.New(fmt.Sprintf("invalid ver:%d from conn:%s", int(buf[0]), conn.RemoteAddr()))
			}
			found := false
			for i := 0; i < ms; i++ {
				if buf[2+i] == 0x0 {
					found = true
					break
				}
			}
			if !found {
				conn.Write([]byte{0x5, 0xFF})
				return nil, errors.New(fmt.Sprintf("not found method:0x0 from conn:%s", conn.RemoteAddr()))
			}
			err := net_utils.SendSpecLen(conn, []byte{0x5, 0x0})
			if err != nil {
				return nil, errors.New(fmt.Sprintf("send socks hand shake fail, err:%v, conn:%s", err, conn.RemoteAddr()))
			}
			copy(buf, buf[ms+1+1:])
			index -= ms + 1 + 1
			hasSendHandshake = true
		}
		if len(buf[:index]) < 7 {
			continue
		}
		total = 0
		atyp = int(buf[3])
		if atyp == proxy.ADDR_TYPE_IPV4 {
			total = 4 + 4 + 2
		} else if atyp == proxy.ADDR_TYPE_DOMAIN {
			total = 4 + 1 + int(buf[4]) + 2
		} else if atyp == proxy.ADDR_TYPE_IPV6 {
			total = 4 + 16 + 2
		} else {
			return nil, errors.New(fmt.Sprintf("recv invalid atyp:%d from conn:%s, databuf:%s", atyp, conn.RemoteAddr(), hex.EncodeToString(buf[:index])))
		}
		if len(buf[:index]) < total {
			continue
		}

		if atyp == proxy.ADDR_TYPE_IPV4 {
			addr = net.IP(buf[4:8]).String()
			port = binary.BigEndian.Uint16(buf[8:10])
		} else if atyp == proxy.ADDR_TYPE_DOMAIN {
			addr = string(buf[5 : 5+int(buf[4])])
			port = binary.BigEndian.Uint16(buf[5+int(buf[4]) : 5+int(buf[4])+2])
		} else if atyp == proxy.ADDR_TYPE_IPV6 {
			addr = net.IP(buf[4:20]).String()
			port = binary.BigEndian.Uint16(buf[20:22])
		}
		break
	}
	ck := true
	if this.onHostCheck != nil {
		ck, addr, port, atyp = this.onHostCheck(addr, port, atyp)
	}
	buf[1] = 0x0 //succ
	if !ck {
		buf[1] = 0x02
	}

	err := net_utils.SendSpecLen(conn, buf[0:total])
	if !ck {
		return nil, errors.New(fmt.Sprintf("host check fail, host:%s, port:%d", addr, port))
	}
	if err != nil {
		return nil, errors.New(fmt.Sprintf("socks conn send connection info rsp fail, err:%v, conn:%s", err, conn.RemoteAddr()))
	}
	var rb []byte
	if total < index {
		rb = buf[index:]
	}
	//log.Infof("Handshake finish, rb len:%d, buf:%s", len(rb), hex.EncodeToString(buf[:index]))
	return &SocksConn{&SocksAddress{fmt.Sprintf("%s:%d", addr, port), atyp, addr, port}, conn, rb}, nil
}

func (this *SocksAcceptor) Start() error {
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

func (this *SocksAcceptor) Accept() (proxy.ProxyConn, error) {
	if this.listener == nil {
		return nil, errors.New("no listener")
	}
	cli := <-this.connectionList
	return cli.Conn, cli.Err
}
