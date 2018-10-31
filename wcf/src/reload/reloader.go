package reload

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"time"
)

//on err return false
type CheckModFunc func(addr string, v interface{}) (bool, interface{})
type DataLoadFunc func(addr string) (interface{}, error)
type LoadFinishFunc func(addr string, result interface{}, err error)

func DefaultFileCheckModFunc(addr string, v interface{}) (bool, interface{}) {
	var md time.Time
	if v != nil {
		md = v.(time.Time)
	}
	s, err := os.Stat(addr)
	if err != nil {
		return false, nil
	}
	if s.ModTime().After(md) {
		return true, s.ModTime()
	}
	return false, nil
}

type ReloadInfo struct {
	cm      CheckModFunc
	dl      DataLoadFunc
	lf      LoadFinishFunc
	file    string
	cmParam interface{}
	wg      *sync.WaitGroup
}

type AutoReload struct {
	mu       sync.RWMutex
	store    map[string]*ReloadInfo
	duration time.Duration
	//
	pmu     sync.Mutex
	peeding []*ReloadInfo
}

var defaultLoader *AutoReload

func init() {
	defaultLoader = New()
	defaultLoader.Start()
}

func (this *AutoReload) Start() {
	go func() {
		for {
			this.pmu.Lock()
			if len(this.peeding) != 0 {
				for _, info := range this.peeding {
					if _, ok := this.store[info.file]; ok {
						panic(fmt.Sprintf("should not add same name file:%s", info.file))
					}
					this.store[info.file] = info
				}
				this.peeding = nil
			}
			this.pmu.Unlock()
			this.mu.Lock()
			for _, info := range this.store {
				if ok, param := info.cm(info.file, info.cmParam); ok {
					data, err := info.dl(info.file)
					info.lf(info.file, data, err)
					if err == nil {
						info.cmParam = param
					}
					if info.wg != nil {
						info.wg.Done()
						info.wg = nil
					}
				}
			}
			this.mu.Unlock()
			time.Sleep(this.duration)
		}
	}()
}

func New() *AutoReload {
	loader := &AutoReload{}
	loader.store = make(map[string]*ReloadInfo)
	loader.duration = 5 * time.Second
	return loader
}

func (this *AutoReload) SetDuration(ts time.Duration) {
	this.duration = ts
}

func (this *AutoReload) AddLoadSync(cm CheckModFunc, dl DataLoadFunc, lf LoadFinishFunc, file string) (error, *sync.WaitGroup) {
	this.mu.Lock()
	defer this.mu.Unlock()
	if _, ok := this.store[file]; ok {
		return errors.New("file already exists, skip"), nil
	}
	ri := &ReloadInfo{cm, dl, lf, file, nil, &sync.WaitGroup{}}
	ri.wg.Add(1)
	this.store[file] = ri
	return nil, ri.wg
}

func (this *AutoReload) AddLoad(cm CheckModFunc, dl DataLoadFunc, lf LoadFinishFunc, file string) (error, *sync.WaitGroup) {
	this.pmu.Lock()
	defer this.pmu.Unlock()
	ri := &ReloadInfo{cm, dl, lf, file, nil, &sync.WaitGroup{}}
	ri.wg.Add(1)
	this.peeding = append(this.peeding, ri)
	return nil, ri.wg
}

func AddLoadSync(cm CheckModFunc, dl DataLoadFunc, lf LoadFinishFunc, file string) (error, *sync.WaitGroup) {
	return defaultLoader.AddLoadSync(cm, dl, lf, file)

}

func AddLoad(cm CheckModFunc, dl DataLoadFunc, lf LoadFinishFunc, file string) (error, *sync.WaitGroup) {
	return defaultLoader.AddLoad(cm, dl, lf, file)
}
