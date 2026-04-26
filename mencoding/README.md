# mencoding - 编码检测与转换组件

自动检测字节流的字符编码，并将其转换为 UTF-8。

## 特性

- 🔍 **自动检测**：支持检测 UTF-8、GBK 等常见编码
- 🔄 **自动转换**：将检测到的编码自动转换为 UTF-8
- 📖 **流式处理**：支持 `bufio.Reader` 流式读取和转换

## 快速开始

### 检测编码

```go
package main

import (
    "fmt"
    "github.com/quietking0312/component/mencoding"
)

func main() {
    data := []byte("你好，世界")
    encoding := mencoding.GetStrCoding(data)
    fmt.Println("编码:", encoding) // UTF-8
}
```

### 转换为 UTF-8

```go
data := []byte{0xc4, 0xe3, 0xba, 0xc3} // GBK 编码的 "你好"
utf8Data, err := mencoding.Byte2Utf8(data)
if err != nil {
    panic(err)
}
fmt.Println(string(utf8Data))
```

### 流式解码

```go
reader := bufio.NewReader(file)
utf8Bytes, err := mencoding.Decoder(reader)
if err != nil {
    panic(err)
}
```

## API 文档

| 函数 | 说明 |
|------|------|
| `GetStrCoding(data []byte) string` | 检测字节流的字符编码 |
| `Byte2Utf8(data []byte) ([]byte, error)` | 将字节流转换为 UTF-8 |
| `Decoder(r *bufio.Reader) ([]byte, error)` | 从 Reader 中读取并转换为 UTF-8 |

## 支持的编码

- UTF-8
- GBK

## 测试

```bash
cd mencoding
go test -v
```
