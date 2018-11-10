package aes_cbc

import (
	"encoding/hex"
	"testing"
)

var key = []byte("this is a test key....")
var iv = []byte("xthis is a test iv...")

var word = "hello this is a test plain"

func newCBC(keyLen int) (*AesCBC, error) {
	cbc := NewAesCBC(keyLen)
	err := cbc.InitRead(key, iv)
	if err != nil {
		return nil, err
	}
	cbc.InitWrite(key, iv)
	return cbc, nil
}

func BenchmarkEncodeAndDecodeWith(b *testing.B) {
	var keylen = 24
	datalen := 32 * 1024
	enc, err := newCBC(keylen)
	if err != nil {
		b.Fatal(err)
	}
	data := make([]byte, datalen)
	dec, _ := newCBC(keylen)
	for i := 0; i < b.N; i++ {
		encData, err := enc.Encode(data)
		if err != nil {
			b.Fatal(err)
		}
		_, err = dec.Decode(encData)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncode(b *testing.B) {
	enc, err := newCBC(32)
	if err != nil {
		b.Fatal(err)
	}
	//decData := make([]byte, 64*1024)
	for i := 0; i < b.N; i++ {
		_, err := enc.Encode([]byte(word))
		if err != nil {
			b.Fatal(err)
		}
	}
}

func testWithKeyLen(t *testing.T, keyLen int) {
	enc, err := newCBC(keyLen)
	if err != nil {
		t.Fatal(err)
	}
	dec, err := newCBC(keyLen)
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
		t.Logf("keylen:%d enc hex:%s, dec hex:%s", keyLen, hex.EncodeToString(encData), hex.EncodeToString(decData))
		if string(decData) != word {
			t.Fatalf("not equal, dec:%s, old:%s", string(decData), word)
		}
	}
}

func TestEnDec(t *testing.T) {
	keyGroup := []int{16, 24, 32}
	for _, v := range keyGroup {
		testWithKeyLen(t, v)
	}
}
