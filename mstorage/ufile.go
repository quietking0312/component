package mstorage

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// UFileConfig UCloud UFile 配置（内部使用，从 UnifiedConfig 转换）
type UFileConfig struct {
	Bucket     string
	Region     string
	PublicKey  string
	PrivateKey string
	FileHost   string
	UseToken   bool
	Token      string
}

// Validate 验证配置
func (c UFileConfig) Validate() error {
	if c.Bucket == "" {
		return fmt.Errorf("bucket is required")
	}
	if c.Region == "" {
		return fmt.Errorf("region is required")
	}
	if c.PublicKey == "" {
		return fmt.Errorf("public_key is required")
	}
	if c.PrivateKey == "" {
		return fmt.Errorf("private_key is required")
	}
	return nil
}

// UFileProvider UCloud UFile 提供商
type UFileProvider struct {
	config     UFileConfig
	httpClient *http.Client
	bucket     string
	baseURL    string
}

// NewUFileProvider 创建 UFile 提供商
func NewUFileProvider(config UFileConfig) (*UFileProvider, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("ufile: %w", err)
	}

	// 构建基础 URL
	var baseURL string
	if config.FileHost != "" {
		baseURL = fmt.Sprintf("https://%s", config.FileHost)
	} else {
		baseURL = fmt.Sprintf("https://%s.ufile.ucloud.cn", config.Bucket)
	}

	return &UFileProvider{
		config:     config,
		httpClient: &http.Client{Timeout: 120 * time.Second},
		bucket:     config.Bucket,
		baseURL:    baseURL,
	}, nil
}

// generateAuth 生成 UFile 认证头
func (p *UFileProvider) generateAuth(method, key string, contentLength int64, contentMD5, contentType string) string {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)

	// 构建待签名字符串
	// Method\nContent-MD5\nContent-Type\nDate\nHeaders\nResource
	signStr := fmt.Sprintf("%s\n%s\n%s\n%s\n\n/%s/%s",
		strings.ToUpper(method),
		contentMD5,
		contentType,
		timestamp,
		p.bucket,
		key,
	)

	// 使用 HMAC-SHA256 签名
	h := hmac.New(sha256.New, []byte(p.config.PrivateKey))
	h.Write([]byte(signStr))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	// 返回 Authorization 头
	return fmt.Sprintf("UCloud %s:%s", p.config.PublicKey, signature)
}

// Upload 上传文件
func (p *UFileProvider) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*UploadResult, error) {
	// 读取数据
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("ufile: read data failed: %w", err)
	}

	// 如果没有提供 contentType，使用默认的
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// 构建请求 URL
	reqURL := fmt.Sprintf("%s/%s", p.baseURL, key)

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, reqURL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("ufile: create request failed: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Content-Length", strconv.Itoa(len(data)))

	// 生成认证头
	auth := p.generateAuth("PUT", key, int64(len(data)), "", contentType)
	req.Header.Set("Authorization", auth)

	// 发送请求
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ufile: upload request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ufile: upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	// 构建文件 URL
	var urlStr string
	if p.config.FileHost != "" {
		urlStr = fmt.Sprintf("https://%s/%s", p.config.FileHost, key)
	} else {
		urlStr = fmt.Sprintf("https://%s.ufile.ucloud.cn/%s", p.bucket, key)
	}

	return &UploadResult{
		URL:         urlStr,
		Key:         key,
		Size:        int64(len(data)),
		ContentType: contentType,
	}, nil
}

// UploadBytes 上传字节数组
func (p *UFileProvider) UploadBytes(ctx context.Context, key string, data []byte, contentType string) (*UploadResult, error) {
	return p.Upload(ctx, key, bytes.NewReader(data), int64(len(data)), contentType)
}

// Delete 删除文件
func (p *UFileProvider) Delete(ctx context.Context, key string) error {
	// 构建请求 URL
	reqURL := fmt.Sprintf("%s/%s", p.baseURL, key)

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, reqURL, nil)
	if err != nil {
		return fmt.Errorf("ufile: create request failed: %w", err)
	}

	// 生成认证头
	auth := p.generateAuth("DELETE", key, 0, "", "")
	req.Header.Set("Authorization", auth)

	// 发送请求
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ufile: delete request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ufile: delete failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetURL 获取文件访问 URL
func (p *UFileProvider) GetURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	if expires <= 0 {
		expires = 15 * time.Minute
	}

	// 构建文件 URL
	var urlStr string
	if p.config.FileHost != "" {
		urlStr = fmt.Sprintf("https://%s/%s", p.config.FileHost, key)
	} else {
		urlStr = fmt.Sprintf("https://%s.ufile.ucloud.cn/%s", p.bucket, key)
	}

	// 生成签名
	timestamp := strconv.FormatInt(time.Now().Add(expires).Unix(), 10)
	signStr := fmt.Sprintf("GET\n\n\n%s\n/%s/%s", timestamp, p.bucket, key)

	h := hmac.New(sha256.New, []byte(p.config.PrivateKey))
	h.Write([]byte(signStr))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	// 添加签名参数
	signedURL := fmt.Sprintf("%s?UCloudPublicKey=%s&Signature=%s&Expires=%s",
		urlStr,
		p.config.PublicKey,
		url.QueryEscape(signature),
		timestamp,
	)

	return signedURL, nil
}

// Exists 检查文件是否存在
func (p *UFileProvider) Exists(ctx context.Context, key string) (bool, error) {
	// 构建请求 URL
	reqURL := fmt.Sprintf("%s/%s", p.baseURL, key)

	// 创建 HEAD 请求
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, reqURL, nil)
	if err != nil {
		return false, fmt.Errorf("ufile: create request failed: %w", err)
	}

	// 发送请求
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("ufile: head request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return true, nil
	} else if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	return false, fmt.Errorf("ufile: check exists failed with status %d", resp.StatusCode)
}

// GetProviderName 获取提供商名称
func (p *UFileProvider) GetProviderName() string {
	return string(ProviderUFile)
}
