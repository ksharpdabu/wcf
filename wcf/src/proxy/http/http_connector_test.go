package http

import (
	"testing"
	"time"
	"proxy"
)

func TestProxy(t *testing.T) {
	conn, err := proxy.DialTimeout("http", "solidot.org:80", "127.0.0.1:8011", time.Second * 5)
	if err != nil {
		t.Fatal(err)
	}
	_, err = conn.Write([]byte("GET / HTTP1.1\r\n\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 1024)
	cnt, err := conn.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(buf[:cnt]))
	conn.Close()
}
