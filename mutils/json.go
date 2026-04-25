package utils

import (
	"bytes"
	"encoding/json"
	"io"
)

var (
	// Marshal 序列化对象为 JSON 字节数组
	Marshal = json.Marshal
	// MarshalIndent 格式化序列化对象为 JSON 字节数组
	MarshalIndent = json.MarshalIndent
	// Unmarshal 解析 JSON 字节数组到对象
	Unmarshal = json.Unmarshal
	// NewEncoder 创建 JSON 编码器
	NewEncoder = json.NewEncoder
	// NewDecoder 创建 JSON 解码器
	NewDecoder = json.NewDecoder
	// Valid 验证 JSON 数据是否有效
	Valid = json.Valid
)

// MarshalToString 序列化对象为 JSON 字符串
func MarshalToString(v any) (string, error) {
	data, err := Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// MustMarshal 序列化对象为 JSON 字节数组，忽略错误
func MustMarshal(v any) []byte {
	data, _ := Marshal(v)
	return data
}

// MustMarshalToString 序列化对象为 JSON 字符串，忽略错误
func MustMarshalToString(v any) string {
	return string(MustMarshal(v))
}

// MustMarshalIndent 格式化序列化对象为 JSON 字节数组，忽略错误
func MustMarshalIndent(v any, prefix, indent string) []byte {
	data, _ := MarshalIndent(v, prefix, indent)
	return data
}

// UnmarshalFromString 从 JSON 字符串解析到对象
func UnmarshalFromString(str string, v any) error {
	return Unmarshal([]byte(str), v)
}

// JSONReader 从 io.Reader 解析 JSON
func JSONReader(r io.Reader, v any) error {
	return NewDecoder(r).Decode(v)
}

// JSONWriter 将对象写入 io.Writer
func JSONWriter(w io.Writer, v any) error {
	return NewEncoder(w).Encode(v)
}

// ToMap 将对象转换为 map[string]any
func ToMap(v any) (map[string]any, error) {
	data, err := Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	err = Unmarshal(data, &m)
	return m, err
}

// FromMap 将 map[string]any 转换为指定类型的对象
func FromMap(m map[string]any, v any) error {
	data, err := Marshal(m)
	if err != nil {
		return err
	}
	return Unmarshal(data, v)
}

// Compact 移除 JSON 中的空白字符
func Compact(dst *bytes.Buffer, src []byte) error {
	return json.Compact(dst, src)
}

// HTMLEscape 将 JSON 中的特殊 HTML 字符转义
func HTMLEscape(dst *bytes.Buffer, src []byte) {
	json.HTMLEscape(dst, src)
}

// Indent 格式化 JSON 添加缩进
func Indent(dst *bytes.Buffer, src []byte, prefix, indent string) error {
	return json.Indent(dst, src, prefix, indent)
}
