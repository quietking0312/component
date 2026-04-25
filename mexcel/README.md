# mexcel - Excel 读写组件

基于 [excelize](https://github.com/xuri/excelize) 封装的 Excel 处理工具，提供简洁的 API 用于 Excel 文件的读写操作。

## 特性

- 🚀 简单易用的 API 设计
- 📝 结构体标签映射（`excel:"列名"`）
- 🔗 支持嵌套结构体（匿名嵌入）
- 💾 支持从文件/字节/Reader 读写
- 🌐 支持 HTTP 直接导出/导入
- 🎨 支持单元格样式、列宽、行高设置
- 📊 支持公式、合并单元格

## 安装

```bash
go get github.com/xuri/excelize/v2
```

## 快速开始

### 定义数据模型

```go
type User struct {
    ID       int     `excel:"ID"`
    Name     string  `excel:"姓名"`
    Age      int     `excel:"年龄"`
    Email    string  `excel:"邮箱"`
    Score    float64 `excel:"分数"`
    IsActive bool    `excel:"是否激活"`
}
```

### 导出数据

```go
// 准备数据
users := []User{
    {ID: 1, Name: "张三", Age: 25, Email: "zhangsan@example.com", Score: 85.5, IsActive: true},
    {ID: 2, Name: "李四", Age: 30, Email: "lisi@example.com", Score: 92.0, IsActive: false},
}

// 导出到文件
err := mexcel.Export(users, "users.xlsx")

// 导出到字节数组（用于 HTTP 响应）
data, err := mexcel.ExportToBytes(users)

// 导出到 io.Writer
err := mexcel.ExportToWriter(users, w)
```

### 导入数据

```go
var users []User

// 从文件导入
err := mexcel.Import("users.xlsx", &users)

// 从 io.Reader 导入
err := mexcel.ImportFromReader(file, &users)

// 指定表头行号（从1开始）
err := mexcel.ImportWithHeader("users.xlsx", &users, 2)
```

## 高级用法

### 自定义 Writer

```go
w := mexcel.NewWriter(
    mexcel.WithSheetName("用户数据"),
)
defer w.Close()

// 设置列宽
w.SetColWidth("A", "F", 20)

// 写入表头
w.WriteHeader("产品", "数量", "单价", "总价")

// 写入数据
w.WriteRow("苹果", 100, 5.5, 550)
w.WriteRow("香蕉", 200, 3.5, 700)

// 设置公式
w.SetCellFormula("D3", "=SUM(D1:D2)")

// 合并单元格
w.MergeCell("A1", "D1")

// 保存
err := w.SaveAs("products.xlsx")
```

### 嵌套结构体

```go
type Address struct {
    City   string `excel:"城市"`
    Street string `excel:"街道"`
}

type Person struct {
    ID      int     `excel:"ID"`
    Name    string  `excel:"姓名"`
    Address         // 匿名嵌入，字段会展开
}

// 导出时会生成列：ID, 姓名, 城市, 街道
```

### HTTP 导出示例

```go
func ExportHandler(w http.ResponseWriter, r *http.Request) {
    users := []User{
        {ID: 1, Name: "张三", Age: 25},
        {ID: 2, Name: "李四", Age: 30},
    }

    w.Header().Set("Content-Type", 
        "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
    w.Header().Set("Content-Disposition", 
        "attachment; filename=users.xlsx")

    err := mexcel.ExportToWriter(users, w)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
}
```

### HTTP 导入示例

```go
func ImportHandler(w http.ResponseWriter, r *http.Request) {
    file, _, err := r.FormFile("file")
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    defer file.Close()

    var users []User
    err = mexcel.ImportFromReader(file, &users)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    fmt.Fprintf(w, "成功导入 %d 条记录", len(users))
}
```

## 支持的类型

- 基础类型：`string`, `int`, `int8`, `int16`, `int32`, `int64`, `uint`, `uint8`, `uint16`, `uint32`, `uint64`, `float32`, `float64`, `bool`
- 指针类型：`*string`, `*int` 等
- 自定义类型：实现 `Convertible` 接口进行导出，`Parseable` 接口进行导入

## 接口扩展

### 自定义类型转换（导出）

```go
type Money float64

func (m Money) ToCell() (string, error) {
    return fmt.Sprintf("%.2f", m), nil
}
```

### 自定义类型解析（导入）

```go
func (m *Money) FromCell(s string) error {
    f, err := strconv.ParseFloat(s, 64)
    if err != nil {
        return err
    }
    *m = Money(f)
    return nil
}
```

## API 文档

### 便捷函数

| 函数 | 说明 |
|------|------|
| `Export(data, filename)` | 导出数据到文件 |
| `ExportToBytes(data)` | 导出数据到字节数组 |
| `ExportToWriter(data, writer)` | 导出数据到 io.Writer |
| `Import(filename, &dest)` | 从文件导入数据 |
| `ImportFromReader(reader, &dest)` | 从 io.Reader 导入数据 |
| `ImportWithHeader(filename, &dest, headerRow)` | 指定表头行号导入 |

### Writer 方法

| 方法 | 说明 |
|------|------|
| `NewWriter(opts...)` | 创建 Writer |
| `WriteHeader(headers...)` | 写入表头 |
| `WriteRow(values...)` | 写入一行数据 |
| `WriteStruct(data)` | 写入单个结构体 |
| `WriteStructs(data)` | 写入结构体切片 |
| `SetCellValue(cell, value)` | 设置单元格值 |
| `SetCellFormula(cell, formula)` | 设置单元格公式 |
| `SetColWidth(start, end, width)` | 设置列宽 |
| `SetRowHeight(row, height)` | 设置行高 |
| `MergeCell(start, end)` | 合并单元格 |
| `SaveAs(filename)` | 保存到文件 |
| `Bytes()` | 返回字节数组 |
| `WriteTo(writer)` | 写入 io.Writer |

### Reader 方法

| 方法 | 说明 |
|------|------|
| `Open(filename, opts...)` | 打开文件 |
| `OpenReader(reader, opts...)` | 从 io.Reader 打开 |
| `GetHeaders()` | 获取表头 |
| `GetRows()` | 获取所有数据行 |
| `ReadRow()` | 读取下一行 |
| `ReadStruct(&dest)` | 读取一行到结构体 |
| `ReadStructs(&dest)` | 读取所有行到结构体切片 |
| `RowCount()` | 返回总行数 |
| `DataRowCount()` | 返回数据行数 |

## 测试

```bash
cd admin_server
go test ./component/mexcel/... -v
```

## 许可证

MIT License
