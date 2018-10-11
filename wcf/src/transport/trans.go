package transport

import (
	"net"
	"time"
	"fmt"
	"github.com/pkg/errors"
	"io/ioutil"
	"encoding/json"
)

type ParamPair struct {
	BindParam interface{}
	DialParam interface{}
}

var bindmp map[string]BindFunc
var dialmp map[string]DialFunc
var initmp map[string]InitFunc
var parammp map[string]*ParamPair

var bindnames []string
var dialnames []string

// 想了下, 这里貌似没必要加锁。。
// 由于go包的初始化init执行顺序跟文件名的字母序有关,
// 所以所有的协议必须要以trans_开头, 避免主init函数发生在子init函数之后导致GG
func init() {
	bindmp = make(map[string]BindFunc)
	dialmp = make(map[string]DialFunc)
	initmp = make(map[string]InitFunc)
	parammp = make(map[string]*ParamPair)
}

//注册bind函数
type BindFunc func(addr string, extra interface{}) (net.Listener, error)
//注册dial函数
type DialFunc func(addr string, timeout time.Duration, extra interface{}) (net.Conn, error)
//用于从json数据构建对应的参数
type InitFunc func(bindData []byte, dialData []byte) (interface{}, interface{}, error)

//bind, dial
func GetParam(pt string) (interface{}, interface{}) {
	v := parammp[pt]
	if v != nil {
		return v.BindParam, v.DialParam
	}
	return nil, nil
}

func InitAllProtocol(file string) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	mp := make(map[string]interface{})
	if err := json.Unmarshal(data, &mp); err != nil {
		return err
	}
	for k, v := range mp {
		fun := initmp[k]
		if fun == nil {
			continue
		}
		//偷下懒, 不知道有没有更好的实现
		kData := v.(map[string]interface{})
		var bindData []byte
		var dialData []byte
		if bd, ok := kData["bind"]; ok {
			if bindData, err = json.Marshal(bd); err != nil {
				return err
			}
		}
		if dd, ok := kData["dial"]; ok {
			if dialData, err = json.Marshal(dd); err != nil {
				return err
			}
		}
		b, d, err := fun(bindData, dialData)
		if err != nil {
			return errors.New(fmt.Sprintf("init pt:%s fail, err:%v", k, err))
		}
		parammp[k] = &ParamPair{b,d}
	}
	return nil
}

func Regist(pt string, bindFunc BindFunc, dialFunc DialFunc, initFunc InitFunc) {
	if bindFunc != nil {
		RegistBind(pt, bindFunc)
	}
	if dialFunc != nil {
		RegistDial(pt, dialFunc)
	}
	if initFunc != nil {
		RegistInit(pt, initFunc)
	}
}

func RegistBind(pt string, bindFunc BindFunc) {
	if _, ok := bindmp[pt]; ok {
		panic(fmt.Sprintf("bind protocol:%s already regist!", pt))
	}
	bindnames = append(bindnames, pt)
	bindmp[pt] = bindFunc
}

func RegistDial(pt string, dialFunc DialFunc) {
	if _, ok := dialmp[pt]; ok {
		panic(fmt.Sprintf("dial protocol:%s already regist!", pt))
	}
	dialnames = append(dialnames, pt)
	dialmp[pt] = dialFunc
}

func RegistInit(pt string, initFunc InitFunc) {
	if _, ok := initmp[pt]; ok {
		panic(fmt.Sprintf("init protocol:%s already regist!", pt))
	}
	initmp[pt] = initFunc
}

func GetAllBindName() []string {
	return bindnames
}

func GetAllDialName() []string {
	return dialnames
}

func Bind(pt string, addr string) (net.Listener, error) {
	if v, ok := bindmp[pt]; ok {
		param, _ := GetParam(pt)
		return v(addr, param)
	}
	return nil, errors.New(fmt.Sprintf("bind protocol:%s not regist", pt))
}

func Dial(pt string, addr string, timeout time.Duration) (net.Conn, error) {
	if v, ok := dialmp[pt]; ok {
		_, param := GetParam(pt)
		return v(addr, timeout, param)
	}
	return nil, errors.New(fmt.Sprintf("dial protocol:%s not regist", pt))
}

