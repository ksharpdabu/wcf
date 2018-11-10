package aes_gcm

import (
	"encoding/hex"
	"testing"
)

var key = []byte("haha, are you ok?")
var iv = []byte("this isx a test ivxxxxsdsadsada")
var word = "this is a test string...."

func newGCM(t *testing.T, keylen int) *AesGCM {
	gcm := NewAesGCM(keylen)
	if err := gcm.InitRead(key, iv); err != nil {
		t.Fatal(err)
	}
	gcm.InitWrite(key, iv)
	return gcm
}

func testWithKeyLen(t *testing.T, keylen int) {
	enc := newGCM(t, keylen)
	dec := newGCM(t, keylen)
	for i := 0; i < 10; i++ {
		encData, err := enc.Encode([]byte(word))
		if err != nil {
			t.Fatal(err)
		}
		decData, err := dec.Decode(encData)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("keylen:%d, enc hex:%s, dec hex:%s", keylen, hex.EncodeToString(encData), hex.EncodeToString(decData))
		if string(decData) != word {
			t.Fatalf("not match, dec:%s, old:%s", string(decData), word)
		}
	}
}

func TestEnDec(t *testing.T) {
	keyGroup := []int{16, 24, 32}
	for _, v := range keyGroup {
		testWithKeyLen(t, v)
	}
}
