package check

import (
	"os"
	log "github.com/sirupsen/logrus"
	"bufio"
	"io"
	"sync"
	"net"
	"regexp"
	"reload"
	"strconv"
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

func(this *Rule) GetHostRule(addr string) *RouteInfo {
	this.mu.RLock()
	defer this.mu.RUnlock()
	ip := net.ParseIP(addr)
	if ip == nil { //domain
		for {
			if v, ok := this.domain[addr]; ok {
				return v
			}
			index := strings.Index(addr, ".")
			if index < 0 {
				break
			}
			addr = addr[index + 1:]
		}
	} else { //v4
		if v, ok :=this.domain[addr]; ok {
			return v
		}
		for k, v := range this.cidr {
			if k.Contains(ip){
				return v
			}
		}
	}
	return defaultInfo
}

//schema, domain, port
func GetUrlInfo(req string) (error, string, string, int) {
	req = strings.ToLower(req)
	reg := regexp.MustCompile("^(https?)?(://)?([-a-zA-Z0-9\\.]+):?(\\d*).*")
	parsed := reg.FindStringSubmatch(req)
	if len(parsed) != 5 {
		return errors.New(fmt.Sprintf("parse fail, data:%v", parsed)), "", "", 0
	}
	schema := parsed[1]
	host := parsed[3]
	sport := parsed[4]
	if len(host) == 0 {
		return errors.New(fmt.Sprintf("invalid host, url:%s", req)), "", "", 0
	}
	var port = 80
	if len(sport) != 0 {
		tmp, e := strconv.ParseInt(sport, 10, 16)
		if e != nil {
			return errors.New(fmt.Sprintf("parse port fail, portstr:%s, url:%s", sport, req)), "", "", 0
		}
		port = int(tmp)
	} else {
		if len(schema) == 0 || schema == "http" {
			port = 80
		} else if schema == "https" {
			port = 443
		} else {
			return errors.New(fmt.Sprintf("invalid schema:%s, url:%s", schema, req)), "", "", 0
		}
	}
	return nil, schema, host, port
}