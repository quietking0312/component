package mcyptos

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"hash"
	"io"
	"os"
)

// MD5 计算数据的 MD5 值，返回十六进制字符串
func MD5(data []byte) string {
	h := md5.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// MD5String 计算字符串的 MD5 值
func MD5String(s string) string {
	return MD5([]byte(s))
}

// MD5File 计算文件的 MD5 值
func MD5File(filepath string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	return MD5Reader(file)
}

// MD5Reader 计算 io.Reader 的 MD5 值
func MD5Reader(r io.Reader) (string, error) {
	h := md5.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// MD5ReaderWithSize 计算 io.Reader 的 MD5 值，并返回数据大小
func MD5ReaderWithSize(r io.Reader) (hash string, size int64, err error) {
	h := md5.New()
	size, err = io.Copy(h, r)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), size, nil
}

// MD5Partial 计算数据的 MD5 值，返回原始字节
func MD5Partial(data []byte) []byte {
	h := md5.New()
	h.Write(data)
	return h.Sum(nil)
}

// ========== SHA 系列 ==========

// SHA1 计算数据的 SHA1 值
func SHA1(data []byte) string {
	h := sha1.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// SHA1String 计算字符串的 SHA1 值
func SHA1String(s string) string {
	return SHA1([]byte(s))
}

// SHA1File 计算文件的 SHA1 值
func SHA1File(filepath string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	return SHA1Reader(file)
}

// SHA1Reader 计算 io.Reader 的 SHA1 值
func SHA1Reader(r io.Reader) (string, error) {
	h := sha1.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// SHA256 计算数据的 SHA256 值
func SHA256(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// SHA256String 计算字符串的 SHA256 值
func SHA256String(s string) string {
	return SHA256([]byte(s))
}

// SHA256File 计算文件的 SHA256 值
func SHA256File(filepath string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	return SHA256Reader(file)
}

// SHA256Reader 计算 io.Reader 的 SHA256 值
func SHA256Reader(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// SHA512 计算数据的 SHA512 值
func SHA512(data []byte) string {
	h := sha512.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// SHA512String 计算字符串的 SHA512 值
func SHA512String(s string) string {
	return SHA512([]byte(s))
}

// SHA512File 计算文件的 SHA512 值
func SHA512File(filepath string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	return SHA512Reader(file)
}

// SHA512Reader 计算 io.Reader 的 SHA512 值
func SHA512Reader(r io.Reader) (string, error) {
	h := sha512.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ========== HMAC 系列 ==========

// HMACMD5 计算 HMAC-MD5
func HMACMD5(key, data []byte) string {
	h := hmac.New(md5.New, key)
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// HMACMD5String 计算字符串的 HMAC-MD5
func HMACMD5String(key, s string) string {
	return HMACMD5([]byte(key), []byte(s))
}

// HMACSHA256 计算 HMAC-SHA256
func HMACSHA256(key, data []byte) string {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// HMACSHA256String 计算字符串的 HMAC-SHA256
func HMACSHA256String(key, s string) string {
	return HMACSHA256([]byte(key), []byte(s))
}

// HMACSHA512 计算 HMAC-SHA512
func HMACSHA512(key, data []byte) string {
	h := hmac.New(sha512.New, key)
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// HMACSHA512String 计算字符串的 HMAC-SHA512
func HMACSHA512String(key, s string) string {
	return HMACSHA512([]byte(key), []byte(s))
}

// ========== Hash 接口 ==========

// Hasher 哈希计算器
type Hasher struct {
	h hash.Hash
}

// NewMD5 创建 MD5 计算器
func NewMD5() *Hasher {
	return &Hasher{h: md5.New()}
}

// NewSHA1 创建 SHA1 计算器
func NewSHA1() *Hasher {
	return &Hasher{h: sha1.New()}
}

// NewSHA256 创建 SHA256 计算器
func NewSHA256() *Hasher {
	return &Hasher{h: sha256.New()}
}

// NewSHA512 创建 SHA512 计算器
func NewSHA512() *Hasher {
	return &Hasher{h: sha512.New()}
}

// Write 写入数据
func (h *Hasher) Write(data []byte) (int, error) {
	return h.h.Write(data)
}

// WriteString 写入字符串
func (h *Hasher) WriteString(s string) (int, error) {
	return h.h.Write([]byte(s))
}

// Sum 返回哈希结果（十六进制字符串）
func (h *Hasher) Sum() string {
	return hex.EncodeToString(h.h.Sum(nil))
}

// SumBytes 返回哈希结果（原始字节）
func (h *Hasher) SumBytes() []byte {
	return h.h.Sum(nil)
}

// Reset 重置计算器
func (h *Hasher) Reset() {
	h.h.Reset()
}

// ========== 批量计算 ==========

// MultiHash 同时计算多种哈希
type MultiHash struct {
	MD5    string
	SHA1   string
	SHA256 string
	SHA512 string
}

// CalculateAll 同时计算 MD5、SHA1、SHA256、SHA512
func CalculateAll(data []byte) *MultiHash {
	return &MultiHash{
		MD5:    MD5(data),
		SHA1:   SHA1(data),
		SHA256: SHA256(data),
		SHA512: SHA512(data),
	}
}

// CalculateAllReader 同时计算 io.Reader 的多种哈希
func CalculateAllReader(r io.Reader) (*MultiHash, int64, error) {
	md5Hash := md5.New()
	sha1Hash := sha1.New()
	sha256Hash := sha256.New()
	sha512Hash := sha512.New()

	// 使用 MultiWriter 同时写入多个 hash
	writer := io.MultiWriter(md5Hash, sha1Hash, sha256Hash, sha512Hash)
	size, err := io.Copy(writer, r)
	if err != nil {
		return nil, 0, err
	}

	return &MultiHash{
		MD5:    hex.EncodeToString(md5Hash.Sum(nil)),
		SHA1:   hex.EncodeToString(sha1Hash.Sum(nil)),
		SHA256: hex.EncodeToString(sha256Hash.Sum(nil)),
		SHA512: hex.EncodeToString(sha512Hash.Sum(nil)),
	}, size, nil
}

// CalculateAllFile 同时计算文件的多种哈希
func CalculateAllFile(filepath string) (*MultiHash, int64, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, 0, err
	}
	defer file.Close()

	return CalculateAllReader(file)
}
