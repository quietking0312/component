# middleware - HTTP 中间件组件

提供 CORS 跨域处理和 Panic 恢复中间件，兼容 Gin 框架和标准 `net/http`。

## 特性

- 🌐 **CORS 跨域**：支持自定义允许来源、请求头、凭证等
- 🛡️ **Panic 恢复**：捕获 panic 并记录日志，防止服务崩溃
- 🔌 **双模式支持**：同时兼容 Gin 框架和标准 HTTP Handler

## 快速开始

### Gin CORS 中间件

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/quietking0312/component/middleware"
)

func main() {
    r := gin.Default()

    // 使用默认 CORS 配置
    r.Use(middleware.Cors(nil))

    // 或自定义配置
    corsCfg := middleware.NewCORSConfig(
        middleware.WithAllowHeaders("Authorization", "Content-Type"),
        middleware.WithExposeHeaders("X-Request-ID"),
        middleware.WithAllowCredentials(true),
    )
    r.Use(corsCfg.GinMiddleware())

    r.GET("/api/data", func(c *gin.Context) {
        c.JSON(200, gin.H{"data": "ok"})
    })

    r.Run(":8080")
}
```

### 标准 HTTP CORS

```go
mux := http.NewServeMux()
mux.HandleFunc("/", handler)

corsCfg := middleware.NewCORSConfig()
http.ListenAndServe(":8080", corsCfg.Handler(mux))
```

### Panic 恢复（Gin）

```go
r := gin.New()
r.Use(middleware.Recovery(func(c *gin.Context, err any) {
    // 自定义 panic 处理逻辑
    c.JSON(500, gin.H{"error": "internal server error"})
}))
```

### Panic 恢复（标准 HTTP）

```go
mux := http.NewServeMux()
mux.HandleFunc("/", handler)

http.ListenAndServe(":8080", middleware.Recovery(mux))
```

## API 文档

### CORS

| 函数 | 说明 |
|------|------|
| `NewCORSConfig(opts ...CORSOption) *CORSConfig` | 创建 CORS 配置 |
| `WithAllowHeaders(headers ...string) CORSOption` | 设置允许的请求头 |
| `WithExposeHeaders(headers ...string) CORSOption` | 设置暴露的响应头 |
| `WithAllowCredentials(allow bool) CORSOption` | 设置是否允许凭证 |
| `(cfg *CORSConfig) Handler(next http.Handler) http.Handler` | 标准 HTTP Handler |
| `(cfg *CORSConfig) GinMiddleware() gin.HandlerFunc` | Gin 中间件 |
| `Cors(w http.ResponseWriter, r *http.Request)` | 独立设置响应头 |

### Recovery

| 函数 | 说明 |
|------|------|
| `Recovery(recoveryFunc gin.RecoveryFunc) gin.HandlerFunc` | Gin panic 恢复 |
| `Recovery(next http.Handler) http.Handler` | 标准 HTTP panic 恢复 |

## 测试

```bash
cd middleware
go test -v
```
