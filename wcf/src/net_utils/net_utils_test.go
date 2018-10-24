package net_utils

import "testing"

func TestResolveRealAddr(t *testing.T) {
	t.Logf(ResolveRealAddr("::1   "))
	t.Logf(ResolveRealAddr("[::1]"))
	t.Logf(ResolveRealAddr("127.0.0.1"))
	t.Logf(ResolveRealAddr("www.test.com"))
	t.Logf(ResolveRealAddr("fe80::d9ee:b5b7:ac1e:a9ae"))
}

