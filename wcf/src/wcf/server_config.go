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

type ServerConfig struct {
	Localaddr       []ProxyAddrConfig    `json:"localaddr"`
	Userinfo        string               `json:"userinfo"`
	Timeout         time.Duration        `json:"timeout"`
	Encrypt         string               `json:"encrypt"`
	Key             string               `json:"key"`
	Host            string               `json:"host"`
	TransportConfig string               `json:"transport"`
	ErrConnect      []ErrRedirectAddress `json:"err_redirect"`
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
