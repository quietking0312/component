package mstorage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Config AWS S3 配置（内部使用，从 UnifiedConfig 转换）
type S3Config struct {
	Bucket          string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	UseToken        bool
	Token           string
	Endpoint        string
	UsePathStyle    bool
}

// Validate 验证配置
func (c S3Config) Validate() error {
	if c.Bucket == "" {
		return fmt.Errorf("bucket is required")
	}
	if c.Region == "" {
		return fmt.Errorf("region is required")
	}
	if c.AccessKeyID == "" {
		return fmt.Errorf("access_key_id is required")
	}
	if c.SecretAccessKey == "" {
		return fmt.Errorf("secret_access_key is required")
	}
	return nil
}

// S3Provider AWS S3 提供商
type S3Provider struct {
	client *s3.Client
	config S3Config
}

// NewS3Provider 创建 S3 提供商
func NewS3Provider(s3Config S3Config) (*S3Provider, error) {
	if err := s3Config.Validate(); err != nil {
		return nil, fmt.Errorf("s3: %w", err)
	}

	// 构建 AWS 配置
	var cfg aws.Config
	var err error

	// 创建凭证提供者
	var creds credentials.StaticCredentialsProvider
	if s3Config.UseToken && s3Config.Token != "" {
		creds = credentials.NewStaticCredentialsProvider(
			s3Config.AccessKeyID,
			s3Config.SecretAccessKey,
			s3Config.Token,
		)
	} else {
		creds = credentials.NewStaticCredentialsProvider(
			s3Config.AccessKeyID,
			s3Config.SecretAccessKey,
			"",
		)
	}

	// 加载 AWS 配置
	cfg, err = config.LoadDefaultConfig(context.Background(),
		config.WithRegion(s3Config.Region),
		config.WithCredentialsProvider(creds),
	)
	if err != nil {
		return nil, fmt.Errorf("s3: load config failed: %w", err)
	}

	// 创建 S3 客户端
	var client *s3.Client
	if s3Config.Endpoint != "" {
		client = s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(s3Config.Endpoint)
			o.UsePathStyle = s3Config.UsePathStyle
		})
	} else {
		client = s3.NewFromConfig(cfg)
	}

	return &S3Provider{
		client: client,
		config: s3Config,
	}, nil
}

// Upload 上传文件
func (p *S3Provider) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*UploadResult, error) {
	input := &s3.PutObjectInput{
		Bucket: aws.String(p.config.Bucket),
		Key:    aws.String(key),
		Body:   reader,
	}

	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}

	if size > 0 {
		input.ContentLength = aws.Int64(size)
	}

	result, err := p.client.PutObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("s3: upload failed: %w", err)
	}

	// 构建文件 URL
	urlStr := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", p.config.Bucket, p.config.Region, key)
	if p.config.Endpoint != "" {
		if p.config.UsePathStyle {
			urlStr = fmt.Sprintf("%s/%s/%s", p.config.Endpoint, p.config.Bucket, key)
		} else {
			urlStr = fmt.Sprintf("%s/%s", p.config.Endpoint, key)
		}
	}

	return &UploadResult{
		URL:         urlStr,
		Key:         key,
		Size:        size,
		ContentType: contentType,
		ETag:        aws.ToString(result.ETag),
	}, nil
}

// UploadBytes 上传字节数组
func (p *S3Provider) UploadBytes(ctx context.Context, key string, data []byte, contentType string) (*UploadResult, error) {
	return p.Upload(ctx, key, NewBytesReader(data), int64(len(data)), contentType)
}

// Delete 删除文件
func (p *S3Provider) Delete(ctx context.Context, key string) error {
	_, err := p.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(p.config.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("s3: delete failed: %w", err)
	}
	return nil
}

// GetURL 获取文件访问 URL
func (p *S3Provider) GetURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	if expires <= 0 {
		expires = 15 * time.Minute
	}

	// 生成预签名 URL
	presignClient := s3.NewPresignClient(p.client)
	req, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(p.config.Bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expires))

	if err != nil {
		return "", fmt.Errorf("s3: presign failed: %w", err)
	}

	return req.URL, nil
}

// Exists 检查文件是否存在
func (p *S3Provider) Exists(ctx context.Context, key string) (bool, error) {
	_, err := p.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(p.config.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var notFound *types.NotFound
		if fmt.Sprintf("%T", err) == "*smithy.OperationError" {
			// AWS SDK v2 使用 smithy 错误类型
			return false, nil
		}
		// 尝试判断是否为 404 错误
		if fmt.Sprint(err) == "NotFound" {
			return false, nil
		}
		_ = notFound // 避免未使用警告
		return false, fmt.Errorf("s3: check exists failed: %w", err)
	}
	return true, nil
}

// GetProviderName 获取提供商名称
func (p *S3Provider) GetProviderName() string {
	return string(ProviderS3)
}
