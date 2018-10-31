package socks

import (
	"github.com/sirupsen/logrus"
	"net"
	"net_utils"
	"proxy"
	"sync"
	"testing"
)

func handleProxy(conn proxy.ProxyConn) {
	var cnt sync.WaitGroup
	cnt.Add(2)
	target, err := net.Dial("tcp", conn.GetTargetAddress())

	if err != nil {
		logrus.Printf("Dial to remote fail, addr:%s, err:%s, client:%s", conn.GetTargetAddress(), err, conn.RemoteAddr())
		conn.Close()
		return
	}
	logrus.Printf("Dial to remote:%s succ, client:%s", conn.GetTargetAddress(), conn.RemoteAddr())
	go func() {
		defer func() {
			cnt.Done()
		}()
		r, w, rerr, werr := net_utils.CopyTo(conn, target)
		logrus.Printf("client -> target, r:%d, w:%d, re:%v, we:%v", r, w, rerr, werr)
	}()
	go func() {
		defer func() {
			cnt.Done()
		}()
		r, w, rerr, werr := net_utils.CopyTo(target, conn)
		logrus.Printf("target -> client, r:%d, w:%d, re:%v, we:%v", r, w, rerr, werr)
	}()
	cnt.Wait()
	target.Close()
	conn.Close()
}

func TestAccept(t *testing.T) {
	acc, err := Bind("localhost:8010")
	if err != nil {
		t.Fatal(err)
	}
	acc.Start()
	for {
		client, err := acc.Accept()
		if err != nil {
			logrus.Printf("recv err:%v", err)
			continue
		}
		logrus.Printf("client:%v", client.GetTargetAddress())
		go handleProxy(client)
	}
}
