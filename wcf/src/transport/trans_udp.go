package transport

import (
	"net"
	"time"
)

func init() {
	Regist("udp", func(addr string, extra interface{}) (net.Listener, error) {
		return net.Listen("udp", addr)
	}, func(addr string, timeout time.Duration, extra interface{}) (net.Conn, error) {
		return net.DialTimeout("udp", addr, timeout)
	}, nil)
}