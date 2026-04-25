package mstorage

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// 全局计数器用于生成唯一值
var randomCounter int64

// BytesReader 实现了 io.Reader 和 io.Seeker 接口的字节读取器
type BytesReader struct {
	*bytes.Reader
	data []byte
}

// NewBytesReader 创建字节读取器
func NewBytesReader(data []byte) *BytesReader {
	return &BytesReader{
		Reader: bytes.NewReader(data),
		data:   data,
	}
}

// Len 返回数据长度
func (r *BytesReader) Len() int {
	return len(r.data)
}

// Bytes 返回底层字节数组
func (r *BytesReader) Bytes() []byte {
	return r.data
}

// GenerateKey 生成存储键名
// prefix: 前缀目录，如 "uploads/2024/01"
// filename: 原始文件名
// keepExt: 是否保留扩展名
func GenerateKey(prefix, filename string, keepExt bool) string {
	// 清理文件名
	filename = filepath.Base(filename)
	filename = strings.TrimSpace(filename)

	// 生成时间戳
	timestamp := time.Now().Format("20060102_150405")

	// 生成随机字符串
	randomStr := generateRandomString(8)

	// 构建 key
	var key string
	if keepExt {
		ext := filepath.Ext(filename)
		name := strings.TrimSuffix(filename, ext)
		// 清理文件名中的特殊字符
		name = sanitizeFilename(name)
		if name == "" {
			name = "file"
		}
		key = filepath.Join(prefix, timestamp+"_"+randomStr+"_"+name+ext)
	} else {
		key = filepath.Join(prefix, timestamp+"_"+randomStr)
	}

	// 统一使用正斜杠
	key = strings.ReplaceAll(key, "\\", "/")

	return key
}

// GenerateUniqueKey 生成唯一存储键名（使用时间戳和随机数）
func GenerateUniqueKey(prefix, ext string) string {
	timestamp := time.Now().Format("20060102150405")
	randomStr := generateRandomString(16)

	key := timestamp + "_" + randomStr
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	key += ext

	if prefix != "" {
		key = strings.TrimSuffix(prefix, "/") + "/" + key
	}

	return key
}

// GetContentType 根据文件扩展名获取 MIME 类型
func GetContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	contentTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".bmp":  "image/bmp",
		".webp": "image/webp",
		".svg":  "image/svg+xml",
		".ico":  "image/x-icon",

		".mp4": "video/mp4",
		".avi": "video/x-msvideo",
		".mov": "video/quicktime",
		".wmv": "video/x-ms-wmv",
		".flv": "video/x-flv",
		".mkv": "video/x-matroska",

		".mp3": "audio/mpeg",
		".wav": "audio/wav",
		".ogg": "audio/ogg",
		".wma": "audio/x-ms-wma",
		".m4a": "audio/mp4",

		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".ppt":  "application/vnd.ms-powerpoint",
		".pptx": "application/vnd.openxmlformats-officedocument.presentationml.presentation",

		".zip": "application/zip",
		".rar": "application/vnd.rar",
		".7z":  "application/x-7z-compressed",
		".tar": "application/x-tar",
		".gz":  "application/gzip",

		".txt":  "text/plain",
		".html": "text/html",
		".htm":  "text/html",
		".css":  "text/css",
		".js":   "application/javascript",
		".json": "application/json",
		".xml":  "application/xml",

		".csv": "text/csv",
		".rtf": "application/rtf",
	}

	if ct, ok := contentTypes[ext]; ok {
		return ct
	}

	return "application/octet-stream"
}

// IsImage 判断是否为图片文件
func IsImage(filename string) bool {
	contentType := GetContentType(filename)
	return strings.HasPrefix(contentType, "image/")
}

// IsVideo 判断是否为视频文件
func IsVideo(filename string) bool {
	contentType := GetContentType(filename)
	return strings.HasPrefix(contentType, "video/")
}

// IsAudio 判断是否为音频文件
func IsAudio(filename string) bool {
	contentType := GetContentType(filename)
	return strings.HasPrefix(contentType, "audio/")
}

// CalculateMD5 计算数据的 MD5 值
func CalculateMD5(data []byte) string {
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}

// CalculateReaderMD5 计算 reader 的 MD5 值
func CalculateReaderMD5(reader io.Reader) (string, error) {
	hash := md5.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// sanitizeFilename 清理文件名中的特殊字符
func sanitizeFilename(name string) string {
	// 替换特殊字符
	replacer := strings.NewReplacer(
		" ", "_",
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(name)
}

// generateRandomString 生成指定长度的随机字符串
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)

	// 使用纳秒时间戳 + 原子计数器作为种子
	counter := atomic.AddInt64(&randomCounter, 1)
	seed := time.Now().UnixNano() + counter

	for i := range result {
		seed = (seed*1103515245 + 12345) & 0x7fffffff
		result[i] = charset[seed%int64(len(charset))]
	}

	return string(result)
}

// FormatSize 格式化文件大小
func FormatSize(size int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)

	switch {
	case size >= TB:
		return formatFloat(float64(size)/float64(TB)) + " TB"
	case size >= GB:
		return formatFloat(float64(size)/float64(GB)) + " GB"
	case size >= MB:
		return formatFloat(float64(size)/float64(MB)) + " MB"
	case size >= KB:
		return formatFloat(float64(size)/float64(KB)) + " KB"
	default:
		return strconv.FormatInt(size, 10) + " B"
	}
}

func formatFloat(f float64) string {
	if f == float64(int64(f)) {
		return strconv.FormatInt(int64(f), 10)
	}
	return strconv.FormatFloat(f, 'f', 2, 64)
}
