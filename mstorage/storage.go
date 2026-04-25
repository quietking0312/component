package mstorage

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"
)

// Storage 统一存储管理器
type Storage struct {
	config          *Config
	providers       map[ProviderType]Provider
	defaultProvider ProviderType
	mu              sync.RWMutex
}

var (
	// 全局存储实例
	globalStorage *Storage
	// 全局实例锁
	globalMu sync.RWMutex
)

// Init 初始化全局存储管理器
func Init(config *Config) error {
	if config == nil {
		config = DefaultConfig()
	}

	storage, err := NewStorage(config)
	if err != nil {
		return err
	}

	globalMu.Lock()
	globalStorage = storage
	globalMu.Unlock()

	return nil
}

// NewStorage 创建存储管理器
func NewStorage(config *Config) (*Storage, error) {
	s := &Storage{
		config:    config,
		providers: make(map[ProviderType]Provider),
	}

	// 初始化各提供商
	if err := s.initProviders(); err != nil {
		return nil, err
	}

	// 设置默认提供商
	if config.DefaultProvider.IsValid() {
		s.defaultProvider = config.DefaultProvider
	} else {
		s.defaultProvider = ProviderCOS
	}

	return s, nil
}

// initProviders 初始化所有配置的提供商
func (s *Storage) initProviders() error {
	// 腾讯云 COS
	if !s.config.Providers.COS.IsEmpty() {
		cosConfig := s.config.Providers.COS.ToCOSConfig()
		if err := cosConfig.Validate(); err != nil {
			return fmt.Errorf("cos config invalid: %w", err)
		}
		provider, err := NewCOSProvider(cosConfig)
		if err != nil {
			return fmt.Errorf("init cos failed: %w", err)
		}
		s.providers[ProviderCOS] = provider
	}

	// 阿里云 OSS
	if !s.config.Providers.OSS.IsEmpty() {
		ossConfig := s.config.Providers.OSS.ToOSSConfig()
		if err := ossConfig.Validate(); err != nil {
			return fmt.Errorf("oss config invalid: %w", err)
		}
		provider, err := NewOSSProvider(ossConfig)
		if err != nil {
			return fmt.Errorf("init oss failed: %w", err)
		}
		s.providers[ProviderOSS] = provider
	}

	// 百度云 BOS
	if !s.config.Providers.BOS.IsEmpty() {
		bosConfig := s.config.Providers.BOS.ToBOSConfig()
		if err := bosConfig.Validate(); err != nil {
			return fmt.Errorf("bos config invalid: %w", err)
		}
		provider, err := NewBOSProvider(bosConfig)
		if err != nil {
			return fmt.Errorf("init bos failed: %w", err)
		}
		s.providers[ProviderBOS] = provider
	}

	// AWS S3
	if !s.config.Providers.S3.IsEmpty() {
		s3Config := s.config.Providers.S3.ToS3Config()
		if err := s3Config.Validate(); err != nil {
			return fmt.Errorf("s3 config invalid: %w", err)
		}
		provider, err := NewS3Provider(s3Config)
		if err != nil {
			return fmt.Errorf("init s3 failed: %w", err)
		}
		s.providers[ProviderS3] = provider
	}

	// UCloud UFile
	if !s.config.Providers.UFile.IsEmpty() {
		ufileConfig := s.config.Providers.UFile.ToUFileConfig()
		if err := ufileConfig.Validate(); err != nil {
			return fmt.Errorf("ufile config invalid: %w", err)
		}
		provider, err := NewUFileProvider(ufileConfig)
		if err != nil {
			return fmt.Errorf("init ufile failed: %w", err)
		}
		s.providers[ProviderUFile] = provider
	}

	return nil
}

// RegisterProvider 注册自定义提供商
func (s *Storage) RegisterProvider(pt ProviderType, provider Provider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.providers[pt] = provider
}

// GetProvider 获取指定类型的提供商
func (s *Storage) GetProvider(pt ProviderType) (Provider, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	provider, ok := s.providers[pt]
	if !ok {
		return nil, fmt.Errorf("provider %s not found or not initialized", pt)
	}

	return provider, nil
}

// GetDefaultProvider 获取默认提供商
func (s *Storage) GetDefaultProvider() (Provider, error) {
	return s.GetProvider(s.defaultProvider)
}

// SetDefaultProvider 设置默认提供商
func (s *Storage) SetDefaultProvider(pt ProviderType) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.defaultProvider = pt
}

// GetConfig 获取存储配置
func (s *Storage) GetConfig() *Config {
	return s.config
}

// GetProviderConfig 获取指定提供商的配置
func (s *Storage) GetProviderConfig(pt ProviderType) (UnifiedConfig, error) {
	return s.config.GetProviderConfig(pt)
}

