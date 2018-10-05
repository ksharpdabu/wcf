package wcf

import (
	"time"
	"io/ioutil"
	"encoding/json"
)

type HostInfo struct {
	BlackHost string     `json:"black"`
	NoProxyHost string   `json:"no_proxy"`
}

type AddrConfig struct {
	Name string          `json:"name"`
	Address string       `json:"address"`
}

type LoadBalanceInfo struct {
	MaxErrCnt int                `json:"max_errcnt"`
	MaxFailTime time.Duration    `json:"max_failtime"`
}

type ProxyAddrInfo struct {
	Addr string  `json:"addr"`
	Weight int   `json:"weight"`
}

type LocalConfig struct {
	Localaddr       []AddrConfig  `json:"localaddr"` //map[string]string `json:"localaddr"`
	Proxyaddr       []ProxyAddrInfo `json:"proxyaddr"`
	Timeout         time.Duration `json:"timeout"`
	User            string `json:"user"`
	Pwd             string `json:"pwd"`
	Host            HostInfo `json:"host"`
	Encrypt         string   `json:"encrypt"`
	Key             string   `json:"key"`
	Lbinfo          LoadBalanceInfo `json:"loadbalance"`
}

func NewLocalConfig() *LocalConfig {
	return &LocalConfig{}
}

func(this *LocalConfig) Parse(file string) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, this)
	if err == nil {
		this.Timeout = this.Timeout * time.Second
		this.Lbinfo.MaxFailTime = this.Lbinfo.MaxFailTime * time.Second
	}
	return err
}

func loadHostFromJson(mp map[string]bool, v []interface{}) (error) {
	for i := 0; i < len(v); i++ {
		item := v[i]
		mp[item.(string)] = true
	}
	return nil
}