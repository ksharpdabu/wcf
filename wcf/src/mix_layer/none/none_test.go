package none

import (
	"bytes"
	"encoding/hex"
	"testing"
)

var key = []byte("this is a test key")
var iv = []byte("this is a test iv")
var word = []byte("this is test plain...")

func getNone() *None {
	n := NewNone()
	n.InitRead(key, iv)
	n.InitWrite(key, iv)
	return n
}

func TestEnDec(t *testing.T) {
	enc := getNone()
	dec := getNone()
	for i := 0; i < 10; i++ {
		encData, err := enc.Encode(word)
		if err != nil {
			t.Fatal(err)
		}
		encRaw := encData
		decRaw, err := dec.Decode(encRaw)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("enc:%s, old:%s", hex.EncodeToString(encRaw), hex.EncodeToString(word))
		if !bytes.Equal(decRaw, word) {
			t.Fatalf("dec:%s not equal old:%s", hex.EncodeToString(decRaw), hex.EncodeToString(word))
		}
	}
}
