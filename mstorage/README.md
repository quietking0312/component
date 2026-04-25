# mstorage - 多云存储组件

支持腾讯云 COS、阿里云 OSS、百度云 BOS、AWS S3、UCloud UFile 的统一文件存储组件。

## 特性

- 🔌 **统一接口**：所有云存储使用相同的 API
- 🌐 **多提供商支持**：COS、OSS、BOS、S3、UFile
- ⚙️ **统一配置**：所有提供商使用相同的配置字段名
- 🔄 **自动推导**：根据 Region 自动推导 Endpoint
- 📦 **便捷工具**：提供文件名生成、MIME 类型检测等工具函数

## 统一配置格式

所有云存储提供商使用统一的配置字段名：

| 字段 | 说明 | 示例 |
|------|------|------|
| `bucket` | 存储桶名称 | `mybucket` |
| `access_key` | 访问密钥 ID | 各服务商的 AK/SecretID/PublicKey |
| `secret_key` | 访问密钥 Secret | 各服务商的 SK/SecretKey/PrivateKey |
| `region` | 所属地域 | `ap-guangzhou`, `cn-hangzhou` |
| `endpoint` | 自定义 Endpoint（可选） | 优先级高于自动推导 |
| `use_token` | 是否使用临时凭证 | `false` |
| `token` | 临时 Token（可选） | STS 临时凭证 |

## 安装依赖

```bash
cd admin_server
go mod tidy
```

## 快速开始

### 1. 配置

在配置文件中添加存储配置（config.yaml）：

```yaml
# 存储配置
storage:
  # 默认使用的提供商: cos, oss, bos, s3, ufile
  default_provider: cos
  # 是否使用 HTTPS
  use_https: true
  
  providers:
    # 腾讯云 COS 配置
    cos:
      bucket: "mybucket-1234567890"
      access_key: "your-secret-id"
      secret_key: "your-secret-key"
      region: "ap-guangzhou"
    
    # 阿里云 OSS 配置
    oss:
      bucket: "mybucket"
      access_key: "your-access-key-id"
      secret_key: "your-access-key-secret"
      region: "cn-hangzhou"
      # 可选：自定义 endpoint（如果不设置则根据 region 自动推导）
      # endpoint: "oss-cn-hangzhou-internal.aliyuncs.com"
    
    # 百度云 BOS 配置
    bos:
      bucket: "mybucket"
      access_key: "your-access-key-id"
      secret_key: "your-secret-access-key"
      region: "bj"  # 会自动推导为 bj.bcebos.com
    
    # AWS S3 配置
    s3:
      bucket: "mybucket"
      access_key: "your-access-key-id"
      secret_key: "your-secret-access-key"
      region: "us-east-1"
      # 可选：自定义 endpoint（用于 MinIO 等兼容 S3 的服务）
      # endpoint: "http://localhost:9000"
      # use_path_style: true  # MinIO 需要设置为 true
    
    # UCloud UFile 配置
    ufile:
      bucket: "mybucket"
      access_key: "your-public-key"
      secret_key: "your-private-key"
      region: "cn-bj"
      # 可选：自定义文件域名（如果不设置则自动生成）
      # file_host: "mybucket.ufile.ucloud.cn"
```

### 2. 初始化

```go
import "admin_server/component/mstorage"

// 方式1: 从配置文件加载
config := &mstorage.Config{
    DefaultProvider: mstorage.ProviderCOS,
    UseHTTPS:        true,
}

// 从 viper 加载配置
if err := mconf.UnmarshalKey("storage", config); err != nil {
    log.Fatal(err)
}

if err := mstorage.Init(config); err != nil {
    log.Fatal(err)
}

// 方式2: 直接创建配置
config := &mstorage.Config{
    DefaultProvider: mstorage.ProviderCOS,
    UseHTTPS:        true,
    Providers: mstorage.UnifiedProviderConfigs{
        COS: mstorage.UnifiedConfig{
            Bucket:    "mybucket-1234567890",
            AccessKey: "your-secret-id",
            SecretKey: "your-secret-key",
            Region:    "ap-guangzhou",
        },
        OSS: mstorage.UnifiedConfig{
            Bucket:    "mybucket",
            AccessKey: "your-access-key-id",
            SecretKey: "your-access-key-secret",
            Region:    "cn-hangzhou",
        },
    },
}

if err := mstorage.Init(config); err != nil {
    log.Fatal(err)
}
```

### 3. 使用

