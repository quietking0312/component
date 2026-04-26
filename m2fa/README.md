# m2fa - 双因素认证组件

基于 TOTP（Time-based One-Time Password）算法的双因素认证工具，兼容 Google Authenticator 等主流验证器。

## 特性

- 🔐 **TOTP 标准实现**：兼容 RFC 6238
- 📱 **主流验证器支持**：Google Authenticator、Microsoft Authenticator 等
- 🚀 **简单易用**：几行代码即可实现 2FA 功能

## 依赖

```go
import "github.com/pquerna/otp/totp"
```

## 快速开始

### 生成密钥

```go
package main

import (
    "fmt"
    "github.com/pquerna/otp/totp"
    "time"
)

func main() {
    key, err := totp.Generate(totp.GenerateOpts{
        Issuer:      "MyApp",
        AccountName: "user@example.com",
    })
    if err != nil {
        panic(err)
    }

    // 将密钥展示给用户扫码绑定
    fmt.Println("Secret:", key.Secret())
    fmt.Println("URL:", key.URL())
}
```

### 生成验证码

```go
otp, err := totp.GenerateCode(secret, time.Now())
if err != nil {
    panic(err)
}
fmt.Println("OTP:", otp)
```

### 验证验证码

```go
valid := totp.Validate(otp, secret)
if valid {
    fmt.Println("验证通过")
} else {
    fmt.Println("验证失败")
}
```

## 注意事项

1. **密钥安全**：用户密钥需要安全存储，建议加密后存入数据库
2. **时间同步**：服务端与客户端时间需要保持同步（允许一定容差）
3. **备份码**：建议为用户提供一次性备份码，防止设备丢失无法登录

## 测试

```bash
cd m2fa
go test -v
```
