package wcf

import (
	"testing"
	"github.com/sirupsen/logrus"
	"time"
	_ "net/http/pprof"
//	"net/http"
)

func TestStartRemote(t *testing.T) {
	//go func() {
	//	http.ListenAndServe("localhost:6060", nil)
	//}()
	cfg := NewServerConfig()
	cfg.Timeout = 5 * time.Second
	cfg.Localaddr = "127.0.0.1:8020"
	cfg.Userinfo = "D:/GoPath/src/wcf/cmd/server/userinfo.dat"
	cli := NewServer(cfg)
	err := cli.Start()
	if err != nil {
		logrus.Fatal(err)
	}
}