```go
ctx := context.Background()

// 上传文件
file, _ := os.Open("image.jpg")
defer file.Close()

key := mstorage.GenerateKey("uploads/2024", "image.jpg", true)
result, err := mstorage.Upload(ctx, key, file, fileSize, "image/jpeg")
if err != nil {
    log.Fatal(err)
}
fmt.Println(result.URL)

// 上传字节数组
data := []byte("Hello, World!")
key = mstorage.GenerateUniqueKey("texts", "txt")
result, err = mstorage.UploadBytes(ctx, key, data, "text/plain")

// 获取文件 URL
url, err := mstorage.GetURL(ctx, key, 15*time.Minute)

// 检查文件是否存在
exists, err := mstorage.Exists(ctx, key)

// 删除文件
err = mstorage.Delete(ctx, key)
```

## 高级用法

### 多提供商切换

```go
storage := mstorage.GetStorage()

// 获取特定提供商
ossProvider, err := storage.GetProvider(mstorage.ProviderOSS)
if err != nil {
    log.Fatal(err)
}

// 使用指定提供商上传
result, err := ossProvider.Upload(ctx, key, reader, size, contentType)

// 切换默认提供商
storage.SetDefaultProvider(mstorage.ProviderOSS)
```

### 备份到多个存储

```go
storage := mstorage.GetStorage()
data := []byte("important data")
key := "backup/data.txt"

// 同时上传到 COS 和 OSS
cosResult, _ := storage.UploadWithProvider(ctx, mstorage.ProviderCOS, key, reader1, size, "text/plain")
ossResult, _ := storage.UploadWithProvider(ctx, mstorage.ProviderOSS, key, reader2, size, "text/plain")

fmt.Printf("COS: %s, OSS: %s\n", cosResult.URL, ossResult.URL)
```

### 使用 MinIO（兼容 S3）

```go
config := &mstorage.Config{
    DefaultProvider: mstorage.ProviderS3,
    Providers: mstorage.UnifiedProviderConfigs{
        S3: mstorage.UnifiedConfig{
            Bucket:       "mybucket",
            AccessKey:    "minioadmin",
            SecretKey:    "minioadmin",
            Region:       "us-east-1",
            Endpoint:     "http://localhost:9000",
            UsePathStyle: true, // MinIO 需要路径样式
        },
    },
}
```

## 地域与 Endpoint 自动推导

对于 OSS 和 BOS，如果设置了 `region` 而没有设置 `endpoint`，组件会自动推导 endpoint：

| 提供商 | Region 示例 | 自动推导的 Endpoint |
|--------|-------------|---------------------|
| COS | `ap-guangzhou` | `cos.ap-guangzhou.myqcloud.com` |
| OSS | `cn-hangzhou` | `oss-cn-hangzhou.aliyuncs.com` |
| BOS | `bj` | `bj.bcebos.com` |

如果需要使用内网地址或自定义域名，可以直接设置 `endpoint` 字段。

## API 文档

### Provider 接口

```go
type Provider interface {
    Upload(ctx context.Context, key string, reader io.Reader, size int64, contentType string) (*UploadResult, error)
    UploadBytes(ctx context.Context, key string, data []byte, contentType string) (*UploadResult, error)
    Delete(ctx context.Context, key string) error
    GetURL(ctx context.Context, key string, expires time.Duration) (string, error)
    Exists(ctx context.Context, key string) (bool, error)
    GetProviderName() string
}
```

### 工具函数

- `GenerateKey(prefix, filename string, keepExt bool) string` - 生成存储键名
- `GenerateUniqueKey(prefix, ext string) string` - 生成唯一键名
- `GetContentType(filename string) string` - 获取 MIME 类型
- `IsImage(filename string) bool` - 判断是否为图片
- `IsVideo(filename string) bool` - 判断是否为视频
- `IsAudio(filename string) bool` - 判断是否为音频
- `FormatSize(size int64) string` - 格式化文件大小
- `NormalizeRegion(provider ProviderType, region string) string` - 规范化地域名称

## 测试

```bash
cd admin_server/component/mstorage
go test -v
```

## 注意事项

1. **密钥安全**：不要将密钥硬编码在代码中，使用环境变量或配置中心
2. **错误处理**：所有操作都可能返回错误，需要正确处理
3. **上下文控制**：上传大文件时，使用 `context.WithTimeout` 控制超时
4. **资源释放**：使用临时凭证时注意 Token 过期时间
5. **地域名称**：不同服务商的地域命名格式不同，使用 `NormalizeRegion` 函数可以规范化常用地域名称
