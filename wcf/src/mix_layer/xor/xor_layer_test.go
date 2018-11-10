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
	for i := 0; i < 10; i++ {
		encData, err := enc.Encode(word)
		if err != nil {
			t.Fatal(err)
		}
		decData, err := dec.Decode(encData)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(decData, word) {
			t.Fatalf("dec:%s not equal old:%s", hex.EncodeToString(decData), hex.EncodeToString(word))
		}
		if bytes.Equal(decData, encData) {
			t.Fatalf("enc:%s equal to dec:%s", hex.EncodeToString(encData), hex.EncodeToString(decData))
		}
	}
}
