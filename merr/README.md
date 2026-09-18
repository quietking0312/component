# merr - 带错误码的错误处理组件

兼容标准 `error` 接口的带码错误体系，支持错误码、链式包装、`errors.Is/As`。

## 特性

- 🏷️ **错误码体系**：每个错误附带可读的错误码（如 `ERR_NOT_FOUND`）
- 🔗 **链式 Unwrap**：支持错误链追踪，兼容 `errors.Is/As`
- 🎯 **预定义错误**：提供常见的预定义错误（参数错误、禁止访问、超时等）
- 📋 **预设错误码消息**：可注册错误码对应的消息，兼容 proto 生成的数字枚举错误码
- 🛡️ **类型安全**：通过接口确保错误类型一致性

## 快速开始

### 创建带码错误

```go
package main

import (
    "fmt"
    "github.com/quietking0312/component/merr"
)

func main() {
    // 自定义错误
    err := merr.New("ERR_USER_NOT_FOUND", "用户 %d 不存在", 10086)
    fmt.Println(err.Error())      // 用户 10086 不存在
    fmt.Println(err.(*merr.MErr).Code()) // ERR_USER_NOT_FOUND
}
```

### 包装已有错误

```go
dbErr := database.Query(...)
err := merr.Wrap("ERR_DB_QUERY", dbErr)
```

### 包装并附加消息

```go
err := merr.Wrapf("ERR_DB_QUERY", dbErr, "查询订单失败: order_id=%s", orderID)
```

### 使用预定义错误

```go
err := merr.InvalidParam("参数 id 不能为空")
err := merr.Forbidden("没有权限访问该资源")
err := merr.Internal("数据库连接失败")
err := merr.Timeout("请求超时")
err := merr.Duplicate("用户已存在")
err := merr.TooManyRequests("请求过于频繁")
```

### 判断错误码

```go
if merr.IsCode(err, "ERR_USER_NOT_FOUND") {
    // 处理用户不存在的情况
}
```

### 注册预设错误码消息（兼容 proto 生成的数字枚举错误码）

`code` 参数通过泛型约束为 `string` 或 `int32`（含具名类型，如 proto 生成的枚举），传其他类型会在**编译期**报错；
数字枚举错误码会自动取其数值作为错误码：

```go
// 一般在 init 中注册一次
merr.RegisterCode(pb.ErrCode_USER_NOT_FOUND, "用户不存在")
merr.RegisterCode(pb.ErrCode_ORDER_EXPIRED, "订单已过期")
// 或批量注册
merr.RegisterCodes(map[string]string{
    "1001": "用户不存在",
    "1002": "订单已过期",
})

// 直接按码构造错误，消息取自预设
err := merr.Preset(pb.ErrCode_USER_NOT_FOUND)
fmt.Println(err.Error()) // [1001] 用户不存在

// New/Wrap/Wrapf 附加的信息会自动与预设消息拼接
err = merr.New(pb.ErrCode_USER_NOT_FOUND, "id=%d", 42)
fmt.Println(err.Error()) // [1001] 用户不存在: id=42

merr.IsCode(err, pb.ErrCode_USER_NOT_FOUND) // true
```

## API 文档

### 创建错误

| 函数 | 说明 |
|------|------|
| `New[T Code](code T, format string, a ...any) *MErr` | 创建新错误 |
| `Wrap[T Code](code T, err error) *MErr` | 包装已有错误 |
| `Wrapf[T Code](code T, err error, format string, a ...any) *MErr` | 包装错误并附加消息 |
| `Preset[T Code](code T) *MErr` | 使用预设消息按码创建错误，无需再传消息 |

其中 `Code` 约束为 `~string \| ~int32`：

```go
type Code interface {
    ~string | ~int32
}
```

### 预设错误码消息

| 函数 | 说明 |
|------|------|
| `RegisterCode[T Code](code T, message string)` | 注册单个错误码的预设消息 |
| `RegisterCodes[T Code](codeMessages map[T]string)` | 批量注册错误码的预设消息 |

### 预定义错误

| 函数 | 错误码 |
|------|--------|
| `InvalidParam(format string, a ...any) *MErr` | `ERR_INVALID_PARAM` |
| `Forbidden(format string, a ...any) *MErr` | `ERR_FORBIDDEN` |
| `Internal(format string, a ...any) *MErr` | `ERR_INTERNAL_ERROR` |
| `Timeout(format string, a ...any) *MErr` | `ERR_TIMEOUT` |
| `Duplicate(format string, a ...any) *MErr` | `ERR_DUPLICATE` |
| `TooManyRequests(format string, a ...any) *MErr` | `ERR_TOO_MANY_REQUESTS` |

### 判断错误

| 函数 | 说明 |
|------|------|
| `IsCode[T Code](err error, code T) bool` | 判断错误是否匹配指定错误码 |

### 接口

```go
type CodedError interface {
    error
    Code() string
    Unwrap() error
}
```

## 测试

```bash
cd merr
go test -v
```
