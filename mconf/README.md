# mconf - 配置管理组件

基于 `spf13/viper` 封装的配置管理工具，支持 YAML/JSON/TOML 等多种格式、环境变量覆盖和配置文件热更新。

## 特性

- 📁 **多格式支持**：YAML、JSON、TOML、INI 等
- 🔧 **环境变量**：支持通过环境变量覆盖配置，自动转换命名格式
- 🔄 **热更新**：监听配置文件变化，自动重新加载
- 📦 **预置默认值**：内置常用的服务器、数据库、日志、JWT 默认值
- 🌐 **全局访问**：初始化后可在任何地方读取配置

## 快速开始

### 初始化配置

```go
package main

import (
    "github.com/quietking0312/component/mconf"
)

func main() {
    // 使用默认配置路径（./config.yaml 或 ./config/config.yaml）
    err := mconf.Init("", "APP")
    if err != nil {
        panic(err)
    }

    // 或指定配置文件
    // err := mconf.Init("./config/app.yaml", "APP")
}
```

### 读取配置

```go
// 读取各种类型
host := mconf.GetString("server.host")
port := mconf.GetInt("server.port")
timeout := mconf.GetDuration("server.read_timeout")
debug := mconf.GetBool("server.mode")

// 读取数组
adds := mconf.GetStringSlice("server.allow_hosts")

// 读取 Map
dbConfig := mconf.GetStringMap("database")

// 读取子配置
dbViper := mconf.Sub("database")
```

### 设置配置值

```go
// 运行时设置
mconf.Set("server.port", 9090)

// 设置默认值（仅在配置文件中未定义时生效）
mconf.SetDefault("server.max_connections", 1000)
```

### 配置文件热更新

```go
// 启用监听
mconf.WatchConfig()

// 注册变化回调
mconf.OnConfigChange(func() {
    fmt.Println("配置文件已更新！")
    // 重新加载相关配置
})
```

### 使用环境变量

初始化时传入前缀：`mconf.Init("", "APP")`

环境变量会自动映射：
- `APP_SERVER_HOST` → `server.host`
- `APP_DATABASE_PASSWORD` → `database.password`

### 写入配置

```go
// 写入当前配置文件
err := mconf.WriteConfig()

// 写入到指定路径
err := mconf.WriteConfigAs("/path/to/config.yaml")

// 安全写入（文件不存在时才写入）
err := mconf.SafeWriteConfig()
```

### 自定义 Viper 实例

```go
v := viper.New()
v.SetConfigFile("custom.yaml")
v.ReadInConfig()

mconf.InitWithConfig(v)
```

## 默认配置项

| 配置键 | 默认值 | 说明 |
|--------|--------|------|
| `server.host` | `0.0.0.0` | 服务器监听地址 |
| `server.port` | `8080` | 服务器端口 |
| `server.mode` | `debug` | 运行模式 |
| `server.read_timeout` | `60` | 读取超时（秒） |
| `server.write_timeout` | `60` | 写入超时（秒） |
| `database.driver` | `mysql` | 数据库驱动 |
| `database.host` | `localhost` | 数据库地址 |
| `database.port` | `3306` | 数据库端口 |
| `database.database` | `admin` | 数据库名 |
| `log.level` | `info` | 日志级别 |
| `log.format` | `console` | 日志格式 |
| `jwt.secret` | `your-secret-key` | JWT 密钥 |
| `jwt.expire` | `86400` | JWT 过期时间（秒） |

## API 文档

### 初始化

| 函数 | 说明 |
|------|------|
| `Init(configPath string, envPrefix string) error` | 初始化全局配置 |
| `InitWithConfig(v *viper.Viper)` | 使用自定义 viper 实例 |

### 读取配置

| 函数 | 说明 |
|------|------|
| `Get(key string) interface{}` | 获取原始值 |
| `GetString(key string) string` | 获取字符串 |
| `GetInt(key string) int` | 获取整数 |
| `GetInt64(key string) int64` | 获取 int64 |
| `GetBool(key string) bool` | 获取布尔值 |
| `GetFloat64(key string) float64` | 获取浮点数 |
| `GetDuration(key string) time.Duration` | 获取时长 |
| `GetStringSlice(key string) []string` | 获取字符串数组 |
| `GetIntSlice(key string) []int` | 获取整数数组 |
| `GetStringMap(key string) map[string]interface{}` | 获取 Map |
| `GetStringMapString(key string) map[string]string` | 获取 string Map |
| `IsSet(key string) bool` | 检查配置是否存在 |
| `Sub(key string) *viper.Viper` | 获取子配置 |
| `AllSettings() map[string]interface{}` | 获取所有配置 |

### 写入配置

| 函数 | 说明 |
|------|------|
| `Set(key string, value interface{})` | 设置配置值 |
| `SetDefault(key string, value interface{})` | 设置默认值 |
| `WriteConfig() error` | 写入配置文件 |
| `WriteConfigAs(filename string) error` | 写入到指定文件 |

### 配置监听

| 函数 | 说明 |
|------|------|
| `WatchConfig()` | 开始监听配置文件变化 |
| `OnConfigChange(run func())` | 注册变化回调 |

## 测试

```bash
cd mconf
go test -v
```
