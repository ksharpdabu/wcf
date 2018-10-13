package transport_delegate

import (
	"transport"
	"net"
	"time"
)

func InitAllProtocol(file string) error {
	return transport.InitAllProtocol(file)
}

func Bind(pt string, addr string) (net.Listener, error) {
	return transport.Bind(pt, addr)
}

func Dial(pt string, addr string, timeout time.Duration) (net.Conn, error, int64) {
	start := time.Now()
	conn, err := transport.Dial(pt, addr, timeout)
	end := time.Now()
	return conn, err, int64(end.Sub(start) / time.Millisecond)
}