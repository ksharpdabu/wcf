package salsa20

import (
	"encoding/hex"
	"testing"
)

var key = []byte("this is a test key....")
var word = "hello this is a test plain"
var iv = []byte("this is a test iv")

func newSalsa20() (*Salsa20, error) {
	cha := NewSalsa20()
	err := cha.InitRead(key, iv)
	if err != nil {
		return nil, err
	}
	cha.InitWrite(key, iv)
	return cha, nil
}

func TestEnDec(t *testing.T) {
	enc, err := newSalsa20()
	if err != nil {
		t.Fatal(err)
	}
	dec, err := newSalsa20()
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
