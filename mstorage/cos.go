package mstorage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// COSConfig 腾讯云 COS 配置（内部使用，从 UnifiedConfig 转换）
type COSConfig struct {
	Bucket    string
	Region    string
	SecretID  string
	SecretKey string
	UseToken  bool
	Token     string
}

// Validate 验证配置
func (c COSConfig) Validate() error {
	if c.Bucket == "" {
		return fmt.Errorf("bucket is required")
	}
	if c.Region == "" {
		return fmt.Errorf("region is required")
	}
	if c.SecretID == "" {
		return fmt.Errorf("secret_id is required")
	}
	if c.SecretKey == "" {
		return fmt.Errorf("secret_key is required")
	}
	return nil
}

// COSProvider 腾讯云 COS 提供商
type COSProvider struct {
	client *cos.Client
	config COSConfig
	bucket string
}

// NewCOSProvider 创建 COS 提供商
func NewCOSProvider(config COSConfig) (*COSProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("cos: %w", err)
	}

	// 构建存储桶 URL
	scheme := "https"
	baseURL, err := url.Parse(fmt.Sprintf("%s://%s.cos.%s.myqcloud.com", scheme, config.Bucket, config.Region))
	if err != nil {
		return nil, fmt.Errorf("cos: failed to parse base url: %w", err)
	}

	// 创建客户端
	client := cos.NewClient(&cos.BaseURL{BucketURL: baseURL}, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  config.SecretID,
			SecretKey: config.SecretKey,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	})

	// 如果使用临时密钥
	if config.UseToken && config.Token != "" {
		client = cos.NewClient(&cos.BaseURL{BucketURL: baseURL}, &http.Client{
			Transport: &cos.AuthorizationTransport{
				SecretID:     config.SecretID,
				SecretKey:    config.SecretKey,
				SessionToken: config.Token,
			},
		})
	}

	return &COSProvider{
		client: client,
		config: config,
		bucket: config.Bucket,
	}, nil
}

// Upload 上传文件
func (p *COSProvider) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*UploadResult, error) {
	opt := &cos.ObjectPutOptions{}
	if contentType != "" {
		opt.ContentType = contentType
	}

	resp, err := p.client.Object.Put(ctx, key, reader, opt)
	if err != nil {
		return nil, fmt.Errorf("cos: upload failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cos: upload failed with status %d", resp.StatusCode)
	}

	// 获取文件 URL
	urlStr := p.client.Object.GetObjectURL(key).String()

	return &UploadResult{
		URL:         urlStr,
		Key:         key,
		Size:        size,
		ContentType: contentType,
		ETag:        resp.Header.Get("ETag"),
	}, nil
}

// UploadBytes 上传字节数组
func (p *COSProvider) UploadBytes(ctx context.Context, key string, data []byte, contentType string) (*UploadResult, error) {
	return p.Upload(ctx, key, NewBytesReader(data), int64(len(data)), contentType)
}

// Delete 删除文件
func (p *COSProvider) Delete(ctx context.Context, key string) error {
	resp, err := p.client.Object.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("cos: delete failed: %w", err)
	}
	defer resp.Body.Close()

	// COS 删除不存在文件返回 204 No Content，也视为成功
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("cos: delete failed with status %d", resp.StatusCode)
	}

	return nil
}

// GetURL 获取文件访问 URL
func (p *COSProvider) GetURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	// 构建预签名 URL
	if expires <= 0 {
		expires = 15 * time.Minute
	}

	presignedURL, err := p.client.Object.GetPresignedURL(ctx, http.MethodGet, key, p.config.SecretID, p.config.SecretKey, expires, nil)
	if err != nil {
		return "", fmt.Errorf("cos: get presigned url failed: %w", err)
	}

	return presignedURL.String(), nil
}

// Exists 检查文件是否存在
func (p *COSProvider) Exists(ctx context.Context, key string) (bool, error) {
	exists, err := p.client.Object.IsExist(ctx, key)
	if err != nil {
		return false, fmt.Errorf("cos: check exists failed: %w", err)
	}
	return exists, nil
}

// GetProviderName 获取提供商名称
func (p *COSProvider) GetProviderName() string {
	return string(ProviderCOS)
}
