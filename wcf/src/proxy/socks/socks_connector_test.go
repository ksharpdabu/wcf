package socks

import (
	"testing"
	"time"
	"proxy"
)

func TestProxy(t *testing.T) {
	conn, err := proxy.DialTimeout("socks", "[::1]:1080", "127.0.0.1:8010", time.Second * 5)
	if err != nil {
		t.Fatal(err)
	}
	_, err = conn.Write([]byte("GET / HTTP1.1\r\n\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 2048)
	cnt, err := conn.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("cnt:%d, string:%s", cnt, string(buf[:cnt]))
	conn.Close()
}

func TestCheckResult(t *testing.T) {
	{
		buffer := []byte { 0x5, 0x0, 0x0, 0x1, 0x7f, 0x0, 0x0, 0x1, 0xf4, 0x1 }
		cnt, err := CheckSocks5RspResult(buffer)
		t.Logf("cnt:%d, err:%v", cnt, err)
	}
	{
		buffer := []byte { 0x5, 0x0, 0x0, 0x3, 0x7f, 0x0, 0x0, 0x1, 0xf4, 0x1 }
		cnt, err := CheckSocks5RspResult(buffer)
		t.Logf("cnt:%d, err:%v", cnt, err)
	}
	{
		buffer := []byte { 0x5, 0x0, 0x0, 0x3, 0x3, 0x0, 0x0, 0x1, 0xf4, 0x1 }
		cnt, err := CheckSocks5RspResult(buffer)
		t.Logf("cnt:%d, err:%v", cnt, err)
	}
	{
		buffer := []byte { 0x5, 0x0, 0x0, 0x3, 0x3, 0x0, 0x0, 0x1, 0xf4, 0x1, 0x5, 0x6, 0x7 }
		cnt, err := CheckSocks5RspResult(buffer)
		t.Logf("cnt:%d, err:%v", cnt, err)
	}
}



