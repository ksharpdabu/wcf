package wcf

import (
	"time"
	"io/ioutil"
	"encoding/json"
)

type RedirectConfig struct {
	Localaddr string `json:"localaddr"`
	Proxyaddr string `json:"proxyaddr"`
	Timeout time.Duration `json:"timeout"`
}

func NewRedirectConfig() *RedirectConfig {
	return &RedirectConfig{}
}

func(this *RedirectConfig) Parse(file string) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, this)
	if err != nil {
		return err
	}
	this.Timeout = this.Timeout * time.Second
	return nil
}
