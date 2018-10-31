package proxy_delegate

import (
	"net"
	"proxy"
	_ "proxy"
	_ "proxy/forward"
	_ "proxy/http"
	_ "proxy/socks"
	"time"
)

func DialTimeout(network string, addr string, px string, timeout time.Duration) (net.Conn, error) {
	return proxy.DialTimeout(network, addr, px, timeout)
}

func Bind(network string, addr string) (proxy.ProxyListener, error) {
	return proxy.Bind(network, addr)
}
