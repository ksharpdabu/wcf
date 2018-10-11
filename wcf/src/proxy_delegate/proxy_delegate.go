package proxy_delegate

import (
	_ "proxy"
	_ "proxy/socks"
	_ "proxy/forward"
	_ "proxy/http"
	"time"
	"net"
	"proxy"
)

func DialTimeout(network string, addr string, px string, timeout time.Duration) (net.Conn, error) {
	return proxy.DialTimeout(network, addr, px, timeout)
}

func Bind(network string, addr string) (proxy.ProxyListener, error) {
	return proxy.Bind(network, addr)
}