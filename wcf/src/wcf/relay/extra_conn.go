package relay

import (
	"errors"
	"fmt"
	"net"
)

type ExtraConn struct {
	net.Conn
	rbuf []byte
}

func (this *ExtraConn) Read(b []byte) (int, error) {
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

func (this *ExtraConn) Close() error {
	if len(this.rbuf) != 0 {
		return errors.New(fmt.Sprintf("data not empty, r:%d", len(this.rbuf)))
	}
	return this.Conn.Close()
}
