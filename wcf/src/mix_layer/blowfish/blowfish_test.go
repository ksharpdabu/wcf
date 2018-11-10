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
	for i := 0; i < 10; i++ {
		encData, err := enc.Encode([]byte(word))
		if err != nil {
			t.Fatal(err)
		}
		decData, err := dec.Decode(encData)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("enc hex:%s, dec hex:%s, old len:%d", hex.EncodeToString(encData), hex.EncodeToString(decData), len(word))
		if string(decData) != word {
			t.Fatalf("not equal, dec:%s, old:%s", string(decData), word)
		}
	}
}
