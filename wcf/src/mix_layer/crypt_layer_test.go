package mix_layer

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"testing"
)

var key = []byte("this is a test key.....")
var iv = []byte("this is a test iv...dsdsdsfsadsadsad.")
var data = []byte("this is test data....")

func TestHMAC_SHA1(t *testing.T) {
	hasher := hmac.New(sha1.New, []byte("tesxxt"))
	hasher.Write([]byte("this is accc tesxxxxdddddddddddddddddddddddddddddddddddddddddddddddt"))
	data := hasher.Sum(nil)
	t.Logf("len:%d, data:%v", len(data), data)
	if len(data) != 20 {
		t.Fatal("hmac len not equal 20")
	}
}

func TestEncodeAndDecode(t *testing.T) {
	enc := make([]byte, 64*1024)
	dec := make([]byte, 64*1024)
	ivin := iv[:20]
	encLen, err := EncodeHeadFrame(data, enc, ivin, key)
	if err != nil {
		t.Fatal(err)
	}
	encData := enc[:encLen]
	dataLen, err := CheckHeadFrame(encData, len(ivin), 64*1024)
	if err != nil || dataLen < 0 {
		t.Fatal(err)
	}
	ivout := make([]byte, HMAC_LENGTH)
	decLen, err := DecodeHeadFrame(encData, dec, ivout, key)
	if err != nil {
		t.Fatal(err)
	}
	decData := dec[:decLen]
	if !bytes.Equal(decData, data) {
		t.Fatalf("data not equal, hex data out:%s, hex data in:%v", hex.EncodeToString(decData), hex.EncodeToString(data))
	}
	if !bytes.Equal(ivout, ivin) {
		t.Fatalf("iv not equal, hex ivout:%s, hex ivin:%s", hex.EncodeToString(ivout), hex.EncodeToString(ivin))
	}
}

func TestCopyNil(t *testing.T) {
	var v []byte
	cnt := copy(v, []byte("123"))
	if cnt != 0 {
		t.Fatal("not 0")
	}
}
