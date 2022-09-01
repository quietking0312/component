package mcyptos

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/base64"
	"fmt"
)

// 密钥必须是 16、24、32字节长度
var (
	key = []byte("devops---aespass")
)

// Get32MD5 获取字符串32位md5
func Get32MD5(args string) string {
	h := md5.New()
	h.Size()
	h.Write([]byte(args))
	md5Str := fmt.Sprintf("%x", h.Sum(nil))
	return md5Str
}

// Get16MD5 获取字符串16位md5
func Get16MD5(args string) string {
	return Get32MD5(args)[8:24]
}

func padding(src []byte, blockSize int) []byte {
	padNum := blockSize - len(src)%blockSize
	pad := bytes.Repeat([]byte{byte(padNum)}, padNum)
	return append(src, pad...)
}

func unPadding(src []byte) []byte {
	n := len(src)
	if n == 0 {
		return src
	}
	unPadNum := int(src[n-1])
	return src[:n-unPadNum]
}

func EncrypterAES(src []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	src = padding(src, block.BlockSize())
	blockMode := cipher.NewCBCEncrypter(block, key)
	blockMode.CryptBlocks(src, src)
	return src, nil
}

func DecryptAES(src []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockMode := cipher.NewCBCDecrypter(block, key)
	blockMode.CryptBlocks(src, src)
	src = unPadding(src)
	return src, nil
}

func EncodeBase64(src []byte) string {
	return base64.StdEncoding.EncodeToString(src)
}

func DecodeBase64(src string) ([]byte, error) {
	dest, err := base64.StdEncoding.DecodeString(src)
	if err != nil {
		return nil, err
	}
	return dest, nil
}
