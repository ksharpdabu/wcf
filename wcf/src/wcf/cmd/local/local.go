package main

import (
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"mix_delegate"
	"net/http"
	_ "net/http/pprof"
	"strings"
	"transport_delegate"
	"wcf"
)

var config *string = flag.String("config", "./wcf/src/config/local.json", "config file")

func main() {
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	log.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true
	//go tool pprof http://localhost:6060/debug/pprof/profile
	go func() {
		http.ListenAndServe("localhost:6062", nil)
	}()
	flag.Parse()
	cfg := wcf.NewLocalConfig()
	err := cfg.Parse(*config)
	if err != nil {
		log.Fatalf("Read config fail, err:%v, config:%s", err, *config)
	}
	if err := transport_delegate.InitAllProtocol(cfg.TransportConfig); err != nil {
		log.Fatalf("Init transport config fail, err:%v, config:%s", err, cfg.TransportConfig)
	}
	log.Infof("Config:%+v", cfg)
	log.Infof("All support mix name:[%s]", strings.Join(mix_delegate.GetAllMixName(), ","))
	if !mix_delegate.CheckMixName(cfg.Encrypt) {
		panic(fmt.Sprintf("could not found mix name:%s", cfg.Encrypt))
	}
	cli := wcf.NewClient(cfg)
	if cli == nil {
		panic("could not create wcf local")
	}
	cli.Start()
}
