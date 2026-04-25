package mstorage

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"
)

// ExampleInit 初始化示例
func ExampleInit() {
	// 方式1: 使用统一的配置格式（推荐）
	// 所有云存储使用相同的字段名：bucket, access_key, secret_key, region
	config := &Config{
		DefaultProvider: ProviderCOS,
		UseHTTPS:        true,
		Providers: UnifiedProviderConfigs{
			// 腾讯云 COS
			COS: UnifiedConfig{
				Bucket:    "mybucket-1234567890",
				AccessKey: "your-secret-id",
				SecretKey: "your-secret-key",
				Region:    "ap-guangzhou",
			},
			// 阿里云 OSS
			OSS: UnifiedConfig{
				Bucket:    "mybucket",
				AccessKey: "your-access-key-id",
				SecretKey: "your-access-key-secret",
				Region:    "cn-hangzhou", // 会自动推导 endpoint
			},
			// 百度云 BOS
			BOS: UnifiedConfig{
				Bucket:    "mybucket",
				AccessKey: "your-access-key-id",
				SecretKey: "your-secret-access-key",
				Region:    "bj", // 会自动推导 endpoint: bj.bcebos.com
			},
			// AWS S3
			S3: UnifiedConfig{
				Bucket:    "mybucket",
				AccessKey: "your-access-key-id",
				SecretKey: "your-secret-access-key",
				Region:    "us-east-1",
			},
			// UCloud UFile
			UFile: UnifiedConfig{
				Bucket:    "mybucket",
				AccessKey: "your-public-key",
				SecretKey: "your-private-key",
				Region:    "cn-bj",
				FileHost:  "mybucket.ufile.ucloud.cn", // 可选，不设置则自动生成
			},
		},
	}

	if err := Init(config); err != nil {
		fmt.Printf("Init failed: %v\n", err)
		return
	}

	fmt.Println("Storage initialized successfully")
}

// ExampleInit_withEndpoint 使用自定义 Endpoint 初始化示例
func ExampleInit_withEndpoint() {
	// 对于 OSS 和 BOS，可以直接指定 endpoint，也可以只指定 region 自动推导
	config := &Config{
		DefaultProvider: ProviderOSS,
		Providers: UnifiedProviderConfigs{
			OSS: UnifiedConfig{
				Bucket:    "mybucket",
				AccessKey: "your-access-key-id",
				SecretKey: "your-access-key-secret",
				// 方式1: 直接指定 endpoint
				Endpoint: "oss-cn-hangzhou-internal.aliyuncs.com",
				// 方式2: 只指定 region，endpoint 会自动推导为 oss-cn-hangzhou.aliyuncs.com
				// Region: "cn-hangzhou",
			},
		},
	}

	if err := Init(config); err != nil {
		fmt.Printf("Init failed: %v\n", err)
		return
	}

	fmt.Println("Storage initialized with custom endpoint")
}

// ExampleInit_minio 使用 MinIO 初始化示例（兼容 S3）
func ExampleInit_minio() {
	config := &Config{
		DefaultProvider: ProviderS3,
		Providers: UnifiedProviderConfigs{
			S3: UnifiedConfig{
				Bucket:       "mybucket",
				AccessKey:    "minioadmin",
				SecretKey:    "minioadmin",
				Region:       "us-east-1",
				Endpoint:     "http://localhost:9000",
				UsePathStyle: true, // MinIO 需要路径样式
			},
		},
	}

	if err := Init(config); err != nil {
		fmt.Printf("Init failed: %v\n", err)
		return
	}

	fmt.Println("MinIO storage initialized")
}

// ExampleUpload 上传文件示例
func ExampleUpload() {
	ctx := context.Background()

	// 从文件上传
	file, err := os.Open("/path/to/file.jpg")
	if err != nil {
		fmt.Printf("Open file failed: %v\n", err)
		return
	}
	defer file.Close()

	// 获取文件信息
	fileInfo, _ := file.Stat()

	// 生成存储键名
	key := GenerateKey("uploads/2024", "file.jpg", true)

	// 上传文件（使用默认提供商）
	result, err := Upload(ctx, key, file, fileInfo.Size(), "image/jpeg")
	if err != nil {
		fmt.Printf("Upload failed: %v\n", err)
		return
	}

	fmt.Printf("Upload success: URL=%s, Key=%s\n", result.URL, result.Key)
}

// ExampleUploadBytes 上传字节数组示例
func ExampleUploadBytes() {
	ctx := context.Background()

	// 从字节数组上传
	data := []byte("Hello, World!")
	key := GenerateUniqueKey("texts", "txt")

	result, err := UploadBytes(ctx, key, data, "text/plain")
	if err != nil {
		fmt.Printf("Upload failed: %v\n", err)
		return
	}

	fmt.Printf("Upload success: URL=%s\n", result.URL)
}

// ExampleStorage_UploadWithProvider 使用指定提供商上传示例
func ExampleStorage_UploadWithProvider() {
	ctx := context.Background()
	storage := GetStorage()

	data := []byte("Hello, World!")
	key := GenerateUniqueKey("data", "txt")

	// 使用阿里云 OSS 上传
	result, err := storage.UploadWithProvider(ctx, ProviderOSS, key, bytes.NewReader(data), int64(len(data)), "text/plain")
	if err != nil {
		fmt.Printf("Upload failed: %v\n", err)
		return
	}

	fmt.Printf("Upload with OSS success: URL=%s\n", result.URL)
}

