package check

import (
	"testing"
	"regexp"
	"fmt"
	"time"
)

func TestReload(t *testing.T) {
	_, err := NewRule("D:/GoPath/src/wcf/cmd/local/black.rule")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1 * time.Minute)
}

func TestReg(t *testing.T) {
	reg := regexp.MustCompile("[0-9A-Za-z]+\\.[a-zA-Z]+$")
	fmt.Print(reg.FindAllString("test.aa.com", -1))
}

func Test_GetRootDomain(t *testing.T) {
	lst := []string {
		"test.aaa.com",
		"hello",
		"xxx.com",
		"xxx.xxx.aaa.ddd.ccc.com",
		"t123.com",
		"x.123",
		"23.323.com",
	}
	for i := 0; i < len(lst); i++ {
		t.Logf(GetRootDomain(lst[i]))
	}
}

func TestHostCheck(t *testing.T) {
	host, err := NewRule("D:/GoPath/src/wcf/cmd/local/black.rule")
	if err != nil {
		t.Fatal(err)
	}
	lst := []string {
		"baidu.com",
		"www.baidu.com",
	}
	for _, item := range lst {
		t.Logf("url:%s, v:%t", item, host.IsExists(item))
	}
	for _, item := range lst {
		t.Logf("url:%s, v:%t", item, host.IsSubExists(item))
	}
}

func TestBuildPoint(t *testing.T) {
	lst := []string {
		"a.com",
		"com",
		"xx.yy.zz.com",
		"a.b.c.d.e.f.g.h.i.j.k.l.com",
		"1.2.3.4.5.6.7.8.9.10.com",
	}
	for _, item := range lst {
		pt := GetAllSubDomainPoint(item)
		domains := BuildSubDomainByPoint(item, pt)
		t.Logf("url:%s, pt:%v domains:%v", item, pt, domains)
	}
}




