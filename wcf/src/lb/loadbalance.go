package lb

import (
	"errors"
	"math/rand"
	"sync"
	"time"
)

//一个简易的负载均衡组件, 实现的比较挫, 考虑到每秒建立连接数的数量非常地低, 所以这个工具还是勉强能用的=.=
//最小的权重值只能为1.。。
type LoadItem struct {
	Current  int
	Base     int
	MarkFail bool
	Errtime  int
	LastFail time.Time
	Extra    interface{}
}

type LoadBalance struct {
	mu      sync.Mutex
	mp      map[string]*LoadItem
	maxErr  int
	maxFail time.Duration
	rnd     *rand.Rand
}

func (this *LoadBalance) autoScan() {
	for {
		this.mu.Lock()
		for _, v := range this.mp {
			if v.MarkFail {
				if v.LastFail.Add(this.maxFail).Before(time.Now()) {
					v.MarkFail = false
					v.LastFail = time.Time{}
					v.Errtime = 0
					v.Current = v.Base/2 + 1
				}
			}
		}
		this.mu.Unlock()

		time.Sleep(10 * time.Second)
	}
}

func New(maxErrTime int, maxFailTime time.Duration) *LoadBalance {
	l := &LoadBalance{mp: make(map[string]*LoadItem), maxErr: maxErrTime, maxFail: maxFailTime}
	l.rnd = rand.New(rand.NewSource(time.Now().Unix()))
	go l.autoScan()
	return l
}

func (this *LoadBalance) Add(addr string, weight int, extra interface{}) {
	this.mu.Lock()
	defer this.mu.Unlock()
	item := &LoadItem{}
	item.Base = weight
	item.Current = weight
	if item.Current <= 0 {
		item.Current = 1
	}
	item.Errtime = 0
	item.LastFail = time.Time{}
	item.Extra = extra
	item.MarkFail = false
	this.mp[addr] = item
}

func (this *LoadBalance) Get() (string, interface{}, error) {
	this.mu.Lock()
	defer this.mu.Unlock()
	var total int = 0
	for _, v := range this.mp {
		if v.MarkFail {
			continue
		}
		total += v.Current
	}
	if total <= 0 {
		return "", nil, errors.New("all ip fail")
	}
	rnd := this.rnd.Intn(total)
	for k, v := range this.mp {
		if v.MarkFail {
			continue
		}
		rnd -= v.Current
		if rnd <= 0 {
			if v.Current > v.Base/2 {
				v.Current--
			}
			if v.Current <= 0 {
				v.Current = 1
			}
			return k, v.Extra, nil
		}
	}
	panic("should not reach here..")
}

func (this *LoadBalance) Update(addr string, result bool) {
	this.mu.Lock()
	defer this.mu.Unlock()
	if v, ok := this.mp[addr]; ok {
		//succ
		if result {
			if v.MarkFail {
				v.MarkFail = false
				v.Current += v.Base/5 + 1
			} else {
				if v.Current <= v.Base/2 {
					v.Current += v.Base/5 + 1
				} else {
					v.Current++
				}
			}
			if v.Current > v.Base {
				v.Current = v.Base
			}
			return
		}
		//fail
		if v.MarkFail {
			return
		}
		v.Errtime++
		if v.Errtime >= this.maxErr {
			v.MarkFail = true
			v.LastFail = time.Now()
		}
		v.Current = v.Current/2 - v.Base/10
		if v.Current <= 0 {
			v.Current = v.Base/10 + 1
		}
		return
	}

}