// Upload 使用默认提供商上传文件
func (s *Storage) Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*UploadResult, error) {
	provider, err := s.GetDefaultProvider()
	if err != nil {
		return nil, err
	}
	return provider.Upload(ctx, key, reader, size, contentType)
}

// UploadWithProvider 使用指定提供商上传文件
func (s *Storage) UploadWithProvider(ctx context.Context, pt ProviderType, key string, reader io.Reader, size int64, contentType string) (*UploadResult, error) {
	provider, err := s.GetProvider(pt)
	if err != nil {
		return nil, err
	}
	return provider.Upload(ctx, key, reader, size, contentType)
}

// UploadBytes 使用默认提供商上传字节数组
func (s *Storage) UploadBytes(ctx context.Context, key string, data []byte, contentType string) (*UploadResult, error) {
	provider, err := s.GetDefaultProvider()
	if err != nil {
		return nil, err
	}
	return provider.UploadBytes(ctx, key, data, contentType)
}

// UploadBytesWithProvider 使用指定提供商上传字节数组
func (s *Storage) UploadBytesWithProvider(ctx context.Context, pt ProviderType, key string, data []byte, contentType string) (*UploadResult, error) {
	provider, err := s.GetProvider(pt)
	if err != nil {
		return nil, err
	}
	return provider.UploadBytes(ctx, key, data, contentType)
}

// Delete 使用默认提供商删除文件
func (s *Storage) Delete(ctx context.Context, key string) error {
	provider, err := s.GetDefaultProvider()
	if err != nil {
		return err
	}
	return provider.Delete(ctx, key)
}

// DeleteWithProvider 使用指定提供商删除文件
func (s *Storage) DeleteWithProvider(ctx context.Context, pt ProviderType, key string) error {
	provider, err := s.GetProvider(pt)
	if err != nil {
		return err
	}
	return provider.Delete(ctx, key)
}

// GetURL 使用默认提供商获取文件 URL
func (s *Storage) GetURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	provider, err := s.GetDefaultProvider()
	if err != nil {
		return "", err
	}
	return provider.GetURL(ctx, key, expires)
}

// GetURLWithProvider 使用指定提供商获取文件 URL
func (s *Storage) GetURLWithProvider(ctx context.Context, pt ProviderType, key string, expires time.Duration) (string, error) {
	provider, err := s.GetProvider(pt)
	if err != nil {
		return "", err
	}
	return provider.GetURL(ctx, key, expires)
}

// Exists 使用默认提供商检查文件是否存在
func (s *Storage) Exists(ctx context.Context, key string) (bool, error) {
	provider, err := s.GetDefaultProvider()
	if err != nil {
		return false, err
	}
	return provider.Exists(ctx, key)
}

// ExistsWithProvider 使用指定提供商检查文件是否存在
func (s *Storage) ExistsWithProvider(ctx context.Context, pt ProviderType, key string) (bool, error) {
	provider, err := s.GetProvider(pt)
	if err != nil {
		return false, err
	}
	return provider.Exists(ctx, key)
}

// GetProviderNames 获取所有已初始化的提供商名称
func (s *Storage) GetProviderNames() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.providers))
	for _, p := range s.providers {
		names = append(names, p.GetProviderName())
	}
	return names
}

// IsProviderAvailable 检查提供商是否可用
func (s *Storage) IsProviderAvailable(pt ProviderType) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.providers[pt]
	return ok
}

// ========== 全局便捷方法 ==========

// GetStorage 获取全局存储实例
func GetStorage() *Storage {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalStorage
}

// SetStorage 设置全局存储实例
func SetStorage(s *Storage) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalStorage = s
}

// Upload 使用全局存储上传文件
func Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*UploadResult, error) {
	s := GetStorage()
	if s == nil {
		return nil, fmt.Errorf("storage not initialized")
	}
	return s.Upload(ctx, key, reader, size, contentType)
}

// UploadBytes 使用全局存储上传字节数组
func UploadBytes(ctx context.Context, key string, data []byte, contentType string) (*UploadResult, error) {
	s := GetStorage()
	if s == nil {
		return nil, fmt.Errorf("storage not initialized")
	}
	return s.UploadBytes(ctx, key, data, contentType)
}

// Delete 使用全局存储删除文件
func Delete(ctx context.Context, key string) error {
	s := GetStorage()
	if s == nil {
		return fmt.Errorf("storage not initialized")
	}
	return s.Delete(ctx, key)
}

// GetURL 使用全局存储获取文件 URL
func GetURL(ctx context.Context, key string, expires time.Duration) (string, error) {
	s := GetStorage()
	if s == nil {
		return "", fmt.Errorf("storage not initialized")
	}
	return s.GetURL(ctx, key, expires)
}

// Exists 使用全局存储检查文件是否存在
func Exists(ctx context.Context, key string) (bool, error) {
	s := GetStorage()
	if s == nil {
		return false, fmt.Errorf("storage not initialized")
	}
	return s.Exists(ctx, key)
}
