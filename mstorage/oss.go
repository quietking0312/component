package mstorage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// OSSConfig 阿里云 OSS 配置（内部使用，从 UnifiedConfig 转换）
type OSSConfig struct {
	Bucket          string
	Endpoint        string
	AccessKeyID     string
	AccessKeySecret string
	UseToken        bool
	Token           string
}

// Validate 验证配置
func (c OSSConfig) Validate() error {
	if c.Bucket == "" {
		return fmt.Errorf("bucket is required")
	}
	if c.Endpoint == "" {
		return fmt.Errorf("endpoint is required (or set region to auto-generate)")
	}
	if c.AccessKeyID == "" {
		return fmt.Errorf("access_key_id is required")
	}
	if c.AccessKeySecret == "" {
		return fmt.Errorf("access_key_secret is required")
	}
	return nil
}

// OSSProvider 阿里云 OSS 提供商
type OSSProvider struct {
	client *oss.Client
	bucket *oss.Bucket
	config OSSConfig
}

// NewOSSProvider 创建 OSS 提供商
func NewOSSProvider(config OSSConfig) (*OSSProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("oss: %w", err)
	}

	// 创建客户端
	var client *oss.Client
	var err error

	if config.UseToken && config.Token != "" {
		client, err = oss.New(config.Endpoint, config.AccessKeyID, config.AccessKeySecret, oss.SecurityToken(config.Token))
	} else {
		client, err = oss.New(config.Endpoint, config.AccessKeyID, config.AccessKeySecret)
	}

	if err != nil {
		return nil, fmt.Errorf("oss: create client failed: %w", err)
	}

	// 获取存储桶
	bucket, err := client.Bucket(config.Bucket)
	if err != nil {
		return nil, fmt.Errorf("oss: get bucket failed: %w", err)
	}

	return &OSSProvider{
		client: client,
		bucket: bucket,
		config: config,
	}, nil
}

// Upload 上传文件
func (p *OSSProvider) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*UploadResult, error) {
	var options []oss.Option
	if contentType != "" {
		options = append(options, oss.ContentType(contentType))
	}

	err := p.bucket.PutObject(key, reader, options...)
	if err != nil {
		return nil, fmt.Errorf("oss: upload failed: %w", err)
	}

	// 获取文件 URL
	urlStr := fmt.Sprintf("https://%s.%s/%s", p.config.Bucket, p.config.Endpoint, key)

	return &UploadResult{
		URL:         urlStr,
		Key:         key,
		Size:        size,
		ContentType: contentType,
	}, nil
}

// UploadBytes 上传字节数组
func (p *OSSProvider) UploadBytes(ctx context.Context, key string, data []byte, contentType string) (*UploadResult, error) {
	return p.Upload(ctx, key, NewBytesReader(data), int64(len(data)), contentType)
}

// Delete 删除文件
func (p *OSSProvider) Delete(ctx context.Context, key string) error {
	err := p.bucket.DeleteObject(key)
	if err != nil {
		return fmt.Errorf("oss: delete failed: %w", err)
	}
	return nil
}

// GetURL 获取文件访问 URL
func (p *OSSProvider) GetURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	if expires <= 0 {
		expires = 15 * time.Minute
	}

	// 生成签名 URL
	signedURL, err := p.bucket.SignURL(key, oss.HTTPGet, int64(expires.Seconds()))
	if err != nil {
		return "", fmt.Errorf("oss: sign url failed: %w", err)
	}

	return signedURL, nil
}

// Exists 检查文件是否存在
func (p *OSSProvider) Exists(ctx context.Context, key string) (bool, error) {
	exists, err := p.bucket.IsObjectExist(key)
	if err != nil {
		return false, fmt.Errorf("oss: check exists failed: %w", err)
	}
	return exists, nil
}

// GetProviderName 获取提供商名称
func (p *OSSProvider) GetProviderName() string {
	return string(ProviderOSS)
}
