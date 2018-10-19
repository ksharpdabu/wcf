package http

import (
	"testing"
	"net"
	"time"
)

func TestSendHttpReq(t *testing.T) {
	conn, err := net.Dial("tcp", "localhost:8020")
	if err != nil {
		t.Fatal(err)
	}
	conn.Write([]byte("GET /test=1 HTTP/1.1\r\n\r\n"))
	buf := make([]byte, 2048)
	for {
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		cnt, err := conn.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("data:%s", string(buf[:cnt]))
	}
}

