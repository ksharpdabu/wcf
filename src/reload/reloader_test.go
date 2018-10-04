package reload

import "testing"
import (
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"time"
)

func Check(addr string, v interface{}) (bool, interface{}) {
	return DefaultFileCheckModFunc(addr, v)
}

func DataLoad(addr string) (interface{}, error) {
	data, err := ioutil.ReadFile(addr)
	return data, err
}

func DataLoadFinish(addr string, data interface{}, err error) {
	log.Infof("Load file:%s success, data:%v, err:%v", addr, data, err)
}

func TestAutoReload(t *testing.T) {
	AddLoad(Check, DataLoad, DataLoadFinish, "D:/GoPath/src/wcf/cmd/server/userinfo.dat")
	time.Sleep(2 * time.Minute)
}

