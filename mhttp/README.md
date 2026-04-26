# mhttp - HTTP 客户端组件

基于标准库 `net/http` 封装的 HTTP 请求构建器，支持链式调用、JSON 自动绑定、超时配置等。

## 特性

- 🔗 **链式 API**：流畅的请求构建体验
- 📦 **JSON 自动绑定**：响应直接绑定到结构体
- ⏱️ **超时控制**：支持请求级和客户端级超时配置
- 🛡️ **错误处理**：统一的响应封装，便捷的成败判断
- 🔐 **Bearer Token**：内置 Token 设置快捷方式

## 快速开始

### 简单 GET 请求

```go
package main

import (
    "fmt"
    "github.com/quietking0312/component/mhttp"
)

func main() {
    resp, err := mhttp.Get("https://api.example.com/users").DoAndBindBytes()
    if err != nil {
        panic(err)
    }
    fmt.Println(string(resp))
}
```

### POST JSON 请求

```go
type User struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

var result User
err := mhttp.Post("https://api.example.com/users").
    SetContentType("application/json").
    BodyString(`{"name":"Alice","email":"alice@example.com"}`).
    DoAndBindJSON(&result)
```

### 使用自定义客户端

```go
client := mhttp.NewClient(
    mhttp.WithTimeout(30 * time.Second),
)

resp, err := client.Get("https://api.example.com/data").DoAndBindBytes()
```

### 查询参数

```go
resp, err := mhttp.Get("https://api.example.com/search").
    Query("q", "golang").
    Query("page", "1").
    DoAndBindBytes()
```

### Bearer Token

```go
resp, err := mhttp.Get("https://api.example.com/profile").
    SetBearerToken("your-jwt-token").
    DoAndBindBytes()
```

## API 文档

### 客户端

| 函数 | 说明 |
|------|------|
| `NewClient(opts ...Option) *Client` | 创建自定义客户端 |
| `WithTimeout(timeout time.Duration) Option` | 设置超时 |
| `WithHTTPClient(client *http.Client) Option` | 使用自定义 http.Client |
| `SetDefaultTimeout(timeout time.Duration)` | 设置默认客户端超时 |

### 快捷方法（默认客户端）

| 函数 | 说明 |
|------|------|
| `Get(url string) *Request` | GET 请求 |
| `Post(url string) *Request` | POST 请求 |
| `Put(url string) *Request` | PUT 请求 |
| `Delete(url string) *Request` | DELETE 请求 |
| `Patch(url string) *Request` | PATCH 请求 |
| `Head(url string) *Request` | HEAD 请求 |
| `Options(url string) *Request` | OPTIONS 请求 |

### 请求构建

| 方法 | 说明 |
|------|------|
| `(r *Request) SetContentType(contentType string) *Request` | 设置 Content-Type |
| `(r *Request) SetBearerToken(token string) *Request` | 设置 Bearer Token |
| `(r *Request) Query(key, value string) *Request` | 添加查询参数 |
| `(r *Request) Queries(queries map[string]string) *Request` | 批量添加查询参数 |
| `(r *Request) BodyString(body string) *Request` | 设置请求体 |
| `(r *Request) WithContext(ctx context.Context) *Request` | 设置 Context |
| `(r *Request) DoAndBindJSON(v interface{}) error` | 执行并绑定 JSON |
| `(r *Request) DoAndBindBytes() ([]byte, error)` | 执行并返回字节 |

### 响应

| 方法 | 说明 |
|------|------|
| `(r *Response) StatusCode() int` | 获取状态码 |
| `(r *Response) Body() []byte` | 获取响应体 |
| `(r *Response) String() string` | 获取响应体字符串 |
| `(r *Response) IsOK() bool` | 判断是否 2xx |
| `(r *Response) JSON(v interface{}) error` | 解析 JSON |

## 测试

```bash
cd mhttp
go test -v
```
