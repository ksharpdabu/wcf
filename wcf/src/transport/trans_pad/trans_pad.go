package trans_pad

import (
	"crypto/rand"
	"encoding/json"
	"exchange"
	"fmt"
	"net"
	"time"
	"transport"
)

//整个协议的作用就是连接成功后client端向server端发送N个字节的数据, 然后接收M个字节的返回。
//N, M都可以为0, 2者都为0的时候等同于通常的tcp
type padBindCfg struct {
	SendLen int `json:"send_len"`
	RecvLen int `json:"recv_len"`
}

type padDialCfg struct {
	SendLen int `json:"send_len"`
	RecvLen int `json:"recv_len"` //两边的sendlen跟recvlen要一致, 不然会导致异常
}

type CounterPadFunc func(counter *int, b []byte) ([]byte, bool)

func initPadBindCfg(bindData []byte) (interface{}, error) {
	bd := &padBindCfg{}
	if err := json.Unmarshal(bindData, bd); err != nil {
		return nil, err
	}
	return bd, nil
}

func initPadDialCfg(dialData []byte) (interface{}, error) {
	dc := &padDialCfg{}
	if err := json.Unmarshal(dialData, dc); err != nil {
		return nil, err
	}
	return dc, nil
}

func initPadConfig(bindData []byte, dialData []byte) (interface{}, interface{}, error) {
	bindCfg, berr := initPadBindCfg(bindData)
	dialCfg, derr := initPadDialCfg(dialData)
	var err error
	if berr != nil || derr != nil {
		err = fmt.Errorf("init cfg fail, berr:%v, derr:%v", berr, derr)
	}
	return bindCfg, dialCfg, err
}

func ExchangeAdaptor(cnt int, fun CounterPadFunc) exchange.OnDataTransfer {
	v := cnt
	return func(b []byte) ([]byte, bool) {
		return fun(&v, b)
	}
}

func padAndCount(counter *int, b []byte) ([]byte, bool) {
	if *counter == 0 {
		return b, false
	}
	skip := *counter
	if skip > len(b) {
		skip = len(b)
	}
	b = b[skip:]
	*counter -= skip
	return b, true
}

type PadListener struct {
	listener net.Listener
	sPad     int
	rPad     int
}

func (this *PadListener) Accept() (net.Conn, error) {
	conn, err := this.listener.Accept()
	if err != nil {
		return nil, err
	}
	writeData := make([]byte, this.sPad)
	rand.Reader.Read(writeData)
	return exchange.NewExchangeConn(conn, nil, writeData, ExchangeAdaptor(this.rPad, padAndCount), nil), err
}

func (this *PadListener) Close() error {
	return this.listener.Close()
}

func (this *PadListener) Addr() net.Addr {
	return this.listener.Addr()
}

func NewPadClient(network, addr string, s, r int) (net.Conn, error) {
	conn, err := net.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	writeData := make([]byte, s)
	rand.Reader.Read(writeData)
	return exchange.NewExchangeConn(conn, nil, writeData, ExchangeAdaptor(r, padAndCount), nil), err
}

func NewPadListener(network, addr string, s, r int) (*PadListener, error) {
	if s < 0 || r < 0 {
		return nil, fmt.Errorf("create pad listener fail, pad len invalid, s:%d, r:%d", s, r)
	}
	l, err := net.Listen(network, addr)
	if err != nil {
		return nil, err
	}
	return &PadListener{l, s, r}, nil
}

func init() {
	transport.Regist("tcp_pad", func(addr string, extra interface{}) (net.Listener, error) {
		s := 59
		r := 72
		if extra != nil {
			bd := extra.(*padBindCfg)
			s = bd.SendLen
			r = bd.RecvLen
		}
		return NewPadListener("tcp", addr, s, r)
	}, func(addr string, timeout time.Duration, extra interface{}) (net.Conn, error) {
		s := 72
		r := 59
		if extra != nil {
			bi := extra.(*padDialCfg)
			s = bi.SendLen
			r = bi.RecvLen
		}
		return NewPadClient("tcp", addr, s, r)
	}, initPadConfig)
}
