package mcyptos

import (
	"encoding/base64"
)

// EncodeBase64 Base64 编码
func EncodeBase64(src []byte) string {
	return base64.StdEncoding.EncodeToString(src)
}

// DecodeBase64 Base64 解码
func DecodeBase64(src string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(src)
}
