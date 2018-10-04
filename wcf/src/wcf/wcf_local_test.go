package wcf

import (
	"testing"
	"time"
	"github.com/sirupsen/logrus"
)

func TestStart(t *testing.T) {
	cfg := LocalConfig{}
	cfg.Timeout = 5 * time.Second
	cfg.Proxyaddr = "127.0.0.1:8020"
	cfg.Localaddr = append(cfg.Localaddr, AddrConfig{Name:"socks", Address:"127.0.0.1:8010"})
	cfg.User = "test"
	cfg.Pwd = "xxx"
	cli := NewClient(&cfg)
	err := cli.Start()
	if err != nil {
		logrus.Fatal(err)
	}
}

