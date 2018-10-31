package limiter

import "sync"

type Limiter struct {
	cnt  int
	base int
	lock sync.Mutex
}

//used, could acqure
func (this *Limiter) TryAcqure() (int, bool) {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.cnt > 0 {
		this.cnt--
		return this.base - this.cnt, true
	}
	return 0, false
}

func (this *Limiter) Release() int {
	this.lock.Lock()
	defer this.lock.Unlock()
	if this.cnt < this.base {
		this.cnt++
	}
	return this.cnt
}

func (this *Limiter) GetCur() int {
	return this.cnt
}

func (this *Limiter) GetBase() int {
	return this.base
}

func (this *Limiter) Reset(cur int, base int) {
	this.lock.Lock()
	defer this.lock.Unlock()
	this.cnt = cur
	this.base = base
}

func NewLimiter(cnt int) *Limiter {
	return &Limiter{cnt: cnt, base: cnt}
}
