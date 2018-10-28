package check

import (
	log "github.com/sirupsen/logrus"
	"limiter"
	"net"
	"sync"
	"time"
)

type CheckContext struct {
	Host  string
	Force bool
}

type SmartRule struct {
	domain     map[string]HostRule
	mu         sync.Mutex
	checkQueue chan CheckContext
	limit      *limiter.Limiter
}

func (this *SmartRule) startCheck() {
	var checkMap sync.Map
	for {
		checkRule := <-this.checkQueue
		force := checkRule.Force
		host := checkRule.Host
		name, _, serr := net.SplitHostPort(host)
		if serr != nil {
			log.Errorf("Check host, but could not split to name:port, skip, host:%s", host)
			continue
		}
		if _, ok := this.domain[name]; ok && !force {
			continue
		}
		if _, ok := checkMap.Load(name); ok { //in progress
			continue
		}
		used, result := this.limit.TryAcqure()
		log.Infof("Checking host:%s, acquire result:%t, used:%d, force:%t", name, result, used, force)
		checkMap.Store(name, true)
		if result {
			go func() {
				conn, err := net.DialTimeout("tcp", host, 1*time.Second)
				if err == nil {
					conn.Close()
				}
				this.limit.Release()
				var stat = RULE_DIRECT
				if err != nil {
					stat = RULE_PROXY
				}
				log.Infof("Add auto proxy rule, addr:%s, rule:%s, err:%v", name, HostRule2String(HostRule(stat)), err)
				this.AddRule(name, HostRule(stat))
				checkMap.Delete(name)
			}()
		}
	}
}

func NewSmartHost() *SmartRule {
	s := &SmartRule{domain: make(map[string]HostRule), checkQueue: make(chan CheckContext, 300), limit: limiter.NewLimiter(200)}
	go s.startCheck()
	return s
}

func (this *SmartRule) AddToCheck(addr string, force bool) {
	select {
	case this.checkQueue <- CheckContext{addr, force}:
		log.Infof("Add host to check queue succ, host:%s, force:%t, queue len:%d", addr, force, len(this.checkQueue))
		break
	default:
		log.Errorf("Check queue full, skip add host to check, host:%s", addr)
	}
}

func (this *SmartRule) AddRule(addr string, rule HostRule) {
	this.mu.Lock()
	defer this.mu.Unlock()
	this.domain[addr] = rule
}

func (this *SmartRule) CheckAndGetRule(addr string) (HostRule, bool) {
	this.mu.Lock()
	defer this.mu.Unlock()
	if v, ok := this.domain[addr]; ok {
		return v, true
	}
	return HostRule(defaultInfo), false
}
