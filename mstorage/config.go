package mstorage

import (
	"fmt"
	"strings"
)

// Config 存储配置
type Config struct {
	// 默认使用的提供商: cos, oss, bos, s3, ufile
	DefaultProvider ProviderType `mapstructure:"default_provider"`
	// 是否使用 HTTPS
	UseHTTPS bool `mapstructure:"use_https"`
	// 是否使用自定义域名
	UseCustomDomain bool `mapstructure:"use_custom_domain"`
	// 自定义域名
	CustomDomain string `mapstructure:"custom_domain"`
	// 各提供商配置
	Providers UnifiedProviderConfigs `mapstructure:"providers"`
}

// UnifiedProviderConfigs 统一的提供商配置
type UnifiedProviderConfigs struct {
	COS   UnifiedConfig `mapstructure:"cos"`
	OSS   UnifiedConfig `mapstructure:"oss"`
	BOS   UnifiedConfig `mapstructure:"bos"`
	S3    UnifiedConfig `mapstructure:"s3"`
	UFile UnifiedConfig `mapstructure:"ufile"`
}

// UnifiedConfig 统一的云存储配置
type UnifiedConfig struct {
	// 存储桶名称
	Bucket string `mapstructure:"bucket"`
	// Access Key / Access Key ID / Secret ID / Public Key
	AccessKey string `mapstructure:"access_key"`
	// Secret Key / Access Key Secret / Secret Access Key / Private Key
	SecretKey string `mapstructure:"secret_key"`
	// 所属地域，如 ap-guangzhou, cn-hangzhou, us-east-1
	// 部分服务商会根据 region 自动推导 endpoint
	Region string `mapstructure:"region"`
	// 自定义 Endpoint，如果不设置则根据 Region 自动推导
	// 优先级: Endpoint > 自动推导
	Endpoint string `mapstructure:"endpoint"`
	// 是否使用临时凭证
	UseToken bool `mapstructure:"use_token"`
	// 临时 Token / Session Token
	Token string `mapstructure:"token"`

	// ========== S3 特有配置 ==========
	// 是否使用路径样式（用于 MinIO 等兼容 S3 的服务）
	UsePathStyle bool `mapstructure:"use_path_style"`

	// ========== UFile 特有配置 ==========
	// 文件域名，如 example.ufileos.com
	// 如果不设置则使用默认格式: {bucket}.ufile.ucloud.cn
	FileHost string `mapstructure:"file_host"`
}

// ToCOSConfig 转换为腾讯云 COS 配置
func (c UnifiedConfig) ToCOSConfig() COSConfig {
	return COSConfig{
		Bucket:    c.Bucket,
		Region:    c.Region,
		SecretID:  c.AccessKey,
		SecretKey: c.SecretKey,
		UseToken:  c.UseToken,
		Token:     c.Token,
	}
}

// ToOSSConfig 转换为阿里云 OSS 配置
func (c UnifiedConfig) ToOSSConfig() OSSConfig {
	endpoint := c.Endpoint
	if endpoint == "" && c.Region != "" {
		// 自动推导 OSS endpoint
		endpoint = fmt.Sprintf("oss-%s.aliyuncs.com", c.Region)
	}

	return OSSConfig{
		Bucket:          c.Bucket,
		Endpoint:        endpoint,
		AccessKeyID:     c.AccessKey,
		AccessKeySecret: c.SecretKey,
		UseToken:        c.UseToken,
		Token:           c.Token,
	}
}

// ToBOSConfig 转换为百度云 BOS 配置
func (c UnifiedConfig) ToBOSConfig() BOSConfig {
	endpoint := c.Endpoint
	if endpoint == "" && c.Region != "" {
		// 自动推导 BOS endpoint
		endpoint = fmt.Sprintf("%s.bcebos.com", c.Region)
	}

	return BOSConfig{
		Bucket:          c.Bucket,
		Endpoint:        endpoint,
		AccessKeyID:     c.AccessKey,
		SecretAccessKey: c.SecretKey,
		UseToken:        c.UseToken,
		Token:           c.Token,
	}
}

// ToS3Config 转换为 AWS S3 配置
func (c UnifiedConfig) ToS3Config() S3Config {
	return S3Config{
		Bucket:          c.Bucket,
		Region:          c.Region,
		AccessKeyID:     c.AccessKey,
		SecretAccessKey: c.SecretKey,
		UseToken:        c.UseToken,
		Token:           c.Token,
		Endpoint:        c.Endpoint,
		UsePathStyle:    c.UsePathStyle,
	}
}

// ToUFileConfig 转换为 UCloud UFile 配置
func (c UnifiedConfig) ToUFileConfig() UFileConfig {
	fileHost := c.FileHost
	if fileHost == "" && c.Bucket != "" {
		// 使用默认格式
		fileHost = fmt.Sprintf("%s.ufile.ucloud.cn", c.Bucket)
	}

	return UFileConfig{
		Bucket:     c.Bucket,
		Region:     c.Region,
		PublicKey:  c.AccessKey,
		PrivateKey: c.SecretKey,
		FileHost:   fileHost,
		UseToken:   c.UseToken,
		Token:      c.Token,
	}
}

// IsEmpty 检查配置是否为空
func (c UnifiedConfig) IsEmpty() bool {
	return c.Bucket == "" || c.AccessKey == "" || c.SecretKey == ""
}

