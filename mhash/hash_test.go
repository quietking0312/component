package mhash

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMD5(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello, World!", "65a8e27d8879283831b664bd8b7f0ad4"},
		{"", "d41d8cd98f00b204e9800998ecf8427e"},
		{"123456", "e10adc3949ba59abbe56e057f20f883e"},
	}

	for _, tt := range tests {
		result := MD5String(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}

func TestMD5File(t *testing.T) {
	// 创建临时文件
	tmpFile, err := os.CreateTemp("", "mhash_test_*.txt")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// 写入测试数据
	testData := []byte("Hello, World!")
	_, err = tmpFile.Write(testData)
	assert.NoError(t, err)
	tmpFile.Close()

	// 计算文件 MD5
	result, err := MD5File(tmpFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, "65a8e27d8879283831b664bd8b7f0ad4", result)
}

func TestMD5Reader(t *testing.T) {
	testData := []byte("Hello, World!")
	reader := bytes.NewReader(testData)

	result, err := MD5Reader(reader)
	assert.NoError(t, err)
	assert.Equal(t, "65a8e27d8879283831b664bd8b7f0ad4", result)
}

func TestMD5ReaderWithSize(t *testing.T) {
	testData := []byte("Hello, World!")
	reader := bytes.NewReader(testData)

	hash, size, err := MD5ReaderWithSize(reader)
	assert.NoError(t, err)
	assert.Equal(t, "65a8e27d8879283831b664bd8b7f0ad4", hash)
	assert.Equal(t, int64(len(testData)), size)
}

func TestSHA1(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello, World!", "0a0a9f2a6772942557ab5355d76af442f8f65e01"},
		{"", "da39a3ee5e6b4b0d3255bfef95601890afd80709"},
	}

	for _, tt := range tests {
		result := SHA1String(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}

func TestSHA256(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello, World!", "dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f"},
		{"", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
	}

	for _, tt := range tests {
		result := SHA256String(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}

func TestSHA512(t *testing.T) {
	result := SHA512String("Hello, World!")
	assert.NotEmpty(t, result)
	assert.Equal(t, 128, len(result)) // SHA512 是 64 字节，十六进制是 128 字符
}

func TestHMACMD5(t *testing.T) {
	key := []byte("secret")
	data := []byte("Hello, World!")

	result := HMACMD5(key, data)
	assert.NotEmpty(t, result)
	assert.Equal(t, 32, len(result)) // MD5 是 16 字节，十六进制是 32 字符
}

func TestHMACSHA256(t *testing.T) {
	key := "secret"
	data := "Hello, World!"

	result := HMACSHA256String(key, data)
	assert.NotEmpty(t, result)
	assert.Equal(t, 64, len(result)) // SHA256 是 32 字节，十六进制是 64 字符
}

func TestHMACSHA512(t *testing.T) {
	key := "secret"
	data := "Hello, World!"

	result := HMACSHA512String(key, data)
	assert.NotEmpty(t, result)
	assert.Equal(t, 128, len(result)) // SHA512 是 64 字节，十六进制是 128 字符
}

func TestHasher(t *testing.T) {
	// 测试 MD5 Hasher
	hasher := NewMD5()
	hasher.WriteString("Hello, ")
	hasher.WriteString("World!")
	result := hasher.Sum()
	assert.Equal(t, "65a8e27d8879283831b664bd8b7f0ad4", result)

	// 测试重置
	hasher.Reset()
	hasher.WriteString("test")
	result2 := hasher.Sum()
	assert.NotEqual(t, result, result2)
}

func TestHasherSHA256(t *testing.T) {
	hasher := NewSHA256()
	hasher.WriteString("Hello, World!")
	result := hasher.Sum()
	assert.Equal(t, "dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f", result)
}

func TestCalculateAll(t *testing.T) {
	data := []byte("Hello, World!")
	result := CalculateAll(data)

	assert.NotEmpty(t, result.MD5)
	assert.NotEmpty(t, result.SHA1)
	assert.NotEmpty(t, result.SHA256)
	assert.NotEmpty(t, result.SHA512)

	// 验证 MD5
	assert.Equal(t, MD5(data), result.MD5)
	// 验证 SHA256
	assert.Equal(t, SHA256(data), result.SHA256)
}

func TestCalculateAllReader(t *testing.T) {
	data := []byte("Hello, World!")
	reader := bytes.NewReader(data)

	result, size, err := CalculateAllReader(reader)
	assert.NoError(t, err)
	assert.Equal(t, int64(len(data)), size)

	assert.NotEmpty(t, result.MD5)
	assert.NotEmpty(t, result.SHA1)
	assert.NotEmpty(t, result.SHA256)
	assert.NotEmpty(t, result.SHA512)

	// 验证 MD5
	assert.Equal(t, MD5(data), result.MD5)
	// 验证 SHA256
	assert.Equal(t, SHA256(data), result.SHA256)
}

func TestCalculateAllFile(t *testing.T) {
	// 创建临时文件
	tmpFile, err := os.CreateTemp("", "mhash_test_*.txt")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// 写入测试数据
	testData := []byte("Hello, World!")
	_, err = tmpFile.Write(testData)
	assert.NoError(t, err)
	tmpFile.Close()

	// 计算所有哈希
	result, size, err := CalculateAllFile(tmpFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, int64(len(testData)), size)

	assert.NotEmpty(t, result.MD5)
	assert.NotEmpty(t, result.SHA1)
	assert.NotEmpty(t, result.SHA256)
	assert.NotEmpty(t, result.SHA512)

	// 验证 MD5
	assert.Equal(t, MD5(testData), result.MD5)
}

func BenchmarkMD5(b *testing.B) {
	data := []byte("Hello, World! This is a benchmark test.")
	for i := 0; i < b.N; i++ {
		MD5(data)
	}
}

func BenchmarkSHA256(b *testing.B) {
	data := []byte("Hello, World! This is a benchmark test.")
	for i := 0; i < b.N; i++ {
		SHA256(data)
	}
}
