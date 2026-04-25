# mhash - 哈希计算组件

封装常用的哈希计算功能，包括 MD5、SHA1、SHA256、SHA512 以及 HMAC 系列。

## 特性

- 🔢 **多种算法**：支持 MD5、SHA1、SHA256、SHA512
- 🔐 **HMAC 支持**：支持 HMAC-MD5、HMAC-SHA256、HMAC-SHA512
- 📁 **多数据源**：支持字节数组、字符串、文件、io.Reader
- ⚡ **批量计算**：可同时计算多种哈希值
- 🔄 **流式计算**：支持分批次写入数据后计算

## 快速开始

### 计算 MD5

```go
import "admin_server/component/mhash"

// 计算字节数组
hash := mhash.MD5([]byte("Hello, World!"))
// 结果: 65a8e27d8879283831b664bd8b7f0ad4

// 计算字符串
hash = mhash.MD5String("Hello, World!")

// 计算文件
hash, err := mhash.MD5File("/path/to/file.txt")
if err != nil {
    log.Fatal(err)
}

// 计算 io.Reader
hash, err := mhash.MD5Reader(reader)
```

### 计算 SHA256

```go
// 计算字节数组
hash := mhash.SHA256([]byte("Hello, World!"))

// 计算字符串
hash = mhash.SHA256String("Hello, World!")

// 计算文件
hash, err := mhash.SHA256File("/path/to/file.txt")
```

### HMAC 计算

```go
// HMAC-MD5
hash := mhash.HMACMD5String("secret-key", "Hello, World!")

// HMAC-SHA256
hash = mhash.HMACSHA256String("secret-key", "Hello, World!")
```

## 高级用法

### 流式计算

适用于大文件或分批次接收数据的场景：

```go
hasher := mhash.NewMD5()

// 分多次写入
hasher.WriteString("Hello, ")
hasher.WriteString("World!")
hasher.Write([]byte(" More data."))

// 获取结果
hash := hasher.Sum()
```

### 批量计算

一次性计算多种哈希值，只需读取一次数据：

```go
// 从字节数组
data := []byte("Hello, World!")
result := mhash.CalculateAll(data)

fmt.Println(result.MD5)
fmt.Println(result.SHA1)
fmt.Println(result.SHA256)
fmt.Println(result.SHA512)

// 从文件
result, size, err := mhash.CalculateAllFile("/path/to/file.txt")
if err != nil {
    log.Fatal(err)
}

// 从 io.Reader
result, size, err := mhash.CalculateAllReader(reader)
```

### 获取数据大小和 MD5

```go
// 同时获取 MD5 和数据大小
hash, size, err := mhash.MD5ReaderWithSize(reader)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Size: %d, MD5: %s\n", size, hash)
```

## API 文档

### 基础哈希函数

| 函数 | 说明 |
|------|------|
| `MD5(data []byte) string` | 计算字节数组的 MD5 |
| `MD5String(s string) string` | 计算字符串的 MD5 |
| `MD5File(filepath string) (string, error)` | 计算文件的 MD5 |
| `MD5Reader(r io.Reader) (string, error)` | 计算 io.Reader 的 MD5 |
| `MD5ReaderWithSize(r io.Reader) (string, int64, error)` | 计算 MD5 并返回数据大小 |
| `SHA1(data []byte) string` | 计算 SHA1 |
| `SHA1String(s string) string` | 计算字符串的 SHA1 |
| `SHA1File(filepath string) (string, error)` | 计算文件的 SHA1 |
| `SHA256(data []byte) string` | 计算 SHA256 |
| `SHA256String(s string) string` | 计算字符串的 SHA256 |
| `SHA256File(filepath string) (string, error)` | 计算文件的 SHA256 |
| `SHA512(data []byte) string` | 计算 SHA512 |
| `SHA512String(s string) string` | 计算字符串的 SHA512 |
| `SHA512File(filepath string) (string, error)` | 计算文件的 SHA512 |

### HMAC 函数

| 函数 | 说明 |
|------|------|
| `HMACMD5(key, data []byte) string` | 计算 HMAC-MD5 |
| `HMACMD5String(key, s string) string` | 计算字符串的 HMAC-MD5 |
| `HMACSHA256(key, data []byte) string` | 计算 HMAC-SHA256 |
| `HMACSHA256String(key, s string) string` | 计算字符串的 HMAC-SHA256 |
| `HMACSHA512(key, data []byte) string` | 计算 HMAC-SHA512 |
| `HMACSHA512String(key, s string) string` | 计算字符串的 HMAC-SHA512 |

### Hasher 流式计算器

| 方法 | 说明 |
|------|------|
| `NewMD5() *Hasher` | 创建 MD5 计算器 |
| `NewSHA1() *Hasher` | 创建 SHA1 计算器 |
| `NewSHA256() *Hasher` | 创建 SHA256 计算器 |
| `NewSHA512() *Hasher` | 创建 SHA512 计算器 |
| `Write(data []byte) (int, error)` | 写入字节数据 |
| `WriteString(s string) (int, error)` | 写入字符串 |
| `Sum() string` | 获取十六进制哈希值 |
| `SumBytes() []byte` | 获取原始字节哈希值 |
| `Reset()` | 重置计算器 |

### 批量计算

| 函数 | 说明 |
|------|------|
| `CalculateAll(data []byte) *MultiHash` | 同时计算 MD5/SHA1/SHA256/SHA512 |
| `CalculateAllReader(r io.Reader) (*MultiHash, int64, error)` | 从 io.Reader 同时计算 |
| `CalculateAllFile(filepath string) (*MultiHash, int64, error)` | 从文件同时计算 |

**MultiHash 结构：**
```go
type MultiHash struct {
    MD5    string
    SHA1   string
    SHA256 string
    SHA512 string
}
```

## 测试

```bash
cd admin_server/component/mhash
go test -v

# 基准测试
go test -bench=.
```

## 注意事项

1. **线程安全**：基础函数是线程安全的，但 Hasher 不是（需要在并发使用时自行加锁）
2. **大文件处理**：对于大文件建议使用 `MD5Reader` 或 `Hasher` 流式计算，避免一次性加载到内存
3. **错误处理**：文件操作需要处理可能的 I/O 错误
