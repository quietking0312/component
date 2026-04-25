package mstorage

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/baidubce/bce-sdk-go/services/bos"
	"github.com/baidubce/bce-sdk-go/services/bos/api"
)

// BOSConfig 百度云 BOS 配置（内部使用，从 UnifiedConfig 转换）
type BOSConfig struct {
	Bucket          string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	UseToken        bool
	Token           string
}

// Validate 验证配置
func (c BOSConfig) Validate() error {
	if c.Bucket == "" {
		return fmt.Errorf("bucket is required")
	}
	if c.Endpoint == "" {
		return fmt.Errorf("endpoint is required (or set region to auto-generate)")
	}
	if c.AccessKeyID == "" {
		return fmt.Errorf("access_key_id is required")
	}
	if c.SecretAccessKey == "" {
		return fmt.Errorf("secret_access_key is required")
	}
	return nil
}

// BOSProvider 百度云 BOS 提供商
type BOSProvider struct {
	client *bos.Client
	config BOSConfig
	bucket string
}

// NewBOSProvider 创建 BOS 提供商
func NewBOSProvider(config BOSConfig) (*BOSProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("bos: %w", err)
	}

	// 构建 endpoint
	endpoint := config.Endpoint
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		endpoint = "https://" + endpoint
	}

	var client *bos.Client
	var err error

	// 创建客户端
	if config.UseToken && config.Token != "" {
		// 使用 STS 临时凭证
		client, err = bos.NewStsClient(config.AccessKeyID, config.SecretAccessKey, endpoint, 3600)
	} else {
		client, err = bos.NewClient(config.AccessKeyID, config.SecretAccessKey, endpoint)
	}

	if err != nil {
		return nil, fmt.Errorf("bos: create client failed: %w", err)
	}

	return &BOSProvider{
		client: client,
		config: config,
		bucket: config.Bucket,
	}, nil
}

// Upload 上传文件
func (p *BOSProvider) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*UploadResult, error) {
	args := &api.PutObjectArgs{}
	if contentType != "" {
		args.ContentType = contentType
	}

	// 使用 PutObjectFromStreamWithContext 上传
	result, err := p.client.PutObjectFromStreamWithContext(ctx, p.bucket, key, reader, args)
	if err != nil {
		return nil, fmt.Errorf("bos: upload failed: %w", err)
	}

	// 构建文件 URL
	scheme := "https"
	urlStr := fmt.Sprintf("%s://%s.%s/%s", scheme, p.bucket, p.config.Endpoint, key)

	return &UploadResult{
		URL:         urlStr,
		Key:         key,
		Size:        size,
		ContentType: contentType,
		ETag:        result,
	}, nil
}

// UploadBytes 上传字节数组
func (p *BOSProvider) UploadBytes(ctx context.Context, key string, data []byte, contentType string) (*UploadResult, error) {
	return p.Upload(ctx, key, NewBytesReader(data), int64(len(data)), contentType)
}

// Delete 删除文件
func (p *BOSProvider) Delete(ctx context.Context, key string) error {
	err := p.client.DeleteObject(p.bucket, key)
	if err != nil {
		return fmt.Errorf("bos: delete failed: %w", err)
	}
	return nil
}

// GetURL 获取文件访问 URL
func (p *BOSProvider) GetURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	if expires <= 0 {
		expires = 15 * time.Minute
	}

	// 生成预签名 URL
	expireSeconds := int(expires.Seconds())
	signedURL := p.client.BasicGeneratePresignedUrl(p.bucket, key, expireSeconds)

	return signedURL, nil
}

// Exists 检查文件是否存在
func (p *BOSProvider) Exists(ctx context.Context, key string) (bool, error) {
	_, err := p.client.GetObjectMeta(p.bucket, key)
	if err != nil {
		// 检查是否为 404 错误
		if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "NoSuchKey") {
			return false, nil
		}
		return false, fmt.Errorf("bos: check exists failed: %w", err)
	}
	return true, nil
}

// GetProviderName 获取提供商名称
func (p *BOSProvider) GetProviderName() string {
	return string(ProviderBOS)
}
