package mix_layer

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
)

func PKCS5Padding(src []byte, blockSize int) []byte {
	padding := blockSize - len(src)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	totalData := make([]byte, len(src)+len(padtext))
	copy(totalData, src)
	copy(totalData[len(src):], padtext)
	return totalData
}

func PKCS5UnPadding(src []byte) ([]byte, error) {
	length := len(src)
	unpadding := int(src[length-1])
	if length <= unpadding {
		return nil, errors.New(fmt.Sprintf("invalid pad data, length:%d, unpadding:%d", length, unpadding))
	}
	return src[:(length - unpadding)], nil
}

//datalen(4) + iv(variable) + data(variable) + hmac-sha1(20)
func EncodeHeadFrame(src []byte, ivin []byte, key []byte) ([]byte, error) {
	total := len(src) + len(ivin) + 4 + HMAC_LENGTH
	out := make([]byte, total)
	binary.BigEndian.PutUint32(out, uint32(total))
	copy(out[4:], ivin)
	copy(out[4+len(ivin):], src)
	hasher := hmac.New(sha1.New, key)
	hasher.Write(out[:4+len(ivin)+len(src)])
	hmacSum := hasher.Sum(nil)
	copy(out[4+len(ivin)+len(src):], hmacSum)
	return out, nil
}

func CheckHeadFrame(src []byte, ivlen int, maxData int) (int, error) {
	if len(src) < 4 {
		return 0, nil
	}
	total := int(binary.BigEndian.Uint32(src))
	if total <= 0 {
		return -2, errors.New(fmt.Sprintf("invalid data frame len:%d", total))
	}
	if total > maxData {
		return -1, errors.New(fmt.Sprintf("data too long, skip, data len:%d, max len:%d", total, maxData))
	}
	if total > len(src) {
		return 0, nil
	}
	if len(src) <= 4+HMAC_LENGTH+ivlen {
		return 0, nil
	}
	return total, nil
}

//return decode data len, iv, error
func DecodeHeadFrame(src []byte, ivout []byte, key []byte) ([]byte, error) {
	sz, err := CheckHeadFrame(src, len(ivout), MAX_CRYPT_PACKET_LEN)
	if sz <= 0 {
		return nil, fmt.Errorf("package check fail, sz:%d, err:%v", sz, err)
	}
	hasher := hmac.New(sha1.New, key)
	hasher.Write(src[:sz-HMAC_LENGTH])
	hmacSum := hasher.Sum(nil)
	if !bytes.Equal(hmacSum, src[sz-HMAC_LENGTH:]) {
		return nil, fmt.Errorf("hmac check fail, acquire hmac:%s, but get hmac:%s",
			hex.EncodeToString(hmacSum), hex.EncodeToString(src[sz-HMAC_LENGTH:]))
	}
	ivlen := copy(ivout, src[4:])
	return src[4+ivlen : sz-HMAC_LENGTH], nil
}

func RandIV(sz int) []byte {
	if sz <= 0 {
		return nil
	}
	iv := make([]byte, sz)
	rand.Reader.Read(iv)
	return iv
}

func GenKeyWithPad(key []byte, padto int) []byte {
	if len(key) == padto {
		return key
	}
	finalKey := make([]byte, padto)
	copy(finalKey, key)
	return finalKey
}
