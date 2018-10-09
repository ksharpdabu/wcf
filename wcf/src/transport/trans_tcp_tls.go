package transport

import (
	"net"
	"crypto/tls"
	"time"
	"encoding/json"
)

type tlsBindCfg struct {
	PemFile string `json:"pem_file"`
	KeyFile string `json:"key_file"`
}

type tlsDialCfg struct {
	SkipInsecure bool `json:"skip_insecure"`
}

func initTlsConfig(bindData []byte, dialData []byte) (interface{}, interface{}, error) {
	var bi interface{}
	var di interface{}
	if bindData != nil {
		bd := &tlsBindCfg{}
		if err := json.Unmarshal(bindData, bd); err != nil {
			return nil, nil, err
		}
		bi = bd
	}
	if dialData != nil {
		dd := &tlsDialCfg{}
		if err := json.Unmarshal(dialData, dd); err != nil {
			return nil, nil, err
		}
		di = dd
	}
	return bi, di, nil
}

func init() {
	Regist("tcp_tls", func(addr string, extra interface{}) (net.Listener, error) {
		bd := extra.(*tlsBindCfg)
		cert, err := tls.LoadX509KeyPair(bd.PemFile, bd.KeyFile)
		if err != nil {
			return nil, err
		}
		cfg := &tls.Config{Certificates: []tls.Certificate{cert}}
		return tls.Listen("tcp", addr, cfg)
	}, func(addr string, timeout time.Duration, extra interface{}) (net.Conn, error) {
		dd := extra.(*tlsDialCfg)
		cfg := &tls.Config{
			InsecureSkipVerify: dd.SkipInsecure,
		}
		return tls.Dial("tcp", addr, cfg)
	}, initTlsConfig)
}
