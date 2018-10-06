package wcf

import (
	"testing"
	"time"
	"github.com/sirupsen/logrus"
	"net"
)

func TestConnect(t *testing.T) {
	conn, err := net.DialTimeout("tcp", "www.pin-cong.com:443", 2 * time.Second)
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
}

func TestStart(t *testing.T) {
	cfg := LocalConfig{}
	cfg.Timeout = 5 * time.Second
	cfg.Proxyaddr = []ProxyAddrInfo {ProxyAddrInfo{"127.0.0.1:8020", 50}}
	cfg.Localaddr = append(cfg.Localaddr, AddrConfig{Name:"socks", Address:"127.0.0.1:8010"})
	cfg.User = "test"
	cfg.Pwd = "xxx"
	cli := NewClient(&cfg)
	err := cli.Start()
	if err != nil {
		logrus.Fatal(err)
	}
}

