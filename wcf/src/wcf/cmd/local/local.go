package main

import (
	"wcf"
	_ "proxy"
	"flag"
	_ "proxy/socks"
	_ "proxy/http"
	_ "proxy/forward"
	log "github.com/sirupsen/logrus"
	"net/http"
	_ "net/http/pprof"
	_ "mix_layer"
	_ "mix_layer/comp"
	_ "mix_layer/xor"
)

var config *string = flag.String("config", "d:/GoProj/wcf_proj/src/wcf/cmd/local/local.json", "config file")

func main() {
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	log.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true

	//go tool pprof http://localhost:6060/debug/pprof/profile
	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()
	flag.Parse()
	cfg := wcf.NewLocalConfig()
	err := cfg.Parse(*config)
	if err != nil {
		log.Fatalf("Read config fail, err:%v, config:%s", err, *config)
	}
	log.Printf("Config:%+v", cfg)
	cli := wcf.NewClient(cfg)
	cli.Start()
}