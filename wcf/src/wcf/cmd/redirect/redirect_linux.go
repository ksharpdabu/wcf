package main

import (
	"flag"
	log "github.com/sirupsen/logrus"
	"net/http"
	_ "net/http/pprof"
	"wcf"
)

var config *string = flag.String("config", "D:/GoProj/wcf_proj/src/wcf/cmd/redirect/redirect.json", "config file")

func main() {
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	log.SetFormatter(customFormatter)
	customFormatter.FullTimestamp = true
	go func() {
		http.ListenAndServe("localhost:6062", nil)
	}()
	flag.Parse()
	cfg := wcf.NewRedirectConfig()
	err := cfg.Parse(*config)
	if err != nil {
		log.Fatalf("Read config fail, err:%v, config:%s", err, *config)
	}
	log.Printf("Config:%+v", cfg)
	cli := wcf.NewRedirect(cfg)
	cli.Start()
}
