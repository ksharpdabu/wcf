package http

import "testing"
import (
	"context"
	log "github.com/sirupsen/logrus"
	"net"
	"net_utils"
	"proxy"
	"time"
)

func handleProxy(conn proxy.ProxyConn) {
	remote, err := net.DialTimeout("tcp", conn.GetTargetAddress(), 5*time.Second)
	if err != nil {
		log.Error(err)
		return
	}
	src := make([]byte, 1024)
	dst := make([]byte, 1024)
	ctx, cancel := context.WithCancel(context.Background())
	net_utils.Pipe(conn, remote, src, dst, ctx, cancel, 5*time.Second)
}

func TestHttpAcceptor_Accept(t *testing.T) {
	svr, err := Bind("127.0.0.1:8011")
	if err != nil {
		log.Fatal(err)
	}
	err = svr.Start()
	if err != nil {
		log.Fatal(err)
	}
	for {
		cli, err := svr.Accept()
		if err != nil {
			log.Errorf("Recv conn fail, err:%v", err)
			continue
		}
		log.Printf("Recv conn from browser, target addr:%s, target type:%d", cli.GetTargetAddress(), cli.GetTargetType())
		go handleProxy(cli)
	}
}
