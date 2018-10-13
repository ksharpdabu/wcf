package relay

import (
	"testing"
	"github.com/sirupsen/logrus"
	"time"
	"sync"
)

var server string = "127.0.0.1:8000"

func getConfig() *RelayConfig {
	cfg := &RelayConfig{}
	cfg.Pwd = "xxx"
	cfg.User = "test"
	cfg.Address.Addr = "segmentfault.com"
	cfg.Address.AddrType = 3
	return cfg
}

func TestRelayListen(t *testing.T) {
	cfg := getConfig()
	acc, err := Bind("tcp", server)
	if err != nil {
		logrus.Fatal(err)
	}
	acc.OnAuth = func(user, pwd string) bool {
		if user == cfg.User && pwd == cfg.Pwd {
			return true
		}
		return false
	}
	acc.Start()
	for {
		cli, err := acc.Accept()
		if err != nil {
			logrus.Fatal(err)
		}
		logrus.Printf("Recv conn succ, client:%s, token:%d", cli.RemoteAddr(), cli.token)
		buffer := make([]byte, 20)
		cnt, _ := cli.Read(buffer)
		buffer = buffer[0:cnt]
		cli.Write(buffer)
		logrus.Printf("Recv data from client:%s, data:%s", cli.RemoteAddr(), string(buffer))
		cli.Close()
	}
}

func TestRelayConnect(t *testing.T) {
	cfg := getConfig()
	doTimes := 100
	var gp sync.WaitGroup
	gp.Add(doTimes)
	for i := 0; i < doTimes; i++ {
		func() {
			defer func() {
				gp.Done()
			}()
			conn, err := DialWithTimeout(server, 1 * time.Second, cfg)
			if err != nil {
				logrus.Fatal(err)
			}
			logrus.Printf("Connect to svr succ, client:%s", conn.RemoteAddr())
			conn.Write([]byte("hello"))
			buffer := make([]byte, 10)
			cnt, _ := conn.Read(buffer)
			buffer = buffer[0:cnt]
			logrus.Printf("Recv data back from svr, data:%s", string(buffer))
			conn.Close()
		} ()
	}
	gp.Wait()
}



