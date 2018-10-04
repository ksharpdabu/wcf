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

type Host struct {
	file string
	mp map[string]bool
	mu sync.RWMutex
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

func NewRule(file string) (*Host, error) {
	r := &Host{file:file, mp:make(map[string]bool)}
	if len(file) == 0 {
		return r, nil
	}
	_, err := os.Stat(file)
	if err != nil {
		return nil, err
	}
	reload.AddLoad(
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
			tmp := make(map[string]bool)
			for {
				line, _, err := r.ReadLine()
				if err == io.EOF {
					break
				}
				if err != nil {
					log.Errorf("Read line from file:%s fail, err:%s", addr, err)
					return nil, err
				}
				log.Debugf("Read host:%s from file", string(line))
				tmp[string(line)] = true
			}
			return tmp, nil
		},
		func(addr string, result interface{}, err error) {
			if err == nil {
				r.mu.Lock()
				defer r.mu.Unlock()
				r.mp = result.(map[string]bool)
			}
			log.Infof("Reload host from file:%s, success, size:%d", addr, len(result.(map[string]bool)))
		},
		file,
	)
	return r, nil
}

func IsIP(url string) bool {
	ip := net.ParseIP(url)
	return ip != nil
}

func GetRootDomain(url string) string {
	reg := regexp.MustCompile("[0-9A-Za-z]+\\.[a-zA-Z]+$")
	lst := reg.FindAllString(url, -1)
	if len(lst) == 1 {
		 return lst[0]
	}
	return url
}

func GetAllSubDomainPoint(url string) []int {
	if IsIP(url) {
		return []int {0}
	}
	var pt []int
	cnt := 10
	first := true
	for i := len(url) - 1; i > 0 && cnt > 0; i-- {
		if url[i] == '.' {
			cnt--
			if first == true {
				first = false
				continue
			}
			pt = append(pt, i + 1)
		}
	}
	pt = append(pt, 0)
	return pt
}

func BuildSubDomainByPoint(url string, pt []int) []string {
	ret := make([]string, len(pt))
	for i := 0; i < len(pt); i++ {
		ret[i] = url[pt[i]:]
	}
	return ret
}

func(this *Host) IsSubExists(url string) bool {
	if IsIP(url) {
		return this.IsExists(url)
	}
	pts := GetAllSubDomainPoint(url)
	for _, pt := range pts {
		if this.IsExists(url[pt:]) {
			return true
		}
	}
	return false
}

func(this *Host) IsExists(url string) bool {
	this.mu.RLock()
	defer this.mu.RUnlock()
	_, ok := this.mp[url]
	return ok
}