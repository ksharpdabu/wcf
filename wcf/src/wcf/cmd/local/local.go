package main

import (
	"wcf"
	"flag"
	log "github.com/sirupsen/logrus"
	"net/http"
	_ "net/http/pprof"
	"transport_delegate"
)

var config *string = flag.String("config", "D:/GoProj/wcf/wcf/src/config/local.json", "config file")

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
	if err :=transport_delegate.InitAllProtocol(cfg.TransportConfig); err != nil {
		log.Fatalf("Init transport config fail, err:%v, config:%s", err, cfg.TransportConfig)
	}
	log.Infof("Config:%+v", cfg)
	cli := wcf.NewClient(cfg)
	if cli == nil {
		panic("could not create wcf local")
	}
	cli.Start()
}