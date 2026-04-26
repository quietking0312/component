# mjwt - JWT 工具组件

基于 `golang-jwt/jwt/v5` 封装的 JWT 生成与解析工具。

## 特性

- 🔏 **简洁 API**：封装了复杂的 JWT 操作，几行代码完成签发和验证
- ⚡ **多种算法**：支持 HS256、HS384、HS512 等对称签名算法
- 📦 **自定义 Claims**：支持任意实现 `jwt.Claims` 接口的结构体

## 快速开始

### 生成 Token

```go
package main

import (
    "fmt"
    "github.com/golang-jwt/jwt/v5"
    "github.com/quietking0312/component/mjwt"
    "time"
)

type MyClaims struct {
    UserID   int    `json:"user_id"`
    Username string `json:"username"`
    jwt.RegisteredClaims
}

func main() {
    key := []byte("your-secret-key")
    j := mjwt.NewJWT(key, jwt.SigningMethodHS256)

    claims := MyClaims{
        UserID:   1,
        Username: "alice",
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }

    j.SetData(&claims)
    tokenString, err := j.SignedString()
    if err != nil {
        panic(err)
    }
    fmt.Println(tokenString)
}
```

### 解析 Token

```go
key := []byte("your-secret-key")
j := mjwt.NewJWT(key, jwt.SigningMethodHS256)

var claims MyClaims
_, err := j.Parse(tokenString, &claims)
if err != nil {
    panic(err)
}

fmt.Println(claims.UserID, claims.Username)
```

## API 文档

| 函数/方法 | 说明 |
|-----------|------|
| `NewJWT(key []byte, method jwt.SigningMethod) *JWT` | 创建 JWT 实例 |
| `(j *JWT) SetData(data jwt.Claims)` | 设置 Claims 数据 |
| `(j *JWT) SignedString() (string, error)` | 生成 Token 字符串 |
| `(j *JWT) Parse(token string, data jwt.Claims) (*JWT, error)` | 解析 Token |

## 注意事项

1. **密钥安全**：请使用强随机密钥，并妥善保管
2. **过期时间**：建议设置合理的 Token 过期时间
3. **算法选择**：生产环境建议使用 RS256 等非对称算法（需自行扩展）

## 测试

```bash
cd mjwt
go test -v
```
