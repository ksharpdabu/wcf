package raw

import (
	"net"
	"wcf/redirect"
	"encoding/json"
	"transport_delegate"
	"time"
	"errors"
	"fmt"
	"net_utils"
	"golang.org/x/net/context"
)

type RawParam struct {
	Protocol string `json:"protocol"`
	Target   string `json:"target"`
}

func init() {
	redirect.Regist("raw", ParseRawParam, ProcessRaw)
}

func ParseRawParam(data []byte) (interface{}, error) {
	param := &RawParam{}
	err := json.Unmarshal(data, param)
	if err != nil {
		return nil, err
	}
	return param, nil
}

func ProcessRaw(conn net.Conn, extra interface{}) (int64, int64, error) {
	param := extra.(*RawParam)
	target, err, _ := transport_delegate.Dial(param.Protocol, param.Target, 5 * time.Second)
	if err != nil {
		return 0, 0, errors.New(fmt.Sprintf("dial to target:%s fail, protocol:%s, err:%v", param.Target, param.Protocol, err))
	}
	defer target.Close()
	sbuf := make([]byte, 16 * 1024)
	dbuf := make([]byte, 16 * 1024)
	ctx, cancel := context.WithCancel(context.Background())
	sr, sw, _, _, sre, swe, dre, dwe := net_utils.Pipe(conn, target, sbuf, dbuf, ctx, cancel, 2 * time.Second)
	var rerr error
	if sre != nil || swe != nil || dre != nil || dwe != nil {
		rerr = errors.New(fmt.Sprintf("sre:%v, swe:%v, dre:%v, dwe:%v", sre, swe, dre, dwe))
	}
	return int64(sr), int64(sw), rerr
}