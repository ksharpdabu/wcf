package exchange

import (
	"fmt"
	"net"
	"net_utils"
)

//return byte array, is data changed
type OnDataTransfer func([]byte) ([]byte, bool)

type ExchangeConn struct {
	net.Conn
	rbuf []byte
	wbuf []byte
	rt   OnDataTransfer
	wt   OnDataTransfer
}

func defaultRead(b []byte) ([]byte, bool) {
	return b, false
}

func defaultWrite(b []byte) ([]byte, bool) {
	return b, false
}

func NewExchangeConn(conn net.Conn, rbuf, wbuf []byte, rt, wt OnDataTransfer) *ExchangeConn {
	ex := &ExchangeConn{}
	ex.Conn = conn
	if rt == nil {
		rt = defaultRead
	}
	if wt == nil {
		wt = defaultWrite
	}
	ex.rt = rt
	ex.wt = wt
	ex.rbuf, _ = ex.rt(rbuf)
	ex.wbuf, _ = ex.wt(wbuf)
	return ex
}

func (this *ExchangeConn) Read(b []byte) (int, error) {
	if len(this.rbuf) != 0 {
		cnt := copy(b, this.rbuf)
		if cnt == len(this.rbuf) {
			this.rbuf = nil
		} else {
			this.rbuf = this.rbuf[cnt:]
		}
		return cnt, nil
	}
	for {
		cnt, err := this.Conn.Read(b)
		if err != nil {
			return 0, err
		}
		filter, chg := this.rt(b[:cnt])
		if len(filter) == 0 {
			continue
		}
		if chg {
			return copy(b, filter), nil
		}
		return len(filter), nil
	}
}

func (this *ExchangeConn) Write(b []byte) (int, error) {
	if len(this.wbuf) != 0 {
		err := net_utils.SendSpecLen(this.Conn, this.wbuf)
		if err != nil {
			return 0, err
		}
		this.wbuf = nil
	}
	writeData, _ := this.wt(b)
	if len(writeData) != 0 {
		if err := net_utils.SendSpecLen(this.Conn, writeData); err != nil {
			return 0, err
		}
	}
	return len(b), nil
}

func (this *ExchangeConn) Close() error {
	if len(this.rbuf) != 0 || len(this.wbuf) != 0 {
		return fmt.Errorf("data spare, rb:%d, wb:%d", len(this.rbuf), len(this.wbuf))
	}
	return nil
}
