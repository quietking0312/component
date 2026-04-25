package mstorage

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestGenerateKey 测试生成存储键名
func TestGenerateKey(t *testing.T) {
	key := GenerateKey("uploads/2024", "test.jpg", true)
	assert.NotEmpty(t, key)
	assert.Contains(t, key, "uploads/2024/")
	assert.Contains(t, key, ".jpg")
	t.Logf("Generated key: %s", key)
}

// TestGenerateUniqueKey 测试生成唯一键名
func TestGenerateUniqueKey(t *testing.T) {
	key1 := GenerateUniqueKey("uploads", "png")
	key2 := GenerateUniqueKey("uploads", ".png")

	assert.NotEmpty(t, key1)
	assert.NotEmpty(t, key2)
	assert.NotEqual(t, key1, key2)
	assert.Contains(t, key1, ".png")
	assert.Contains(t, key2, ".png")
	t.Logf("Key1: %s, Key2: %s", key1, key2)
}

// TestGetContentType 测试获取 MIME 类型
func TestGetContentType(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"test.jpg", "image/jpeg"},
		{"test.png", "image/png"},
		{"test.pdf", "application/pdf"},
		{"test.txt", "text/plain"},
		{"test.unknown", "application/octet-stream"},
	}

	for _, tt := range tests {
		contentType := GetContentType(tt.filename)
		assert.Equal(t, tt.expected, contentType)
	}
}

// TestIsImage 测试图片类型判断
func TestIsImage(t *testing.T) {
	assert.True(t, IsImage("test.jpg"))
	assert.True(t, IsImage("test.png"))
	assert.True(t, IsImage("test.gif"))
	assert.False(t, IsImage("test.pdf"))
	assert.False(t, IsImage("test.mp4"))
}

// TestBytesReader 测试字节读取器
func TestBytesReader(t *testing.T) {
	data := []byte("Hello, World!")
	reader := NewBytesReader(data)

	assert.Equal(t, len(data), reader.Len())
	assert.Equal(t, data, reader.Bytes())

	// 测试读取
	buf := make([]byte, len(data))
	n, err := reader.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, data, buf)
}

// TestCalculateMD5 测试 MD5 计算
func TestCalculateMD5(t *testing.T) {
	data := []byte("Hello, World!")
	md5 := CalculateMD5(data)
	assert.NotEmpty(t, md5)
	assert.Equal(t, 32, len(md5))
	t.Logf("MD5: %s", md5)
}

// TestFormatSize 测试文件大小格式化
func TestFormatSize(t *testing.T) {
	tests := []struct {
		size     int64
		expected string
	}{
		{100, "100 B"},
		{1024, "1 KB"},
		{1024 * 1024, "1 MB"},
		{1024 * 1024 * 1024, "1 GB"},
	}

	for _, tt := range tests {
		result := FormatSize(tt.size)
		assert.Contains(t, result, tt.expected[:len(tt.expected)-2])
	}
}

// TestProviderType 测试提供商类型
func TestProviderType(t *testing.T) {
	assert.True(t, ProviderCOS.IsValid())
	assert.True(t, ProviderOSS.IsValid())
	assert.True(t, ProviderBOS.IsValid())
	assert.True(t, ProviderS3.IsValid())
	assert.True(t, ProviderUFile.IsValid())

	// 测试无效的提供商
	invalidType := ProviderType("invalid")
	assert.False(t, invalidType.IsValid())
}

// TestDefaultConfig 测试默认配置
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.Equal(t, ProviderCOS, config.DefaultProvider)
	assert.True(t, config.UseHTTPS)
	assert.False(t, config.UseCustomDomain)
}

// TestUnifiedConfig 测试统一配置
func TestUnifiedConfig(t *testing.T) {
	config := UnifiedConfig{
		Bucket:    "test-bucket",
		AccessKey: "test-access-key",
		SecretKey: "test-secret-key",
		Region:    "ap-guangzhou",
	}

	assert.False(t, config.IsEmpty())
	assert.NoError(t, config.Validate())

	// 测试转换为 COS 配置
	cosConfig := config.ToCOSConfig()
	assert.Equal(t, config.Bucket, cosConfig.Bucket)
	assert.Equal(t, config.Region, cosConfig.Region)
	assert.Equal(t, config.AccessKey, cosConfig.SecretID)
	assert.Equal(t, config.SecretKey, cosConfig.SecretKey)

	// 测试转换为 OSS 配置（自动推导 endpoint）
	ossConfig := config.ToOSSConfig()
	assert.Equal(t, config.Bucket, ossConfig.Bucket)
	assert.Equal(t, config.AccessKey, ossConfig.AccessKeyID)
	assert.Equal(t, config.SecretKey, ossConfig.AccessKeySecret)
	assert.Contains(t, ossConfig.Endpoint, config.Region)
}

// TestUnifiedConfigIsEmpty 测试空配置检查
func TestUnifiedConfigIsEmpty(t *testing.T) {
	emptyConfig := UnifiedConfig{}
	assert.True(t, emptyConfig.IsEmpty())

	partialConfig := UnifiedConfig{
		Bucket: "test",
	}
	assert.True(t, partialConfig.IsEmpty()) // 缺少 access_key 和 secret_key

	fullConfig := UnifiedConfig{
		Bucket:    "test",
		AccessKey: "key",
		SecretKey: "secret",
	}
	assert.False(t, fullConfig.IsEmpty())
}

