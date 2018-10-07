package check

import (
	"testing"
)

func TestReload(t *testing.T) {
	rule, err := NewRule("D:/GoProj/wcf/wcf/src/wcf/cmd/local/host.rule")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("domain:%d, cidr:%d", len(rule.domain), len(rule.cidr))
	lst := []string {
		"api.baidu.com",
		"test.solidot.org",
		"127.0.0.1",
		"192.168.1.1",
		"hello.com",
		"localhost",
	}
	for _, v := range lst {
		info := rule.GetHostRule(v)
		t.Logf("addr:%s, %+v", v, info)
	}
}




