package wcf

import (
	"github.com/sirupsen/logrus"
	_ "net/http/pprof"
	"testing"
	"time"
	//    "net/http"
	"io/ioutil"
	"net/http"
)

func TestStartRemote(t *testing.T) {
	//go func() {
	//    http.ListenAndServe("localhost:6060", nil)
	//}()
	cfg := NewServerConfig()
	cfg.Timeout = 5 * time.Second
	cfg.Localaddr = append(cfg.Localaddr, ProxyAddrConfig{"tcp", "127.0.0.1:8020"})
	cfg.Userinfo = "D:/GoPath/src/wcf/cmd/server/userinfo.dat"
	cli := NewServer(cfg)
	err := cli.Start()
	if err != nil {
		logrus.Fatal(err)
	}
}

func TestConfuse(t *testing.T) {
	resp, err := http.Get("http://127.0.0.1:8020")
	if err != nil {
		logrus.Fatal(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.Info(string(body))
}
