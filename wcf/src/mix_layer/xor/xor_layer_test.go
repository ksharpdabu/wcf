package xor

import (
	"encoding/hex"
	"testing"
)

func TestXor_SetKey(t *testing.T) {
	key := "hello world     xcxld     xcxld     xcxld     xcxld     xcxld     xcxld     xcxld     xcxxx"
	xor := Xor{}
	xor.SetKey(key)
	t.Logf("hex(key):%s, keylen:%d", hex.EncodeToString([]byte(xor.key)), len(xor.key))
}

func TestXor_Compare(t *testing.T) {
	xor := Xor{}
	xor.SetKey("helloworld21")
	tmp := "this is a test"
	word := make([]byte, len(tmp))
	copy(word, tmp)
	xor.xor(word, 0)
	t.Log(string(word))
	t.Log(hex.EncodeToString([]byte(word)))
	xor.xor(word, 0)
	t.Log(string(word))
}
