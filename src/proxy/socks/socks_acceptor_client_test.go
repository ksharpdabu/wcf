package socks

import (
	"testing"
	"net"
	"github.com/sirupsen/logrus"
)

func TestSend(t *testing.T) {
	method := []byte {0x5, 0x1, 0x0}
	host := []byte { 0x05, 0x01, 0x00, 0x01, 0xCA, 0x67, 0xBE, 0x1B, 0x1C, 0x21 }
	buf := make([]byte, 2)
	conn, err := net.Dial("tcp", "localhost:8010")
	if err != nil {
		logrus.Fatal(err)
	}
	conn.Write(method)
	conn.Read(buf)
	conn.Write(host)
}

