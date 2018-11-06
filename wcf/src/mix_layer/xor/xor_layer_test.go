package xor

import (
	"bytes"
	"encoding/hex"
	"testing"
)

var key = []byte("this is a test key")
var iv = []byte("this is a test iv")
var word = []byte("this is test plain...")

func getXor() *Xor {
	xor := NewXor()
	xor.InitRead(key, iv)
	xor.InitWrite(key, iv)
	return xor
}

func TestXor_Compare(t *testing.T) {
	enc := getXor()
	dec := getXor()

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
		if bytes.Equal(decRaw, encRaw) {
			t.Fatalf("enc:%s equal to dec:%s", hex.EncodeToString(encRaw), hex.EncodeToString(decRaw))
		}
	}
}
