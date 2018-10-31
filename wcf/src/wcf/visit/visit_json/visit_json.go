package visit_json

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"wcf/visit"
)

func init() {
	visit.Regist("json", func() (visit.Visitor, error) {
		return &VisitJson{}, nil
	})
}

type VisitJson struct {
	Ptr *os.File
	Cnt int
}

type JsonConfig struct {
	StoreLocation string `json:"store"`
}

func (this *VisitJson) Init(file string) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	cfg := &JsonConfig{}
	json.Unmarshal(data, cfg)
	f, err := os.OpenFile(cfg.StoreLocation, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	this.Ptr = f
	return nil
}

func (this *VisitJson) OnView(view *visit.VisitInfo) error {
	line, err := json.Marshal(view)
	if err != nil {
		return err
	}
	fmt.Fprintf(this.Ptr, "%s\n", string(line))
	this.Cnt++
	if this.Cnt%5 == 0 {
		this.Ptr.Sync()
		this.Cnt = 0
	}
	return nil
}
