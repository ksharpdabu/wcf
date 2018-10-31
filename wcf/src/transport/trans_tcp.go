package transport

import (
	"net"
	"time"
)

func init() {
	Regist("tcp", func(addr string, extra interface{}) (net.Listener, error) {
		return net.Listen("tcp", addr)
	}, func(addr string, timeout time.Duration, extra interface{}) (net.Conn, error) {
		return net.DialTimeout("tcp", addr, timeout)
	}, nil)
}
