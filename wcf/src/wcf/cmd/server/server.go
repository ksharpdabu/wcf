package main

import (
	"flag"
	"net/http"
	_ "net/http/pprof"
	log "github.com/sirupsen/logrus"
	"wcf"
	"transport_delegate"
)

var config *string = flag.String("config", "D:/GoProj/wcf/wcf/src/config/server.json", "config file")

func main() {
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	log.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true
	go func() {
		http.ListenAndServe("localhost:6061", nil)
	}()
	flag.Parse()
	cfg := wcf.NewServerConfig()
	err := cfg.Parse(*config)
	if err != nil {
		log.Fatalf("Read config fail, err:%v, config:%s", err, *config)
	}
	transport_delegate.InitAllProtocol(cfg.TransportConfig)
	log.Printf("Config:%+v", cfg)
	cli := wcf.NewServer(cfg)
	if cli == nil {
		panic("could not create wcf server")
	}
	cli.Start()
}
