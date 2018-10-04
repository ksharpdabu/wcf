package wcf

import (
	"testing"
	"time"
)

func TestReload(t *testing.T) {
	_, err := NewUserHolder("D:/GoPath/src/wcf/cmd/server/userinfo.dat")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1 * time.Minute)
}
