package check

import (
	"testing"
	"net_utils"
)

var file = "D:/GoProj/wcf/wcf/src/config/local_host.rule"

func TestCheckIPIn(t *testing.T) {
	lst := RouteList{
		&CIDRRange{Start:1, End:2},
		&CIDRRange{Start:4, End:5},
		&CIDRRange{Start:7, End:9},
		&CIDRRange{Start:17, End:19},
		&CIDRRange{Start:20, End:25},
	}
	checkInList := []uint32 {1, 5, 8, 18, 19, 20, 21, 25}
	checkNotInList := []uint32 {0, 3, 6, 10, 11, 12, 13, 14, 15, 26, 27}
	for _, k := range checkInList {
		if !checkIPInRangeList(k, lst) {
			t.Errorf("invalid, k:%d shoud in!", k)
		}
	}
	for _, k := range checkNotInList {
		if checkIPInRangeList(k, lst) {
			t.Errorf("invalid, k:%d shoud not in!", k)
		}
	}
}

func TestNewRule(t *testing.T) {
	rule, err := NewRule(file)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range rule.routeV4 {
		t.Logf("OPType:%d", k)
		for _, iv := range v {
			t.Logf("%d:%d, cidrs:%v, %s->%s], ", iv.Start, iv.End, iv.CIDRs, net_utils.InetNtoA(iv.Start), net_utils.InetNtoA(iv.End))
		}
	}
}

func TestReload(t *testing.T) {
	rule, err := NewRule(file)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("domain:%d, cidr:%d", len(rule.domain), len(rule.cidr))
	lst := []string {
		"[bb:b::1]",
		"bb:b::1",
		"api.baidu.com",
		"test.solidot.org",
		"127.0.0.1",
		"192.168.1.1",
		"hello.com",
		"localhost",
		"[::1]",
		"::1",
		"[bb::1]",
		"bb::1",
	}
	for _, v := range lst {
		info := rule.GetHostRule(v)
		t.Logf("addr:%s, %+v", v, info)
	}
}




