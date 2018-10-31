package check

import (
	"bufio"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"net_utils"
	"os"
	"reload"
	"sort"
	"strings"
	"sync"
)

const (
	RULE_BLOCK  = 0
	RULE_DIRECT = 1
	RULE_PROXY  = 2
)

func HostRule2String(rule HostRule) string {
	if rule == 0 {
		return "block"
	} else if rule == 1 {
		return "direct"
	} else if rule == 2 {
		return "proxy"
	} else {
		return "unknow"
	}
}

type HostRule int

var defaultInfo = RULE_PROXY

type CIDRRange struct {
	Start uint32
	End   uint32
	CIDRs []string
}

type RouteList []*CIDRRange

func (s RouteList) Len() int {
	return len(s)
}
func (s RouteList) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s RouteList) Less(i, j int) bool {
	if s[i].Start == s[j].Start {
		return s[i].End < s[j].End
	}
	return s[i].Start < s[j].Start
}

type Rule struct {
	file    string
	domain  map[string]HostRule
	cidr    map[*net.IPNet]HostRule
	routeV4 map[int]RouteList
	mu      sync.RWMutex
}

func sortAndMerge(lst RouteList) RouteList {
	if len(lst) <= 1 {
		return lst
	}
	sort.Sort(lst)
	var tmp RouteList
	tmp = append(tmp, lst[0])
	for i := 1; i < len(lst); i++ {
		back := tmp[len(tmp)-1]
		if lst[i].Start <= back.End {
			if lst[i].End > back.End {
				back.End = lst[i].End
			}
			back.CIDRs = append(back.CIDRs, lst[i].CIDRs...)
		} else {
			tmp = append(tmp, lst[i])
		}
	}
	return tmp
}

func (this *Rule) onCheck(addr string, v interface{}) (bool, interface{}) {
	return reload.DefaultFileCheckModFunc(addr, v)
}

func (this *Rule) onLoad(addr string) (interface{}, error) {
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
	domain := make(map[string]HostRule)
	cidr := make(map[*net.IPNet]HostRule)
	routeV4 := make(map[int]RouteList)
	tmp["domain"] = domain
	tmp["cidr"] = cidr
	tmp["route_v4"] = routeV4
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
		rt := HostRule(op)
		if strings.Contains(addr, "/") { //cidr
			ip, cidritem, err := net.ParseCIDR(addr)
			if err != nil {
				log.Errorf("Parse cidr fail, err:%v, addr:%s, line:%s", err, addr, line)
				continue
			}
			if ip.To4() == nil { //v6
				cidr[cidritem] = rt
			} else { //v4
				st, ed := net_utils.ResolveIPRange(addr)
				routeV4[op] = append(routeV4[op], &CIDRRange{st, ed, []string{addr}})
			}
		} else { //ip
			domain[addr] = rt
		}
	}
	for rule, lst := range routeV4 {
		routeV4[rule] = sortAndMerge(lst)
	}
	return tmp, nil
}

func (this *Rule) onLoadSucc(addr string, result interface{}, err error) {
	if err == nil {
		this.mu.Lock()
		defer this.mu.Unlock()
		mp := result.(map[string]interface{})
		this.domain = mp["domain"].(map[string]HostRule)
		this.cidr = mp["cidr"].(map[*net.IPNet]HostRule)
		this.routeV4 = mp["route_v4"].(map[int]RouteList)
		log.Infof("Reload host from file:%s, success, domain:%d, v6cidr:%d, v4route:[block:%d, direct:%d, proxy:%d]", addr, len(this.domain), len(this.cidr), len(this.routeV4[0]), len(this.routeV4[1]), len(this.routeV4[2]))
	} else {
		log.Errorf("Reload host from file:%s fail, err:%v", this.file, err)
	}
}

func NewRule(file string) (*Rule, error) {
	r := &Rule{file: file, domain: make(map[string]HostRule), cidr: make(map[*net.IPNet]HostRule)}
	if len(file) == 0 {
		return r, nil
	}
	_, err := os.Stat(file)
	if err != nil {
		return nil, err
	}
	rd := reload.New()
	err, wg := rd.AddLoad(
		r.onCheck,
		r.onLoad,
		r.onLoadSucc,
		file,
	)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("add host reload item fail, err:%v", err))
	}
	rd.Start()
	wg.Wait()
	return r, nil
}

func (this *Rule) GetHostRuleOptional(addr string, safe bool) HostRule {
	v, _ := this.CheckAndGetRuleOptional(addr, safe)
	return v
}

func (this *Rule) GetHostRule(addr string) HostRule {
	return this.GetHostRuleOptional(addr, true)
}

func checkIPInRangeList(ip uint32, lst RouteList) bool {
	var (
		start = 0
		end   = len(lst) - 1
		mid   = 0
	)
	for start <= end {
		mid = (start + end) / 2
		if lst[mid].Start <= ip && ip <= lst[mid].End {
			return true
		}
		if ip > lst[mid].Start {
			start = mid + 1
		} else if ip < lst[mid].End {
			end = mid - 1
		}
	}
	return false
}

func (this *Rule) CheckAndGetRuleOptional(addr string, safe bool) (HostRule, bool) {
	addr = strings.Trim(addr, "\r\n\t ")
	if strings.HasPrefix(addr, "[") && strings.HasSuffix(addr, "]") {
		addr = addr[1 : len(addr)-1]
	}
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
				tmpaddr = tmpaddr[index+1:]
			}
			if safe { //正常来说, 客户端做这个检查毫无意义, 除了拖慢速度, 所以弄成可选的。
				newip, err := net.ResolveIPAddr("ip", addr)
				if err != nil {
					log.Errorf("Resolve ip addr err, domain addr:%s, err:%v", addr, err)
					break
				}
				ip = newip.IP
			} else {
				break
			}
		}
		if v, ok := this.domain[addr]; ok {
			return v, true
		}
		if v4 := ip.To4(); v4 != nil { //v4
			for op, lst := range this.routeV4 {
				if checkIPInRangeList(net_utils.InetAtoNBytes(v4), lst) {
					return HostRule(op), true
				}
			}
		} else { //v6
			for k, v := range this.cidr {
				if k.Contains(ip) {
					return v, true
				}
			}
		}
		break
	}
	return HostRule(defaultInfo), false
}

func (this *Rule) CheckAndGetRule(addr string) (HostRule, bool) {
	return this.CheckAndGetRuleOptional(addr, true)
}
