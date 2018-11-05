package blowfish

import (
	"encoding/hex"
	"testing"
)

var key = []byte("this is a test key....")
var word = "hello this is a test plain"
var iv = []byte("this is a test iv")

func newBlowFish() (*BlowFish, error) {
	fish := NewBlowFish()
	err := fish.InitRead(key, iv[:fish.IVLen()])
	if err != nil {
		return nil, err
	}
	fish.InitWrite(key, iv[:fish.IVLen()])
	return fish, nil
}

func TestEnDec(t *testing.T) {
	enc, err := newBlowFish()
	if err != nil {
		t.Fatal(err)
	}
	dec, err := newBlowFish()
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
		t.Logf("enc hex:%s, dec hex:%s, enc len:%d, dec len:%d, old len:%d", hex.EncodeToString(encData[:encLen]), hex.EncodeToString(decData[:decLen]), encLen, decLen, len(word))
		if string(decData[:decLen]) != word {
			t.Fatalf("not equal, dec:%s, old:%s", string(decData[:decLen]), word)
		}
	}
}
