package mix_layer

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
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
		return nil, fmt.Errorf("invalid pad data, length:%d, unpadding:%d", length, unpadding)
	}
	return src[:(length - unpadding)], nil
}

func GenHMAC_SHA1(b []byte, key []byte) []byte {
	hasher := hmac.New(sha1.New, key)
	hasher.Write(b)
	return hasher.Sum(nil)
}

func GenHMAC_SHA1_WITH_LENGTH(b []byte, key []byte, sz int) []byte {
	mac := GenHMAC_SHA1(b, key)
	if sz == len(mac) {
		return mac
	} else if sz < len(mac) {
		return mac[:sz]
	}
	rp := sz - len(mac)
	mac = append(mac, bytes.Repeat([]byte{byte(rp)}, rp)...)
	return mac
}

//datalen(4) hmac-sha1(4) + iv(variable) + data(variable) + hmac-sha1(4)
func EncodeHeadFrame(src []byte, ivin []byte, key []byte) ([]byte, error) {
	total := len(src) + HMAC_LENGTH + len(ivin) + 4 + HMAC_LENGTH
	out := make([]byte, total)
	binary.BigEndian.PutUint32(out, uint32(total))
	//填充对长度字段的校验
	copy(out[4:], GenHMAC_SHA1_WITH_LENGTH(out[:4], key, HMAC_LENGTH))
	copy(out[4+HMAC_LENGTH:], ivin)
	copy(out[4+HMAC_LENGTH+len(ivin):], src)
	hmacSum := GenHMAC_SHA1_WITH_LENGTH(out[:4+HMAC_LENGTH+len(ivin)+len(src)], key, HMAC_LENGTH)
	copy(out[4+HMAC_LENGTH+len(ivin)+len(src):], hmacSum)
	return out, nil
}

func CheckHeadFrameWithKey(src []byte, ivlen int, maxData int, key []byte) (int, error) {
	if len(src) < 4 {
		return 0, nil
	}
	total := int(binary.BigEndian.Uint32(src))
	if total <= 0 {
		return -2, fmt.Errorf("invalid data frame len:%d", total)
	}
	if total > maxData {
		return -1, fmt.Errorf("data too long, skip, data len:%d, max len:%d", total, maxData)
	}
	//不带key就不校验这个字段了。。
	if len(key) != 0 {
		if len(src) < 8 {
			return 0, nil
		}
		mac := GenHMAC_SHA1_WITH_LENGTH(src[:4], key, HMAC_LENGTH)
		if !bytes.Equal(mac, src[4:8]) {
			return -3, fmt.Errorf("data len hmac check fail, carry:%s, calc:%s",
				hex.EncodeToString(src[4:8]), hex.EncodeToString(mac))
		}
	}
	if total > len(src) {
		return 0, nil
	}
	if len(src) <= 4+HMAC_LENGTH+ivlen+HMAC_LENGTH {
		return 0, nil
	}
	return total, nil
}

func CheckHeadFrame(src []byte, ivlen int, maxData int) (int, error) {
	return CheckHeadFrameWithKey(src, ivlen, maxData, nil)
}

//return decode data len, iv, error
func DecodeHeadFrame(src []byte, ivout []byte, key []byte) ([]byte, error) {
	sz, err := CheckHeadFrame(src, len(ivout), MAX_CRYPT_PACKET_LEN)
	if sz <= 0 {
		return nil, fmt.Errorf("package check fail, sz:%d, err:%v", sz, err)
	}
	hmacSum := GenHMAC_SHA1_WITH_LENGTH(src[:sz-HMAC_LENGTH], key, HMAC_LENGTH)
	if !bytes.Equal(hmacSum, src[sz-HMAC_LENGTH:]) {
		return nil, fmt.Errorf("hmac check fail, acquire hmac:%s, but get hmac:%s",
			hex.EncodeToString(hmacSum), hex.EncodeToString(src[sz-HMAC_LENGTH:]))
	}
	ivlen := copy(ivout, src[4+HMAC_LENGTH:])
	return src[4+HMAC_LENGTH+ivlen : sz-HMAC_LENGTH], nil
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
