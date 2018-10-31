package redirect

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
)

type ProcessFunc func(net.Conn, interface{}) (int64, int64, error)
type ParseFunc func(data []byte) (interface{}, error)

var processmap = make(map[string]ProcessFunc)
var parsemap = make(map[string]ParseFunc)
var parsedata = make(map[string]interface{})

func InitAll(file string) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return errors.New(fmt.Sprintf("read cfg file fail, err:%v", err))
	}
	all := make(map[string]interface{})
	err = json.Unmarshal(data, &all)
	if err != nil {
		return errors.New(fmt.Sprintf("parse cfg json fail, err:%v", err))
	}
	for k, v := range all {
		if parse, exist := parsemap[k]; exist {
			subData, _ := json.Marshal(v)
			parsed, err := parse(subData)
			if err != nil {
				return errors.New(fmt.Sprintf("parse sub config fail, protocol:%s, err:%s", k, err))
			}
			parsedata[k] = parsed
		}
	}
	return nil
}

func Regist(name string, parseFunc ParseFunc, processFunc ProcessFunc) {
	processmap[name] = processFunc
	parsemap[name] = parseFunc
}

func Process(name string, conn net.Conn) (int64, int64, error) {
	if v, ok := processmap[name]; ok {
		var extra interface{}
		if param, ok := parsedata[name]; ok {
			extra = param
		}
		return v(conn, extra)
	}
	err := errors.New(fmt.Sprintf("not found processor:%s", name))
	return 0, 0, err
}
