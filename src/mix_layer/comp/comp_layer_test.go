package comp

import (
	"testing"
	"net"
	log "github.com/sirupsen/logrus"
	"time"
)

func handleProxy(conn net.Conn) {
	defer func() {
		conn.Close()
	}()
	cli, err := Wrap("hello", conn)
	if err != nil {
		log.Error(err)
		return
	}
	buf := make([]byte, 1024)
	for {
		cnt, err := cli.Read(buf)
		if err != nil {
			log.Errorf("Read data fail, e:%v", err)
			return
		}
		cnt, err = cli.Write(buf[:cnt])
		if err != nil {
			log.Errorf("Write data fail, e:%v", err)
			return
		}
	}
}

func TestSvr(t *testing.T) {
	svr, err := net.Listen("tcp", "127.0.0.1:8555")
	if err != nil {
		log.Fatal(err)
	}
	for {
		cli, err := svr.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go handleProxy(cli)
	}
}

func TestClient(t *testing.T) {
	cli, err := net.DialTimeout("tcp", "127.0.0.1:8555", 2 * time.Second)
	if err != nil {
		log.Fatal(err)
	}
	conn, err := Wrap("hello", cli)
	if err != nil {
		log.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		cnt, err := conn.Write([]byte("hello this is a test"))
		if err != nil {
			log.Fatal(err)
		}
		buffer := make([]byte, 1024)
		cnt, err = conn.Read(buffer)
		log.Infof("Read:%s", string(buffer[:cnt]))
	}
	conn.Close()
}

