# mssh - SSH/SFTP 客户端组件

SSH 远程命令执行和 SFTP 文件传输工具，支持带进度条的文件上传下载。

## 特性

- 🔐 **SSH 连接**：支持密码和密钥认证
- 📁 **SFTP 传输**：上传下载文件
- 📊 **进度显示**：文件传输带实时进度条
- 🖥️ **远程执行**：在远程服务器执行命令并获取输出

## 快速开始

### SSH 连接并执行命令

```go
package main

import (
    "fmt"
    "github.com/quietking0312/component/mssh"
)

func main() {
    cli := &mssh.Cli{
        Host:     "192.168.1.100",
        Port:     22,
        User:     "root",
        Password: "password",
    }

    if err := cli.Connect(); err != nil {
        panic(err)
    }
    defer cli.Close()

    output, err := cli.Run("uname -a")
    if err != nil {
        panic(err)
    }
    fmt.Println(output)
}
```

### 上传文件

```go
err := cli.UploadFile("/local/path/file.txt", "/remote/path/file.txt")
if err != nil {
    panic(err)
}
```

### 带进度条上传

```go
ch := make(chan int64)
go func() {
    for progress := range ch {
        fmt.Printf("已上传: %d bytes\n", progress)
    }
}()

file, _ := os.Open("/local/bigfile.zip")
err := cli.UploadFileAndProgress(file, "/remote/bigfile.zip", ch)
```

### 下载文件

```go
err := cli.DownloadFile("/remote/path/file.txt", "/local/dir", "file.txt")
if err != nil {
    panic(err)
}
```

## API 文档

### 类型

```go
type Cli struct {
    Host       string
    Port       int
    User       string
    Password   string
    PrivateKey string // 私钥路径或内容
}
```

### 方法

| 方法 | 说明 |
|------|------|
| `(c *Cli) Connect() error` | 建立 SSH 连接 |
| `(c *Cli) Close() error` | 关闭连接 |
| `(c *Cli) Run(command string) (string, error)` | 执行远程命令 |
| `(c *Cli) UploadFile(localFilePath, remotePath string) error` | 上传文件 |
| `(c *Cli) UploadFileAndProgress(srcFile io.Reader, remoteFile string, ch chan<- int64) error` | 带进度上传 |
| `(c *Cli) DownloadFile(remotePath, localDir, localFileName string) error` | 下载文件 |

## 测试

```bash
cd mssh
go test -v
```

## 注意事项

1. 首次连接新服务器时，需要处理主机密钥确认
2. 大文件传输建议使用带进度条的方法，便于监控
3. 私钥认证时，请确保证书格式正确（PEM）
