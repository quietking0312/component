package mcyptos

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// 密钥必须是 16、24、32字节长度
//var (
//	key = []byte("devops---aespass")
//)

func GetMD5(arg []byte) []byte {
	h := md5.New()
	h.Write(arg)
	return h.Sum(nil)
}

// Get32MD5 获取字符串32位md5
func Get32MD5(args []byte) string {
	return fmt.Sprintf("%x", GetMD5(args))
}

// Get16MD5 获取字符串16位md5
func Get16MD5(args []byte) string {
	return Get32MD5(args)[8:24]
}

func GetFileMd5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	h := md5.New()
	_, _ = io.Copy(h, file)
	return hex.EncodeToString(h.Sum(nil)), nil
}

func SHA1(args []byte) string {
	s := sha1.New()
	s.Write(args)
	return fmt.Sprintf("%x", s.Sum(nil))
}

func _padding(src []byte, blockSize int) []byte {
	padNum := blockSize - len(src)%blockSize
	pad := bytes.Repeat([]byte{byte(padNum)}, padNum)
	return append(src, pad...)
}

func _unPadding(src []byte) []byte {
	n := len(src)
	if n == 0 {
		return src
	}
	unPadNum := int(src[n-1])
	return src[:n-unPadNum]
}

func EncrypterAES(key []byte, src []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	src = _padding(src, len(key))
	iv := key[:aes.BlockSize]
	blockMode := cipher.NewCBCEncrypter(block, iv)
	blockMode.CryptBlocks(src, src)
	return src, nil
}

func DecryptAES(key []byte, src []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	iv := key[:aes.BlockSize]
	blockMode := cipher.NewCBCDecrypter(block, iv)
	blockMode.CryptBlocks(src, src)
	src = _unPadding(src)
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
