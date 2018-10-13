package relay

import (
	"testing"
	"net"
	log "github.com/sirupsen/logrus"
)

func TestDoAuth(t *testing.T) {
	conn, err := net.Dial("tcp", "127.0.0.1:8020")
	if err != nil {
		log.Fatalf("connect err:%v", err)
	}
	cfg := &RelayConfig{}
	cfg.Address.Addr = "solidot.org:443"
	cfg.Address.Name = "solidot.org"
	cfg.Address.Port = 443
	cfg.User = "test"
	cfg.Pwd = "xxx"
	cn, err, token := doAuth(conn, cfg)
	log.Infof("err:%v, token:%d, cn:%v", err, token, cn)
}

