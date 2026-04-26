# memail - 邮件发送组件

基于 gomail 封装的邮件客户端，支持 SMTP 登录认证和连接池管理。

## 特性

- 📧 **SMTP 发送**：支持标准 SMTP 协议发送邮件
- 🔐 **LOGIN 认证**：支持 LOGIN 认证机制
- 🔄 **连接池**：支持连接缓存，减少重复拨号开销
- 🚀 **异步发送**：支持后台协程批量发送

## 快速开始

### 创建邮件客户端

```go
package main

import (
    "github.com/quietking0312/component/memail"
    "gopkg.in/gomail.v2"
)

func main() {
    client := memail.NewEmailClient(
        "smtp.example.com", // SMTP 服务器地址
        587,                // 端口
        "user@example.com", // 用户名
        "password",         // 密码
        10,                 // 连接池大小
    )
    defer client.Close()
}
```

### 发送邮件

```go
msg := gomail.NewMessage()
msg.SetHeader("From", "sender@example.com")
msg.SetHeader("To", "recipient@example.com")
msg.SetHeader("Subject", "测试邮件")
msg.SetBody("text/plain", "这是一封测试邮件")

err := client.Send("sender@example.com", []string{"recipient@example.com"}, msg)
if err != nil {
    panic(err)
}
```

### 异步发送

```go
client.GoRun() // 启动后台发送协程
// 邮件会被加入队列异步发送
```

## API 文档

### 类型

- `type EmailClient struct` - 邮件客户端

### 函数

- `NewEmailClient(host string, port int, username string, password string, cacheNumber int) *EmailClient` - 创建客户端

### 方法

- `(e *EmailClient) Dial() (gomail.SendCloser, error)` - 建立 SMTP 连接
- `(e *EmailClient) Send(from string, to []string, msg io.WriterTo) error` - 发送邮件
- `(e *EmailClient) Close() error` - 关闭客户端
- `(e *EmailClient) GoRun()` - 启动异步发送协程

## 注意事项

1. **端口选择**：常用 SMTP 端口为 25（非加密）、587（STARTTLS）、465（SSL/TLS）
2. **连接池**：根据发送频率设置合适的连接池大小
3. **异步模式**：异步发送不保证实时送达，适合批量通知场景

## 测试

```bash
cd memail
go test -v
```
