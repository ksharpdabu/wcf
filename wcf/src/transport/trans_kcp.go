package transport

import (
	"encoding/json"
	"github.com/xtaci/kcp-go"
	"net"
	"time"
)

type kcpCfg struct {
	DataShards   int `json:"data_shards"`
	ParityShards int `json:"parity_shards"`
}

func initKcpConfig(bindData []byte, dialData []byte) (interface{}, interface{}, error) {
	var bi interface{}
	var di interface{}
	if bindData != nil {
		bd := &kcpCfg{}
		if err := json.Unmarshal(bindData, bd); err != nil {
			return nil, nil, err
		}
		bi = bd
	}
	if dialData != nil {
		dd := &kcpCfg{}
		if err := json.Unmarshal(dialData, dd); err != nil {
			return nil, nil, err
		}
		di = dd
	}
	return bi, di, nil
}

func init() {
	Regist("kcp", func(addr string, extra interface{}) (net.Listener, error) {
		cfg := extra.(*kcpCfg)
		return kcp.ListenWithOptions(addr, nil, cfg.DataShards, cfg.ParityShards)
	}, func(addr string, timeout time.Duration, extra interface{}) (net.Conn, error) {
		cfg := extra.(*kcpCfg)
		return kcp.DialWithOptions(addr, nil, cfg.DataShards, cfg.ParityShards)
	}, initKcpConfig)
}
