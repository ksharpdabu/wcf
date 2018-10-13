package wcf

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"reload"
	"sync"
)

type ForwardInfo struct {
	EnableForward bool   `json:"enable"`
	ForwardAddr   string `json:"address"`
}

type UserInfo struct {
	User          string      `json:"user"`
	Pwd           string      `json:"pwd"`
	Forward       ForwardInfo `json:"forward"`
	MaxConnection int         `json:"max_conn"`
	ConnLimiter   Limiter
}

type UserHolder struct {
	mu       sync.RWMutex
	userinfo map[string]*UserInfo
	file     string
}

func ReadAllLine(f string) ([][]byte, error) {
	file, err := os.Open(f)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("open file:%s for read fail, err:%v", f, err))
	}
	defer func() {
		file.Close()
	}()
	r := bufio.NewReader(file)
	var tmp [][]byte
	for {
		line, _, err := r.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.New(fmt.Sprintf("read line fail, err:%v, file:%s", err, f))
		}
		tmp = append(tmp, line)
	}
	return tmp, nil
}

func NewUserHolder(file string) (*UserHolder, error) {
	r := &UserHolder{file: file, userinfo: make(map[string]*UserInfo)}
	if len(file) == 0 {
		return r, nil
	}
	_, err := os.Stat(file)
	if err != nil {
		return nil, err
	}
	rd := reload.New()
	err, gp := rd.AddLoad(
		func(addr string, v interface{}) (bool, interface{}) {
			return reload.DefaultFileCheckModFunc(addr, v)
		},
		func(addr string) (interface{}, error) {
			lines, err := ReadAllLine(addr)
			if err != nil {
				log.Errorf("Read all line fail, file:%s, err:%v", addr, err)
				return nil, err
			}
			tmp := make(map[string]*UserInfo)
			for index, line := range lines {
				ui := &UserInfo{}
				err := json.Unmarshal(line, ui)
				if err != nil {
					log.Errorf("Parse user json fail, err:%v, line:%d, data:%s", err, index, string(line))
					return nil, err
				}
				if ui.MaxConnection == 0 {
					ui.MaxConnection = 200
				}
				log.Infof("Read user:%+v from file", ui)
				ui.ConnLimiter.Reset(ui.MaxConnection, ui.MaxConnection)
				tmp[ui.User] = ui
			}
			return tmp, nil
		},
		func(addr string, result interface{}, err error) {
			if err == nil {
				r.mu.Lock()
				defer r.mu.Unlock()
				//简单处理, 更新部分用户数据的时候重置所有用户数据, 不然要写到吐血
				r.userinfo = result.(map[string]*UserInfo)
			}
			log.Infof("Reload userinfo from file:%s, success, size:%d", addr, len(result.(map[string]*UserInfo)))
		},
		file,
	)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("add reload item fail, err:%v", err))
	}
	rd.Start()
	gp.Wait()
	return r, nil
}

func (this *UserHolder) GetUserInfo(name string) *UserInfo {
	this.mu.RLock()
	defer this.mu.RUnlock()
	if v, ok := this.userinfo[name]; ok {
		return v
	}
	return nil
}

func (this *UserHolder) Check(name, pwd string) bool {
	ui := this.GetUserInfo(name)
	if ui != nil {
		if ui.Pwd == pwd {
			return true
		}
	}
	return false
}
