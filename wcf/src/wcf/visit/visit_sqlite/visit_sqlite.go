package visit_sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"time"
	"wcf/visit"
)

func init() {
	visit.Regist("sqlite3", func() (visit.Visitor, error) {
		return &VisitSqlite{}, nil
	})
}

type VisitSqlite struct {
	db  *sql.DB
	cfg *VisitConfig
}

type VisitConfig struct {
	InitSQL       string `json:"init_template"`
	InsertSQL     string `json:"insert_template"`
	StoreLocation string `json:"store_location"`
	PreCreateDay  int    `json:"pre_create_n_day"`
}

func todayAfterNDay(n int) string {
	return time.Now().AddDate(0, 0, n).Format("20060102")
}

func (this *VisitSqlite) Init(file string) error {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	cfg := VisitConfig{}
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return err
	}
	this.cfg = &cfg
	db, err := sql.Open("sqlite3", cfg.StoreLocation)
	if err != nil {
		return err
	}
	_, err = db.Exec(fmt.Sprintf(cfg.InitSQL, todayAfterNDay(0)))
	if err != nil {
		return err
	}
	this.db = db
	go this.autoNewTable()
	return nil
}

func (this *VisitSqlite) autoNewTable() {
	for {
		for i := 0; i < this.cfg.PreCreateDay; i++ {
			_, err := this.db.Exec(fmt.Sprintf(this.cfg.InitSQL, todayAfterNDay(i+1)))
			if err != nil {
				//TO DO SOMETHING?
			}
		}
		time.Sleep(12 * time.Hour)
	}
}

func (this *VisitSqlite) OnView(view *visit.VisitInfo) error {
	stmt, err := this.db.Prepare(fmt.Sprintf(this.cfg.InsertSQL, todayAfterNDay(0)))
	if err != nil {
		return nil
	}
	defer stmt.Close()
	_, err = stmt.Exec(view.Name, view.Host, view.From, view.Start, view.End, view.Read, view.Write, view.ConnectCost)
	if err != nil {
		return err
	}
	return nil
}
