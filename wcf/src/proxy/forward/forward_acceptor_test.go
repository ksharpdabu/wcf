package forward

import (
	log "github.com/sirupsen/logrus"
	"net"
	"testing"
	"time"
)

func handleProxy(conn net.Conn, t *testing.T) {
	defer func() {
		conn.Close()
	}()
	buf := make([]byte, 1024)
	cnt, err := conn.Read(buf)
	if err != nil {
		log.Errorf("Recv err:%v, conn:%s", err, conn.RemoteAddr())
		return
	}
	log.Infof("Recv:%s from client:%s", string(buf[:cnt]), conn.RemoteAddr())
	cnt, err = conn.Write(buf[:cnt])
	if err != nil {
		log.Errorf("Write back:%s to remote err:%s, conn:%s", string(buf[:cnt]), err, conn.RemoteAddr())
		return
	}
	log.Infof("Write back:%s to remote succ, conn:%s", string(buf[:cnt]), conn.RemoteAddr())
}

func TestEchoSvr(t *testing.T) {
	listen, err := net.Listen("tcp", "127.0.0.1:8000")
	if err != nil {
		t.Fatal(err)
	}
	for {
		cli, err := listen.Accept()
		if err != nil {
			t.Fatal(err)
		}
		log.Infof("Recv client:%s", cli.RemoteAddr())
		go handleProxy(cli, t)
	}
}

func TestEchoClient(t *testing.T) {
	cli, err := net.DialTimeout("tcp", "127.0.0.1:8012", 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	cli.Write([]byte("hello"))
	buf := make([]byte, 1024)
	cnt, err := cli.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	log.Infof("Recv:%s", string(buf[:cnt]))

}
