package aes_ofb

import (
	"encoding/hex"
	"testing"
)

var key = []byte("this is a test key....")
var iv = []byte("this is a test iv...")

var word = "hello this is a test plain"

func newOFB() (*AesOFB, error) {
	cfb := NewAesOFB(32)
	err := cfb.Init(key, iv)
	if err != nil {
		return nil, err
	}
	return cfb, nil
}

func TestEnDec(t *testing.T) {
	enc, err := newOFB()
	if err != nil {
		t.Fatal(err)
	}
	dec, err := newOFB()
	if err != nil {
		t.Fatal(err)
	}
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
			t.Fatalf("not equal, dec:%s, old:%s", string(decData[:decLen]), word)
		}
	}
}
