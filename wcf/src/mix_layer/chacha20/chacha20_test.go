package chacha20

import (
	"encoding/hex"
	"testing"
)

var key = []byte("this is a test key....")
var word = "hello this is a test plain"
var iv = []byte("this is a test iv")

func newChacha20() (*Chacha20, error) {
	cha := NewChacha20()
	err := cha.InitRead(key, iv)
	if err != nil {
		return nil, err
	}
	cha.InitWrite(key, iv)
	return cha, nil
}

func TestEnDec(t *testing.T) {
	enc, err := newChacha20()
	if err != nil {
		t.Fatal(err)
	}
	dec, err := newChacha20()
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
