package wcf

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"time"
)

type ProxyAddrConfig struct {
	Protocol string `json:"protocol"`
	Address  string `json:"address"`
}

type ErrRedirectAddress struct {
	Protocol string `json:"protocol"`
	Address  string `json:"address"`
}

type ReportVisitConfig struct {
	Enable        bool   `json:"enable"`
	Visitor       string `json:"visitor"`
	VisitorConfig string `json:"visitor_config"`
}

type RedirectorConfig struct {
	Enable         bool   `json:"enable"`
	Redirector     string `json:"redirector"`
	RedirectConfig string `json:"redirect_config"`
}

type ServerConfig struct {
	Localaddr       []ProxyAddrConfig `json:"localaddr"`
	Userinfo        string            `json:"userinfo"`
	Timeout         time.Duration     `json:"timeout"`
	Encrypt         string            `json:"encrypt"`
	Key             string            `json:"key"`
	Host            string            `json:"host"`
	TransportConfig string            `json:"transport"`
	Redirect        RedirectorConfig  `json:"redirect"`
	ReportVisit     ReportVisitConfig `json:"report"`
}

func NewServerConfig() *ServerConfig {
	return &ServerConfig{}
}

func (this *ServerConfig) Parse(file string) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return errors.New(fmt.Sprintf("read json file fail, err:%v, file:%s", err, file))
	}

	err = json.Unmarshal(data, this)
	if err == nil {
		this.Timeout = this.Timeout * time.Second
	}
	return err
}
