package mhash

import (
	"bytes"
	"fmt"
	"log"
)

// ExampleMD5 计算 MD5 示例
func ExampleMD5() {
	data := []byte("Hello, World!")
	hash := MD5(data)
	fmt.Printf("MD5: %s\n", hash)
	// Output: MD5: 65a8e27d8879283831b664bd8b7f0ad4
}

// ExampleMD5String 计算字符串 MD5 示例
func ExampleMD5String() {
	hash := MD5String("Hello, World!")
	fmt.Printf("MD5: %s\n", hash)
	// Output: MD5: 65a8e27d8879283831b664bd8b7f0ad4
}

// ExampleMD5File 计算文件 MD5 示例
func ExampleMD5File() {
	// 假设有一个文件 /path/to/file.txt
	hash, err := MD5File("/path/to/file.txt")
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("File MD5: %s\n", hash)
}

// ExampleMD5Reader 计算 Reader MD5 示例
func ExampleMD5Reader() {
	data := []byte("Hello, World!")
	reader := bytes.NewReader(data)

	hash, err := MD5Reader(reader)
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}
	fmt.Printf("MD5: %s\n", hash)
	// Output: MD5: 65a8e27d8879283831b664bd8b7f0ad4
}

// ExampleSHA256 计算 SHA256 示例
func ExampleSHA256() {
	data := []byte("Hello, World!")
	hash := SHA256(data)
	fmt.Printf("SHA256: %s\n", hash)
	// Output: SHA256: dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f
}

// ExampleSHA256String 计算字符串 SHA256 示例
func ExampleSHA256String() {
	hash := SHA256String("Hello, World!")
	fmt.Printf("SHA256: %s\n", hash)
	// Output: SHA256: dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f
}

// ExampleHMACMD5 计算 HMAC-MD5 示例
func ExampleHMACMD5() {
	key := []byte("my-secret-key")
	data := []byte("Hello, World!")

	hash := HMACMD5(key, data)
	fmt.Printf("HMAC-MD5: %s\n", hash)
}

// ExampleHMACSHA256 计算 HMAC-SHA256 示例
func ExampleHMACSHA256() {
	key := []byte("my-secret-key")
	data := []byte("Hello, World!")

	hash := HMACSHA256(key, data)
	fmt.Printf("HMAC-SHA256: %s\n", hash)
}

// ExampleHasher 使用 Hasher 流式计算示例
func ExampleHasher() {
	// 创建 MD5 计算器
	hasher := NewMD5()

	// 分多次写入数据
	hasher.WriteString("Hello, ")
	hasher.WriteString("World!")

	// 获取结果
	hash := hasher.Sum()
	fmt.Printf("MD5: %s\n", hash)
	// Output: MD5: 65a8e27d8879283831b664bd8b7f0ad4
}

// ExampleHasher_reset 重置 Hasher 示例
func ExampleHasher_reset() {
	hasher := NewMD5()

	// 第一次计算
	hasher.WriteString("Hello")
	hash1 := hasher.Sum()
	fmt.Printf("First: %s\n", hash1)

	// 重置后计算
	hasher.Reset()
	hasher.WriteString("World")
	hash2 := hasher.Sum()
	fmt.Printf("Second: %s\n", hash2)
}

// ExampleCalculateAll 同时计算多种哈希示例
func ExampleCalculateAll() {
	data := []byte("Hello, World!")

	result := CalculateAll(data)

	fmt.Printf("MD5: %s\n", result.MD5)
	fmt.Printf("SHA1: %s\n", result.SHA1)
	fmt.Printf("SHA256: %s\n", result.SHA256)
	fmt.Printf("SHA512: %s\n", result.SHA512)

	// Output:
	// MD5: 65a8e27d8879283831b664bd8b7f0ad4
	// SHA1: 0a0a9f2a6772942557ab5355d76af442f8f65e01
	// SHA256: dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f
	// SHA512: 374d794a95cdcfd8b35993185fef9ba368f160d8daf432d08ba9f1ed1e5abe6cc69291e0fa2fe0006a52570ef18c19def4e617c33ce52ef0a6e5fbe318cb0387
}

// ExampleCalculateAllFile 同时计算文件多种哈希示例
func ExampleCalculateAllFile() {
	// 假设有一个文件 /path/to/file.txt
	result, size, err := CalculateAllFile("/path/to/file.txt")
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("File size: %d bytes\n", size)
	fmt.Printf("MD5: %s\n", result.MD5)
	fmt.Printf("SHA256: %s\n", result.SHA256)
}
