package check

import (
	"os"
	log "github.com/sirupsen/logrus"
	"bufio"
	"io"
	"sync"
	"net"
	"reload"
	"strings"
	"errors"
	"fmt"
)

const (
	RULE_BLOCK = 0
	RULE_DIRECT = 1
	RULE_PROXY = 2
)

type RouteInfo struct {
	HostRule    int
	NewHostValue   string
}

var defaultInfo = &RouteInfo{ RULE_PROXY, "" }

type Rule struct {
	file string
	domain map[string]*RouteInfo
	cidr map[*net.IPNet]*RouteInfo
	mu sync.RWMutex
}

func NewRule(file string) (*Rule, error) {
	r := &Rule{file:file, domain:make(map[string]*RouteInfo), cidr:make(map[*net.IPNet]*RouteInfo)}
	if len(file) == 0 {
		return r, nil
	}
	_, err := os.Stat(file)
	if err != nil {
		return nil, err
	}
	rd := reload.New()
	err, wg := rd.AddLoad(
		func(addr string, v interface{}) (bool, interface{}) {
			return reload.DefaultFileCheckModFunc(addr, v)
		},
		func(addr string) (interface{}, error) {
			file, err := os.Open(addr)
			if err != nil {
				log.Errorf("Open file:%s fail, err:%v", addr, err)
				return nil, err
			}
			defer func() {
				file.Close()
			}()
			r := bufio.NewReader(file)
			tmp := make(map[string]interface{})
			domain := make(map[string]*RouteInfo)
			cidr := make(map[*net.IPNet]*RouteInfo)
			tmp["domain"] = domain
			tmp["cidr"] = cidr
			for {
				bline, _, err := r.ReadLine()
				if err == io.EOF {
					break
				}
				line := strings.ToLower(strings.Trim(string(bline), "\t \n\r"))
				if strings.HasPrefix(line, "#") || len(line) == 0 {
					continue
				}
				sp := strings.Split(line, ",")
				if len(sp) < 2 {
					log.Errorf("Invalid rule line:%s, format should like this `domain,op_type[,replace host]`", line)
					continue
				}
				addr := sp[0]
				op := 0
				extra := ""
				switch sp[1] {
				case "proxy":
					op = RULE_PROXY
					break
				case "direct":
					op = RULE_DIRECT
					break
				case "block":
					op = RULE_BLOCK
					break
				}
				if len(sp) == 3 {
					extra = sp[2]
				}
				rt := &RouteInfo{HostRule:op, NewHostValue:extra}
				if strings.Contains(addr, "/") { //cidr
					_, cidritem, err := net.ParseCIDR(addr)
					if err != nil {
						log.Errorf("Parse cidr fail, err:%v, addr:%s, line:%s", err, addr, line)
						continue
					}
					cidr[cidritem] = rt
				} else { //ip
					domain[addr] = rt
				}
			}
			return tmp, nil
		},
		func(addr string, result interface{}, err error) {
			if err == nil {
				r.mu.Lock()
				defer r.mu.Unlock()
				mp := result.(map[string]interface{})
				r.domain = mp["domain"].(map[string]*RouteInfo)
				r.cidr = mp["cidr"].(map[*net.IPNet]*RouteInfo)
			}
			log.Infof("Reload host from file:%s, success, domain:%d, cidr:%d", addr, len(r.domain), len(r.cidr))
		},
		file,
	)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("add host reload item fail, err:%v", err))
	}
	rd.Start()
	wg.Wait()
	return r, nil
}

func(this *Rule) GetHostRuleOptional(addr string, safe bool) (*RouteInfo) {
	v, _ := this.CheckAndGetRuleOptional(addr, safe)
	return v
}

func(this *Rule) GetHostRule(addr string) (*RouteInfo) {
	return this.GetHostRuleOptional(addr, true)
}

func(this *Rule) CheckAndGetRuleOptional(addr string, safe bool) (*RouteInfo, bool) {
	this.mu.RLock()
	defer this.mu.RUnlock()
	ip := net.ParseIP(addr)
	for {
		if ip == nil {
			tmpaddr := addr
			for {
				if v, ok := this.domain[tmpaddr]; ok {
					return v, true
				}
				index := strings.Index(tmpaddr, ".")
				if index < 0 {
					break
				}
				tmpaddr = tmpaddr[index + 1:]
			}
			if safe {  //正常来说, 客户端做这个检查毫无意义, 除了拖慢速度, 所以弄成可选的。
				newip, err := net.ResolveIPAddr("ip", addr)
				if err != nil {
					log.Errorf("Resolve ip addr err, domain addr:%s, err:%v", addr, err)
					break
				}
				ip = newip.IP
			}
		}
		if v, ok :=this.domain[addr]; ok {
			return v, true
		}
		for k, v := range this.cidr {
			if k.Contains(ip){
				return v, true
			}
		}
		break
	}
	return defaultInfo, false
}

func(this *Rule) CheckAndGetRule(addr string) (*RouteInfo, bool) {
	return this.CheckAndGetRuleOptional(addr, true)
}
