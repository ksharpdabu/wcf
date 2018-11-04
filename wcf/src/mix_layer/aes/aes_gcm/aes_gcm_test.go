package aes_gcm

import (
	"encoding/hex"
	"testing"
)

var key = []byte("haha, are you ok?")
var iv = []byte("this is a test iv")
var word = "this is a test string...."

func newGCM(t *testing.T) *AesGCM {
	gcm := NewAesGCM(32)
	if err := gcm.Init(key, iv); err != nil {
		t.Fatal(err)
	}
	return gcm
}

func TestEnDec(t *testing.T) {
	enc := newGCM(t)
	dec := newGCM(t)
	encData := make([]byte, 64*1024)
	decData := make([]byte, 64*1024)
	for i := 0; i < 10; i++ {
		encLen, err := enc.Encode([]byte(word), encData)
		if err != nil {
			t.Fatal(err)
		}
		decLen, err := dec.Decode(encData[:encLen], decData)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("enc hex:%s, dec hex:%s", hex.EncodeToString(encData[:encLen]), hex.EncodeToString(decData[:decLen]))
		if string(decData[:decLen]) != word {
			t.Fatalf("not match, dec:%s, old:%s", string(decData[:decLen]), word)
		}
	}
}
