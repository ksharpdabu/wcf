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
func EncodeHeadFrame(src []byte, dst []byte, ivin []byte, key []byte) (int, error) {
	if len(dst) < len(src)+len(ivin)+4+HMAC_LENGTH {
		return 0, errors.New(fmt.Sprintf("buffer too small, src len:%d, buf len:%d", len(src), len(dst)))
	}
	binary.BigEndian.PutUint32(dst, uint32(len(src)+len(ivin)+4+HMAC_LENGTH))
	copy(dst[4:], ivin)
	copy(dst[4+len(ivin):], src)
	hasher := hmac.New(sha1.New, key)
	hasher.Write(dst[4 : 4+len(ivin)+len(src)])
	hmacSum := hasher.Sum(nil)
	copy(dst[4+len(ivin)+len(src):], hmacSum)
	return 4 + len(src) + len(ivin) + len(hmacSum), nil
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
func DecodeHeadFrame(src []byte, dst []byte, ivout []byte, key []byte) (int, error) {
	sz, err := CheckHeadFrame(src, len(ivout), MAX_CRYPT_PACKET_LEN)
	if sz <= 0 {
		return 0, errors.New(fmt.Sprintf("package check fail, sz:%d, err:%v", sz, err))
	}
	hasher := hmac.New(sha1.New, key)
	hasher.Write(src[4 : len(src)-HMAC_LENGTH])
	hmacSum := hasher.Sum(nil)
	if !bytes.Equal(hmacSum, src[len(src)-HMAC_LENGTH:]) {
		return 0, errors.New(fmt.Sprintf("hmac check fail, acquire hmac:%s, but get hmac:%s",
			hex.EncodeToString(hmacSum), hex.EncodeToString(src[len(src)-HMAC_LENGTH:])))
	}
	ivlen := copy(ivout, src[4:])
	return copy(dst, src[4+ivlen:sz-HMAC_LENGTH]), nil
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
