package transport

import "testing"

func TestGetAll(t *testing.T) {
	t.Log(GetAllBindName())
	t.Log(GetAllDialName())
}

func TestInitAllProtocol(t *testing.T) {
	InitAllProtocol("d:/GoProj/wcf/wcf/src/config/transport.json")
	for k, v := range parammp {
		t.Logf("protocols:%s, bindp:%+v, dialp:%+v", k, v.BindParam, v.DialParam)
	}
}
