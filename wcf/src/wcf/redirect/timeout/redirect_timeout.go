package timeout

import (
	"encoding/json"
	"math/rand"
	"net"
	"time"
	"wcf/redirect"
)

func init() {
	redirect.Regist("timeout", ParseTimeoutParam, ProcessTimeout)
}

type TimeoutParam struct {
	MinDuration int64 `json:"min_duration"`
	MaxDuration int64 `json:"max_duration"`
}

func ParseTimeoutParam(data []byte) (interface{}, error) {
	param := &TimeoutParam{}
	err := json.Unmarshal(data, param)
	if err != nil {
		return nil, err
	}
	if param.MinDuration < 0 {
		param.MinDuration = 0
	}
	if param.MaxDuration < 0 {
		param.MaxDuration = 0
	}
	if param.MinDuration > param.MaxDuration {
		param.MinDuration = param.MaxDuration
	}
	return param, nil
}

func ProcessTimeout(conn net.Conn, extra interface{}) (int64, int64, error) {
	param := extra.(*TimeoutParam)
	var ts int64
	if param.MaxDuration == param.MinDuration {
		ts = param.MaxDuration
	} else {
		ts = param.MinDuration + rand.Int63n(param.MaxDuration-param.MinDuration)
	}
	if ts != 0 {
		dur := time.Duration(ts) * time.Second
		time.Sleep(dur)
	}
	return 0, 0, nil
}
