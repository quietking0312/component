package mexcel

import (
	"fmt"
	"log"
	"net/http"
)

// User 用户结构体示例
type User struct {
	ID       int     `excel:"ID"`
	Name     string  `excel:"姓名"`
	Age      int     `excel:"年龄"`
	Email    string  `excel:"邮箱"`
	Phone    string  `excel:"电话"`
	Score    float64 `excel:"分数"`
	IsActive bool    `excel:"是否激活"`
}

// ExampleExport 导出示例
func ExampleExport() {
	// 准备数据
	users := []User{
		{ID: 1, Name: "张三", Age: 25, Email: "zhangsan@example.com", Phone: "13800138001", Score: 85.5, IsActive: true},
		{ID: 2, Name: "李四", Age: 30, Email: "lisi@example.com", Phone: "13800138002", Score: 92.0, IsActive: true},
		{ID: 3, Name: "王五", Age: 28, Email: "wangwu@example.com", Phone: "13800138003", Score: 78.5, IsActive: false},
	}

	// 导出到文件
	err := Export(users, "users.xlsx")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("导出成功: users.xlsx")
}

// ExampleExportToBytes 导出到字节数组示例
func ExampleExportToBytes() {
	users := []User{
		{ID: 1, Name: "张三", Age: 25, Email: "zhangsan@example.com"},
		{ID: 2, Name: "李四", Age: 30, Email: "lisi@example.com"},
	}

	// 导出到字节数组（可用于 HTTP 响应）
	data, err := ExportToBytes(users)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("导出成功，大小: %d bytes\n", len(data))
}

// ExampleImport 导入示例
func ExampleImport() {
	var users []User

	// 从文件导入
	err := Import("users.xlsx", &users)
	if err != nil {
		log.Fatal(err)
	}

	// 打印导入的数据
	for _, u := range users {
		fmt.Printf("ID: %d, Name: %s, Age: %d\n", u.ID, u.Name, u.Age)
	}
}

// ExampleWriter 使用 Writer 自定义导出
func ExampleWriter() {
	w := NewWriter()
	defer w.Close()

	// 设置列宽
	w.SetColWidth("A", "C", 20)

	// 写入表头
	w.WriteHeader("产品", "数量", "单价", "总价")

	// 写入数据
	w.WriteRow("苹果", 100, 5.5, 550)
	w.WriteRow("香蕉", 200, 3.5, 700)
	w.WriteRow("橙子", 150, 4.0, 600)

	// 添加公式行
	w.WriteRow("总计", "", "", "")
	w.SetCellFormula("D5", "=SUM(D2:D4)")

	// 保存文件
	err := w.SaveAs("products.xlsx")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("导出成功: products.xlsx")
}

// ExampleReader 使用 Reader 读取
func ExampleReader() {
	r, err := Open("users.xlsx")
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()

	// 获取表头
	headers := r.GetHeaders()
	fmt.Println("表头:", headers)

	// 读取所有行
	rows := r.GetRows()
	for i, row := range rows {
		fmt.Printf("第%d行: %v\n", i+1, row)
	}
}

// ExampleWriter_advanced 高级用法示例
func ExampleWriter_advanced() {
	// 创建带有自定义配置的工作簿
	w := NewWriter(
		WithSheetName("销售数据"), // 自定义工作表名称
		WithTagName("excel"),  // 自定义标签名（默认就是 excel）
	)
	defer w.Close()

	// 写入多个工作表
	w.AddSheet("汇总")
	w.SetActiveSheet("汇总")

	// 写入合并单元格
	w.WriteRow("销售汇总报表")
	w.MergeCell("A1", "D1")

	// 回到第一个工作表写入数据
	w.SetActiveSheet("销售数据")

	// 写入结构体数据
	users := []User{
		{ID: 1, Name: "张三", Age: 25, Email: "zhangsan@example.com", Score: 85.5},
		{ID: 2, Name: "李四", Age: 30, Email: "lisi@example.com", Score: 92.0},
	}
	w.WriteStructs(users)

	err := w.SaveAs("advanced.xlsx")
	if err != nil {
		log.Fatal(err)
	}
}

// ExampleImportWithHeader 从指定行开始读取
func ExampleImportWithHeader() {
	var users []User

	// 从第2行开始读取（表头在第2行）
	err := ImportWithHeader("users.xlsx", &users, 2)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("导入 %d 条记录\n", len(users))
}

// Example_exportHTTPHandler 导出 HTTP Handler 示例
func Example_exportHTTPHandler() {
	// 导出 Handler
	handler := func(w http.ResponseWriter, r *http.Request) {
		users := []User{
			{ID: 1, Name: "张三", Age: 25, Email: "zhangsan@example.com"},
			{ID: 2, Name: "李四", Age: 30, Email: "lisi@example.com"},
		}

		// 设置响应头
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", "attachment; filename=users.xlsx")

		// 直接写入响应
		err := ExportToWriter(users, w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	_ = handler
}

// Example_importHTTPHandler 导入 HTTP Handler 示例
func Example_importHTTPHandler() {
	// 导入 Handler
	handler := func(w http.ResponseWriter, r *http.Request) {
		// 解析上传的文件
		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		var users []User
		err = ImportFromReader(file, &users)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "成功导入 %d 条记录", len(users))
	}

	_ = handler
}
