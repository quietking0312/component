package mcyptos

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"fmt"
)

// Cipher 对称加密接口
type Cipher interface {
	// Encrypt 加密明文，返回密文
	Encrypt(plaintext []byte) ([]byte, error)
	// Decrypt 解密密文，返回明文
	Decrypt(ciphertext []byte) ([]byte, error)
	// BlockSize 返回块大小
	BlockSize() int
}

// AESCipher AES 加密器（支持 AES-128/192/256，CBC 模式）
//
// 使用示例：
//
//	c, err := mcyptos.NewAESCipher(key, iv)
//	if err != nil { ... }
//	cipher, err := c.Encrypt([]byte("hello"))
//	plain, err := c.Decrypt(cipher)
type AESCipher struct {
	block cipher.Block
	iv    []byte
}

// NewAESCipher 创建 AES 加密器
//   - key: 密钥，长度必须是 16、24 或 32 字节（分别对应 AES-128/192/256）
//   - iv: 初始化向量，长度必须为 16 字节；如果为 nil，则默认使用 key 的前 16 字节
func NewAESCipher(key, iv []byte) (*AESCipher, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes: invalid key: %w", err)
	}
	blockSize := block.BlockSize()

	if iv == nil {
		if len(key) < blockSize {
			return nil, fmt.Errorf("aes: key length must be at least %d bytes when iv is nil", blockSize)
		}
		iv = key[:blockSize]
	}
	if len(iv) != blockSize {
		return nil, fmt.Errorf("aes: iv length must be %d bytes", blockSize)
	}

	// 复制 iv，避免外部修改影响内部状态
	ivCopy := make([]byte, blockSize)
	copy(ivCopy, iv)

	return &AESCipher{
		block: block,
		iv:    ivCopy,
	}, nil
}

// Encrypt 使用 CBC 模式加密明文，采用 PKCS7 填充
func (c *AESCipher) Encrypt(plaintext []byte) ([]byte, error) {
	blockSize := c.block.BlockSize()
	padded := pkcs7Padding(plaintext, blockSize)
	ciphertext := make([]byte, len(padded))

	blockMode := cipher.NewCBCEncrypter(c.block, c.iv)
	blockMode.CryptBlocks(ciphertext, padded)
	return ciphertext, nil
}

// Decrypt 使用 CBC 模式解密密文，去除 PKCS7 填充
func (c *AESCipher) Decrypt(ciphertext []byte) ([]byte, error) {
	blockSize := c.block.BlockSize()
	if len(ciphertext)%blockSize != 0 {
		return nil, fmt.Errorf("aes: ciphertext length is not a multiple of block size")
	}

	plaintext := make([]byte, len(ciphertext))
	blockMode := cipher.NewCBCDecrypter(c.block, c.iv)
	blockMode.CryptBlocks(plaintext, ciphertext)

	return pkcs7UnPadding(plaintext)
}

// BlockSize 返回 AES 块大小（固定为 16 字节）
func (c *AESCipher) BlockSize() int {
	return c.block.BlockSize()
}

// pkcs7Padding 对数据进行 PKCS7 填充
func pkcs7Padding(data []byte, blockSize int) []byte {
	padNum := blockSize - len(data)%blockSize
	pad := bytes.Repeat([]byte{byte(padNum)}, padNum)
	return append(data, pad...)
}

// pkcs7UnPadding 去除 PKCS7 填充
func pkcs7UnPadding(data []byte) ([]byte, error) {
	n := len(data)
	if n == 0 {
		return data, nil
	}
	padNum := int(data[n-1])
	if padNum == 0 || padNum > aes.BlockSize || padNum > n {
		return nil, fmt.Errorf("aes: invalid PKCS7 padding")
	}
	return data[:n-padNum], nil
}
