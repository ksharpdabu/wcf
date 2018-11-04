package aes_ctr

import (
	"encoding/hex"
	"testing"
)

var key = []byte("this is a test key....")
var iv = []byte("this is a test iv...")

var word = "hello this is a test plain"

func newCTR(keyLen int) (*AesCTR, error) {
	cfb := NewAesCTR(keyLen)
	err := cfb.Init(key, iv)
	if err != nil {
		return nil, err
	}
	return cfb, nil
}

func testWithKeyLen(t *testing.T, keyLen int) {
	enc, err := newCTR(keyLen)
	if err != nil {
		t.Fatal(err)
	}
	dec, err := newCTR(keyLen)
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
		t.Logf("keylen:%d enc hex:%s, dec hex:%s", keyLen, hex.EncodeToString(encData[:encLen]), hex.EncodeToString(decData[:decLen]))
		if string(decData[:decLen]) != word {
			t.Fatalf("not equal, dec:%s, old:%s", string(decData[:decLen]), word)
		}
	}
}

func TestEnDec(t *testing.T) {
	keyGroup := []int{16, 24, 32}
	for _, v := range keyGroup {
		testWithKeyLen(t, v)
	}
}
