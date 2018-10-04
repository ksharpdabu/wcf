package wcf

import (
	"time"
	"errors"
	"io/ioutil"
	"fmt"
	"encoding/json"
)

type ServerConfig struct {
	Localaddr string         `json:"localaddr"`
	Userinfo  string         `json:"userinfo"`
	Timeout   time.Duration  `json:"timeout"`
	Encrypt   string         `json:"encrypt"`
	Key       string         `json:"key"`
	EnableSecureCheck bool   `json:"secure_check"`
}

func NewServerConfig() *ServerConfig {
	return &ServerConfig{}
}

func(this *ServerConfig) Parse(file string) error {
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