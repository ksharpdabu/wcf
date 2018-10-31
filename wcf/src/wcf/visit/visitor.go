package visit

import (
	"errors"
	"fmt"
	"time"
)

type VisitInfo struct {
	Name        string    `json:"user"`
	From        string    `json:"from"`
	Host        string    `json:"host"`
	Read        int64     `json:"read_cnt"`
	Write       int64     `json:"write_cnt"`
	ConnectCost int64     `json:"connect_cost"`
	Start       time.Time `json:"start"`
	End         time.Time `json:"end"`
}

type CreateFunc func() (Visitor, error)

type Visitor interface {
	Init(file string) error
	OnView(view *VisitInfo) error
}

var mp = make(map[string]CreateFunc)

func Regist(name string, ctf CreateFunc) {
	mp[name] = ctf
}

func Get(name string) (Visitor, error) {
	if v, ok := mp[name]; ok {
		return v()
	}
	return nil, errors.New(fmt.Sprintf("visitor:%s not found"))
}