// TestNormalizeRegion 测试地域名称规范化
func TestNormalizeRegion(t *testing.T) {
	assert.Equal(t, "ap-beijing", NormalizeRegion(ProviderCOS, "beijing"))
	assert.Equal(t, "ap-shanghai", NormalizeRegion(ProviderCOS, "shanghai"))
	assert.Equal(t, "ap-guangzhou", NormalizeRegion(ProviderCOS, "guangzhou"))

	assert.Equal(t, "cn-hangzhou", NormalizeRegion(ProviderOSS, "hangzhou"))
	assert.Equal(t, "cn-shanghai", NormalizeRegion(ProviderOSS, "shanghai"))
	assert.Equal(t, "cn-beijing", NormalizeRegion(ProviderOSS, "beijing"))

	assert.Equal(t, "bj", NormalizeRegion(ProviderBOS, "beijing"))
	assert.Equal(t, "gz", NormalizeRegion(ProviderBOS, "guangzhou"))
}

// MockProvider 是一个用于测试的 mock 提供商
type MockProvider struct {
	name     string
	uploaded map[string][]byte
	deleted  []string
}

func NewMockProvider(name string) *MockProvider {
	return &MockProvider{
		name:     name,
		uploaded: make(map[string][]byte),
		deleted:  []string{},
	}
}

func (m *MockProvider) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*UploadResult, error) {
	data, _ := io.ReadAll(reader)
	m.uploaded[key] = data
	return &UploadResult{
		URL:         "https://example.com/" + key,
		Key:         key,
		Size:        size,
		ContentType: contentType,
	}, nil
}

func (m *MockProvider) UploadBytes(ctx context.Context, key string, data []byte, contentType string) (*UploadResult, error) {
	m.uploaded[key] = data
	return &UploadResult{
		URL:         "https://example.com/" + key,
		Key:         key,
		Size:        int64(len(data)),
		ContentType: contentType,
	}, nil
}

func (m *MockProvider) Delete(ctx context.Context, key string) error {
	m.deleted = append(m.deleted, key)
	delete(m.uploaded, key)
	return nil
}

func (m *MockProvider) GetURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	return "https://example.com/" + key + "?expires=" + expires.String(), nil
}

func (m *MockProvider) Exists(ctx context.Context, key string) (bool, error) {
	_, ok := m.uploaded[key]
	return ok, nil
}

func (m *MockProvider) GetProviderName() string {
	return m.name
}

// TestStorage 测试存储管理器
func TestStorage(t *testing.T) {
	// 创建测试配置
	config := &Config{
		DefaultProvider: ProviderCOS,
		UseHTTPS:        true,
	}

	storage, err := NewStorage(config)
	assert.NoError(t, err)
	assert.NotNil(t, storage)

	// 注册 mock 提供商
	mockProvider := NewMockProvider("mock")
	storage.RegisterProvider(ProviderCOS, mockProvider)

	// 测试上传
	ctx := context.Background()
	data := []byte("test data")
	result, err := storage.UploadBytes(ctx, "test.txt", data, "text/plain")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "test.txt", result.Key)

	// 测试存在性检查
	exists, err := storage.Exists(ctx, "test.txt")
	assert.NoError(t, err)
	assert.True(t, exists)

	// 测试删除
	err = storage.Delete(ctx, "test.txt")
	assert.NoError(t, err)

	// 验证删除
	exists, err = storage.Exists(ctx, "test.txt")
	assert.NoError(t, err)
	assert.False(t, exists)
}

// TestGlobalStorage 测试全局存储
func TestGlobalStorage(t *testing.T) {
	// 初始化全局存储
	config := &Config{
		DefaultProvider: ProviderCOS,
		UseHTTPS:        true,
	}

	err := Init(config)
	assert.NoError(t, err)

	// 验证全局存储已设置
	s := GetStorage()
	assert.NotNil(t, s)
}

// TestCOSConfigValidation 测试 COS 配置验证
func TestCOSConfigValidation(t *testing.T) {
	validConfig := COSConfig{
		Bucket:    "test-bucket",
		Region:    "ap-guangzhou",
		SecretID:  "test-id",
		SecretKey: "test-key",
	}
	assert.NoError(t, validConfig.Validate())

	invalidConfig := COSConfig{
		Bucket: "test-bucket",
	}
	assert.Error(t, invalidConfig.Validate())
	assert.Contains(t, invalidConfig.Validate().Error(), "region")
}

// TestOSSConfigValidation 测试 OSS 配置验证
func TestOSSConfigValidation(t *testing.T) {
	validConfig := OSSConfig{
		Bucket:          "test-bucket",
		Endpoint:        "oss-cn-hangzhou.aliyuncs.com",
		AccessKeyID:     "test-id",
		AccessKeySecret: "test-secret",
	}
	assert.NoError(t, validConfig.Validate())

	invalidConfig := OSSConfig{
		Bucket: "test-bucket",
	}
	assert.Error(t, invalidConfig.Validate())
	assert.Contains(t, invalidConfig.Validate().Error(), "endpoint")
}