// ExampleGetURL 获取文件访问 URL 示例
func ExampleGetURL() {
	ctx := context.Background()

	// 获取文件 URL（公有读桶直接返回）
	url, err := GetURL(ctx, "uploads/2024/file.jpg", 0)
	if err != nil {
		fmt.Printf("Get URL failed: %v\n", err)
		return
	}

	fmt.Printf("File URL: %s\n", url)

	// 获取私有桶的临时访问 URL（15分钟有效）
	presignedURL, err := GetURL(ctx, "private/file.jpg", 15*time.Minute)
	if err != nil {
		fmt.Printf("Get presigned URL failed: %v\n", err)
		return
	}

	fmt.Printf("Presigned URL: %s\n", presignedURL)
}

// ExampleDelete 删除文件示例
func ExampleDelete() {
	ctx := context.Background()

	err := Delete(ctx, "uploads/2024/old-file.jpg")
	if err != nil {
		fmt.Printf("Delete failed: %v\n", err)
		return
	}

	fmt.Println("Delete success")
}

// ExampleExists 检查文件是否存在示例
func ExampleExists() {
	ctx := context.Background()

	exists, err := Exists(ctx, "uploads/2024/file.jpg")
	if err != nil {
		fmt.Printf("Check exists failed: %v\n", err)
		return
	}

	if exists {
		fmt.Println("File exists")
	} else {
		fmt.Println("File not found")
	}
}

// ExampleStorage_multipleProviders 多提供商使用示例
func ExampleStorage_multipleProviders() {
	// 统一配置，使用相同的字段名
	config := &Config{
		DefaultProvider: ProviderCOS,
		Providers: UnifiedProviderConfigs{
			COS: UnifiedConfig{
				Bucket:    "primary-bucket",
				AccessKey: "cos-access-key",
				SecretKey: "cos-secret-key",
				Region:    "ap-beijing",
			},
			OSS: UnifiedConfig{
				Bucket:    "backup-bucket",
				AccessKey: "oss-access-key",
				SecretKey: "oss-secret-key",
				Region:    "cn-hangzhou",
			},
		},
	}

	storage, err := NewStorage(config)
	if err != nil {
		fmt.Printf("Create storage failed: %v\n", err)
		return
	}

	ctx := context.Background()
	data := []byte("Important backup data")

	// 上传到主存储（COS）
	cosResult, err := storage.UploadWithProvider(ctx, ProviderCOS, "backup/data.txt", bytes.NewReader(data), int64(len(data)), "text/plain")
	if err != nil {
		fmt.Printf("Upload to COS failed: %v\n", err)
		return
	}
	fmt.Printf("COS: %s\n", cosResult.URL)

	// 同时备份到 OSS
	ossResult, err := storage.UploadWithProvider(ctx, ProviderOSS, "backup/data.txt", bytes.NewReader(data), int64(len(data)), "text/plain")
	if err != nil {
		fmt.Printf("Upload to OSS failed: %v\n", err)
		return
	}
	fmt.Printf("OSS: %s\n", ossResult.URL)
}

// Example_storageUsage 完整使用流程示例
func Example_storageUsage() {
	// 1. 初始化存储
	config := &Config{
		DefaultProvider: ProviderCOS,
		UseHTTPS:        true,
		Providers: UnifiedProviderConfigs{
			COS: UnifiedConfig{
				Bucket:    "my-app-bucket",
				AccessKey: "your-secret-id",
				SecretKey: "your-secret-key",
				Region:    "ap-beijing",
			},
		},
	}

	if err := Init(config); err != nil {
		fmt.Printf("Init failed: %v\n", err)
		return
	}

	ctx := context.Background()

	// 2. 上传图片
	imageData := []byte("fake image data") // 实际应该是图片文件内容
	imageKey := GenerateUniqueKey("images/2024/01", "jpg")

	uploadResult, err := UploadBytes(ctx, imageKey, imageData, "image/jpeg")
	if err != nil {
		fmt.Printf("Upload failed: %v\n", err)
		return
	}

	fmt.Printf("Image uploaded: %s\n", uploadResult.URL)

	// 3. 获取访问 URL
	url, err := GetURL(ctx, imageKey, 30*time.Minute)
	if err != nil {
		fmt.Printf("Get URL failed: %v\n", err)
		return
	}

	fmt.Printf("Access URL: %s\n", url)

	// 4. 检查文件是否存在
	exists, err := Exists(ctx, imageKey)
	if err != nil {
		fmt.Printf("Check exists failed: %v\n", err)
		return
	}

	if exists {
		fmt.Println("File confirmed to exist")
	}

	// 5. 删除文件（清理示例）
	if err := Delete(ctx, imageKey); err != nil {
		fmt.Printf("Delete failed: %v\n", err)
		return
	}

	fmt.Println("File deleted successfully")
}

// Example_configFromYAML YAML 配置示例
func Example_configFromYAML() {
	// 统一的 YAML 配置格式
	yamlConfig := `
storage:
  default_provider: cos
  use_https: true
  
  providers:
    cos:
      bucket: "mybucket-1234567890"
      access_key: "your-secret-id"
      secret_key: "your-secret-key"
      region: "ap-guangzhou"
    
    oss:
      bucket: "mybucket"
      access_key: "your-access-key-id"
      secret_key: "your-access-key-secret"
      region: "cn-hangzhou"
    
    bos:
      bucket: "mybucket"
      access_key: "your-access-key-id"
      secret_key: "your-secret-access-key"
      region: "bj"
    
    s3:
      bucket: "mybucket"
      access_key: "your-access-key-id"
      secret_key: "your-secret-access-key"
      region: "us-east-1"
    
    ufile:
      bucket: "mybucket"
      access_key: "your-public-key"
      secret_key: "your-private-key"
      region: "cn-bj"
`
	fmt.Println("YAML config format:")
	fmt.Println(yamlConfig)
}
