package aes_layer

import (
	"testing"
)

var data = []byte("haha this is a test")

func TestEncAndDec(t *testing.T) {
	buf := make([]byte, 1024)
	cnt, err := EncodeHeadFrame(data, buf)
	if err != nil {
		t.Fatal(err)
	}
	if total, err := CheckHeadFrame(buf[:cnt], 1024); err != nil || total <= 0 {
		t.Fatalf("check fail, err:%v, total:%d", err, total)
	}
	decBuf := make([]byte, 1024)
	cnt, err = DecodeHeadFrame(buf, decBuf)
	if err != nil {
		t.Fatal(err)
	}
	if string(decBuf[:cnt]) != string(data) {
		t.Fatal("data not match, dec:%s, old:%s", string(decBuf[:cnt]), string(data))
	}
	t.Logf("decode:%s, old:%s, dec cnt:%d, old cnt:%d", string(decBuf[:cnt]), string(data), cnt, len(data))
}
