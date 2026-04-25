package mstorage

import (
	"context"
	"io"
	"time"
)

// UploadResult 上传结果
type UploadResult struct {
	// 文件访问 URL
	URL string `json:"url"`
	// 文件在存储桶中的 key
	Key string `json:"key"`
	// 文件大小（字节）
	Size int64 `json:"size"`
	// 文件 MIME 类型
	ContentType string `json:"content_type"`
	// ETag 或文件哈希
	ETag string `json:"etag,omitempty"`
}

// Provider 云存储提供商接口
type Provider interface {
	// Upload 上传文件
	// ctx: 上下文
	// key: 文件在存储桶中的路径/名称
	// reader: 文件内容读取器
	// size: 文件大小（字节），-1 表示未知
	// contentType: 文件 MIME 类型，为空则自动检测
	Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*UploadResult, error)

	// UploadBytes 上传字节数组
	UploadBytes(ctx context.Context, key string, data []byte, contentType string) (*UploadResult, error)

	// Delete 删除文件
	Delete(ctx context.Context, key string) error

	// GetURL 获取文件访问 URL（私有桶需要签名）
	// expires: URL 过期时间，为零则使用默认过期时间
	GetURL(ctx context.Context, key string, expires time.Duration) (string, error)

	// Exists 检查文件是否存在
	Exists(ctx context.Context, key string) (bool, error)

	// GetProviderName 获取提供商名称
	GetProviderName() string
}

// ProviderType 云存储提供商类型
type ProviderType string

const (
	// ProviderCOS 腾讯云 COS
	ProviderCOS ProviderType = "cos"
	// ProviderOSS 阿里云 OSS
	ProviderOSS ProviderType = "oss"
	// ProviderBOS 百度云 BOS
	ProviderBOS ProviderType = "bos"
	// ProviderS3 AWS S3
	ProviderS3 ProviderType = "s3"
	// ProviderUFile UCloud UFile
	ProviderUFile ProviderType = "ufile"
)

// IsValid 检查提供商类型是否有效
func (pt ProviderType) IsValid() bool {
	switch pt {
	case ProviderCOS, ProviderOSS, ProviderBOS, ProviderS3, ProviderUFile:
		return true
	}
	return false
}
