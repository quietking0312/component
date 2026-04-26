# mcyptos - 加密与哈希工具组件

提供对称加密、Base64 编解码、多种哈希算法等常用密码学工具。

## 特性

- 🔒 **AES 对称加密**：支持 AES-128/192/256，自动 PKCS7 填充
- 🔑 **多种哈希算法**：MD5、SHA1、SHA256、SHA512、HMAC 系列
- 📦 **Base64 编解码**：标准 Base64 编码和解码
- ⚡ **批量哈希**：支持同时计算多种哈希值

## 快速开始

### AES 加密解密

```go
package main

import (
    "fmt"
    "github.com/quietking0312/component/mcyptos"
)

func main() {
    key := []byte("1234567890123456") // 16 字节对应 AES-128
    iv := []byte("1234567890123456")  // 16 字节初始化向量

    cipher, err := mcyptos.NewAESCipher(key, iv)
    if err != nil {
        panic(err)
    }

    plaintext := []byte("Hello, World!")
    ciphertext, err := cipher.Encrypt(plaintext)
    if err != nil {
        panic(err)
    }

    decrypted, err := cipher.Decrypt(ciphertext)
    if err != nil {
        panic(err)
    }

    fmt.Println(string(decrypted)) // Hello, World!
}
```

### Base64 编解码

```go
encoded := mcyptos.EncodeBase64([]byte("hello"))
decoded, err := mcyptos.DecodeBase64(encoded)
```

### 哈希计算

```go
// MD5
md5Bytes := mcyptos.MD5Partial([]byte("data"))

// HMAC
hmacSha256 := mcyptos.HMACSHA256String("secret-key", "data")

// 同时计算多种哈希
hash := mcyptos.CalculateAll([]byte("data"))
// hash.MD5, hash.SHA1, hash.SHA256, hash.SHA512
```

## API 文档

### AES 加密

| 函数/方法 | 说明 |
|-----------|------|
| `NewAESCipher(key, iv []byte) (*AESCipher, error)` | 创建 AES 加密器 |
| `(c *AESCipher) Encrypt(plaintext []byte) ([]byte, error)` | 加密 |
| `(c *AESCipher) Decrypt(ciphertext []byte) ([]byte, error)` | 解密 |

### Base64

| 函数 | 说明 |
|------|------|
| `EncodeBase64(src []byte) string` | Base64 编码 |
| `DecodeBase64(src string) ([]byte, error)` | Base64 解码 |

### 哈希

| 函数 | 说明 |
|------|------|
| `MD5Partial(data []byte) []byte` | MD5 哈希 |
| `HMACMD5(key, data []byte) string` | HMAC-MD5 |
| `HMACSHA256(key, data []byte) string` | HMAC-SHA256 |
| `HMACSHA512(key, data []byte) string` | HMAC-SHA512 |
| `CalculateAll(data []byte) *MultiHash` | 同时计算 MD5/SHA1/SHA256/SHA512 |

## 注意事项

1. **密钥长度**：AES 密钥必须是 16/24/32 字节，分别对应 AES-128/192/256
2. **IV 长度**：初始化向量长度必须为 16 字节；如果传入 nil，默认使用 key 的前 16 字节
3. **安全建议**：生产环境请勿硬编码密钥，应使用密钥管理服务

## 测试

```bash
cd mcyptos
go test -v
```
