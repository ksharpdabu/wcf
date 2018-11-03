package aes_gcm

import (
	"encoding/hex"
	"testing"
)

var key = []byte("haha, are you ok?")
var iv = []byte("this is a test iv")
var word = "this is a test string...."

func newGCM(t *testing.T) *AesGCM {
	gcm := &AesGCM{}
	if err := gcm.Init(key, iv); err != nil {
		t.Fatal(err)
	}
	return gcm
}

func TestEnDec(t *testing.T) {
	enc := newGCM(t)
	dec := newGCM(t)
	for i := 0; i < 10; i++ {
		data, err := enc.Encode([]byte(word))
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("enc hex:%s", hex.EncodeToString(data))
		raw, err := dec.Decode(data)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("dec hex:%s, raw:%s", hex.EncodeToString(raw), string(raw))
		if string(raw) != word {
			t.Fatalf("not match, dec:%s, old:%s", string(raw), word)
		}
	}
}
