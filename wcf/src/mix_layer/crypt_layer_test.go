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

func TestHMAC_SHA1_Sum(t *testing.T) {
	writer := hmac.New(sha1.New, key)
	encRaw := []byte("this is a test")
	encRawWithHMAC := writer.Sum(encRaw)
	//t.Logf("%s", encRawWithHMAC)
	if len(encRawWithHMAC) != len(encRaw)+20 {
		t.Fatal("not expect length!")
	}
	writer2 := hmac.New(sha1.New, key)
	writer.Write(encRaw)
	hmac2 := writer2.Sum(nil)
	if !bytes.Equal(hmac2, encRawWithHMAC[len(encRaw):]) {
		t.Fatal("hmac not equal")
	}
}

func TestEncodeAndDecode(t *testing.T) {
	ivin := iv[:20]
	enc, err := EncodeHeadFrame(data, ivin, key)
	if err != nil {
		t.Fatal(err)
	}
	encData := enc
	dataLen, err := CheckHeadFrame(encData, len(ivin), 64*1024)
	if err != nil || dataLen < 0 {
		t.Fatal(err)
	}
	ivout := make([]byte, HMAC_LENGTH)
	dec, err := DecodeHeadFrame(encData, ivout, key)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(dec, data) {
		t.Fatalf("data not equal, hex data out:%s, hex data in:%v", hex.EncodeToString(dec), hex.EncodeToString(data))
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
