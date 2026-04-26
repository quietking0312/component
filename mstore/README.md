# mstore - 通用键值存储组件

基于 `sync.Map` 的线程安全键值存储，支持数据变更监听（Watch）。

## 特性

- 🧵 **线程安全**：基于 `sync.Map`，支持高并发读写
- 👁️ **变更监听**：支持注册 Watch 函数，数据变化时异步通知
- 🛡️ **Panic 保护**：Watch 回调中的 panic 会被捕获，不影响主流程

## 快速开始

### 基本使用

```go
package main

import (
    "fmt"
    "github.com/quietking0312/component/mstore"
)

func main() {
    store := mstore.NewStore()

    // 设置值
    err := store.Set("user:1", map[string]interface{}{
        "name": "Alice",
        "age":  30,
    })
    if err != nil {
        panic(err)
    }

    // 获取值
    val, err := store.Get("user:1")
    if err != nil {
        panic(err)
    }
    fmt.Println(val)
}
```

### 监听数据变化

```go
store := mstore.NewStore()

// 注册监听函数
store.Register("user:1", func(oldV, newV any) {
    fmt.Printf("数据变化: %v -> %v\n", oldV, newV)
})

// 修改数据，会自动触发 Watch 回调
store.Set("user:1", map[string]interface{}{
    "name": "Bob",
    "age":  25,
})
```

## API 文档

### 类型

```go
type WatchFunc func(oldV, newV any)
```

### 函数/方法

| 函数/方法 | 说明 |
|-----------|------|
| `NewStore() *Store` | 创建存储实例 |
| `(s *Store) Get(key string) (interface{}, error)` | 获取值 |
| `(s *Store) Set(k string, v interface{}) error` | 设置值 |
| `(s *Store) Register(key string, fc WatchFunc)` | 注册变更监听器 |

## 注意事项

1. Watch 回调是**异步执行**的，不保证实时性
2. 如果新旧值通过 `reflect.DeepEqual` 判断相等，不会触发 Watch
3. 每个 key 可以注册多个 Watch 函数
4. Watch 回调中的 panic 会被自动捕获，避免影响主程序

## 测试

```bash
cd mstore
go test -v
```
