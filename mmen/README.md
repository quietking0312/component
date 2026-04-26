# mmen - 系统内存与进程信息组件

获取系统内存使用情况和进程列表，支持 Windows 和 Linux 平台。

## 特性

- 🖥️ **跨平台**：支持 Windows 和 Linux
- 📊 **进程信息**：获取进程 PID、名称、命令行参数
- 💾 **内存统计**：系统内存使用情况（待扩展）

## 快速开始

### 获取进程列表

```go
package main

import (
    "fmt"
    "github.com/quietking0312/component/mmen"
)

func main() {
    processes, err := mmen.GetProcesses()
    if err != nil {
        panic(err)
    }

    for _, p := range processes {
        fmt.Printf("PID: %d, Name: %s, Cmd: %s\n", p.Pid, p.Name, p.Cmdline)
    }
}
```

## API 文档

### 类型

```go
type Process struct {
    Pid     int    // 进程 ID
    Name    string // 进程名称
    Cmdline string // 命令行参数
}
```

### 函数

| 函数 | 说明 |
|------|------|
| `GetProcesses() ([]Process, error)` | 获取所有进程列表 |

## 测试

```bash
cd mmen
go test -v
```
