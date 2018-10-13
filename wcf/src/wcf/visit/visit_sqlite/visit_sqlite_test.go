package visit_sqlite

import (
	"testing"
	"time"
)

func TestFormat(t *testing.T) {
	timeStr := time.Now().Format("20060102")
	t.Logf(timeStr)
}

func TestVisitSqlite_toDay(t *testing.T) {
	t.Log(todayAfterNDay(1))
}