// Validate 验证配置
func (c UnifiedConfig) Validate() error {
	if c.Bucket == "" {
		return fmt.Errorf("bucket is required")
	}
	if c.AccessKey == "" {
		return fmt.Errorf("access_key is required")
	}
	if c.SecretKey == "" {
		return fmt.Errorf("secret_key is required")
	}

	// 检查是否需要 region 或 endpoint
	// COS, S3, UFile 需要 Region
	// OSS, BOS 可以通过 Region 推导 Endpoint，也可以直接设置 Endpoint

	return nil
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		DefaultProvider: ProviderCOS,
		UseHTTPS:        true,
		UseCustomDomain: false,
		Providers: UnifiedProviderConfigs{
			COS:   UnifiedConfig{},
			OSS:   UnifiedConfig{},
			BOS:   UnifiedConfig{},
			S3:    UnifiedConfig{},
			UFile: UnifiedConfig{},
		},
	}
}

// GetProviderConfig 获取指定提供商的配置
func (c *Config) GetProviderConfig(pt ProviderType) (UnifiedConfig, error) {
	switch pt {
	case ProviderCOS:
		return c.Providers.COS, nil
	case ProviderOSS:
		return c.Providers.OSS, nil
	case ProviderBOS:
		return c.Providers.BOS, nil
	case ProviderS3:
		return c.Providers.S3, nil
	case ProviderUFile:
		return c.Providers.UFile, nil
	default:
		return UnifiedConfig{}, fmt.Errorf("unknown provider type: %s", pt)
	}
}

// RegionEndpoints 常用地域对应的 Endpoint 映射（用于文档和提示）
var RegionEndpoints = map[ProviderType]map[string]string{
	ProviderCOS: {
		"ap-beijing":       "cos.ap-beijing.myqcloud.com",
		"ap-guangzhou":     "cos.ap-guangzhou.myqcloud.com",
		"ap-shanghai":      "cos.ap-shanghai.myqcloud.com",
		"ap-singapore":     "cos.ap-singapore.myqcloud.com",
		"ap-hongkong":      "cos.ap-hongkong.myqcloud.com",
		"na-siliconvalley": "cos.na-siliconvalley.myqcloud.com",
		"na-toronto":       "cos.na-toronto.myqcloud.com",
	},
	ProviderOSS: {
		"cn-hangzhou":    "oss-cn-hangzhou.aliyuncs.com",
		"cn-shanghai":    "oss-cn-shanghai.aliyuncs.com",
		"cn-beijing":     "oss-cn-beijing.aliyuncs.com",
		"cn-shenzhen":    "oss-cn-shenzhen.aliyuncs.com",
		"cn-hongkong":    "oss-cn-hongkong.aliyuncs.com",
		"ap-southeast-1": "oss-ap-southeast-1.aliyuncs.com",
		"us-west-1":      "oss-us-west-1.aliyuncs.com",
		"us-east-1":      "oss-us-east-1.aliyuncs.com",
	},
	ProviderBOS: {
		"bj":  "bj.bcebos.com",
		"gz":  "gz.bcebos.com",
		"su":  "su.bcebos.com",
		"bd":  "bd.bcebos.com",
		"hkg": "hkg.bcebos.com",
		"fwh": "fwh.bcebos.com",
		"sin": "sin.bcebos.com",
	},
}

// GetCommonRegions 获取常用地域列表
func GetCommonRegions(pt ProviderType) []string {
	if endpoints, ok := RegionEndpoints[pt]; ok {
		regions := make([]string, 0, len(endpoints))
		for region := range endpoints {
			regions = append(regions, region)
		}
		return regions
	}
	return nil
}

// AutoCompleteEndpoint 根据 region 自动补全 endpoint
func AutoCompleteEndpoint(pt ProviderType, region string) string {
	if endpoints, ok := RegionEndpoints[pt]; ok {
		if endpoint, ok := endpoints[region]; ok {
			return endpoint
		}
	}
	return ""
}

// NormalizeRegion 规范化地域名称
func NormalizeRegion(provider ProviderType, region string) string {
	region = strings.ToLower(strings.TrimSpace(region))

	// 根据提供商进行特定规范化
	switch provider {
	case ProviderCOS:
		// COS: 确保以 ap- 或 na- 等前缀开头
		if !strings.HasPrefix(region, "ap-") &&
			!strings.HasPrefix(region, "na-") &&
			!strings.HasPrefix(region, "eu-") &&
			!strings.HasPrefix(region, "sa-") {
			// 尝试添加前缀
			if region == "beijing" {
				return "ap-beijing"
			}
			if region == "shanghai" {
				return "ap-shanghai"
			}
			if region == "guangzhou" {
				return "ap-guangzhou"
			}
		}

	case ProviderOSS:
		// OSS: 确保以 cn- 或 ap- 等前缀开头
		if !strings.HasPrefix(region, "cn-") &&
			!strings.HasPrefix(region, "ap-") &&
			!strings.HasPrefix(region, "us-") &&
			!strings.HasPrefix(region, "eu-") {
			// 尝试添加前缀
			if region == "hangzhou" {
				return "cn-hangzhou"
			}
			if region == "shanghai" {
				return "cn-shanghai"
			}
			if region == "beijing" {
				return "cn-beijing"
			}
			if region == "shenzhen" {
				return "cn-shenzhen"
			}
		}

	case ProviderBOS:
		// BOS: 通常使用简称如 bj, gz
		if region == "beijing" {
			return "bj"
		}
		if region == "guangzhou" {
			return "gz"
		}
		if region == "suzhou" {
			return "su"
		}
	}

	return region
}
