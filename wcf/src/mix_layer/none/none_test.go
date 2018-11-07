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
	encData := make([]byte, 64*1024)
	decData := make([]byte, 64*1024)
	for i := 0; i < 10; i++ {
		encLen, err := enc.Encode(word, encData)
		if err != nil {
			t.Fatal(err)
		}
		encRaw := encData[:encLen]
		decLen, err := dec.Decode(encRaw, decData)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("enc:%s, old:%s", hex.EncodeToString(encRaw), hex.EncodeToString(word))
		decRaw := decData[:decLen]
		if !bytes.Equal(decRaw, word) {
			t.Fatalf("dec:%s not equal old:%s", hex.EncodeToString(decRaw), hex.EncodeToString(word))
		}
	}
}
