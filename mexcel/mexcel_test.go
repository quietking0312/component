package mexcel

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 测试结构体
type TestUser struct {
	ID       int     `excel:"ID"`
	Name     string  `excel:"姓名"`
	Age      int     `excel:"年龄"`
	Email    string  `excel:"邮箱"`
	Score    float64 `excel:"分数"`
	IsActive bool    `excel:"是否激活"`
}

// 测试嵌套结构体
type Address struct {
	City   string `excel:"城市"`
	Street string `excel:"街道"`
}

type Person struct {
	ID      int    `excel:"ID"`
	Name    string `excel:"姓名"`
	Address        // 匿名嵌入
}

func TestExportAndImport(t *testing.T) {
	// 准备测试数据
	users := []TestUser{
		{ID: 1, Name: "张三", Age: 25, Email: "zhangsan@test.com", Score: 85.5, IsActive: true},
		{ID: 2, Name: "李四", Age: 30, Email: "lisi@test.com", Score: 92.0, IsActive: false},
		{ID: 3, Name: "王五", Age: 28, Email: "wangwu@test.com", Score: 78.5, IsActive: true},
	}

	// 创建临时文件
	tmpFile := "test_export.xlsx"
	defer os.Remove(tmpFile)

	// 测试导出
	err := Export(users, tmpFile)
	require.NoError(t, err)

	// 验证文件存在
	_, err = os.Stat(tmpFile)
	require.NoError(t, err)

	// 测试导入
	var importedUsers []TestUser
	err = Import(tmpFile, &importedUsers)
	require.NoError(t, err)

	// 验证数据
	assert.Equal(t, len(users), len(importedUsers))
	for i, u := range users {
		assert.Equal(t, u.ID, importedUsers[i].ID)
		assert.Equal(t, u.Name, importedUsers[i].Name)
		assert.Equal(t, u.Age, importedUsers[i].Age)
		assert.Equal(t, u.Email, importedUsers[i].Email)
		assert.InDelta(t, u.Score, importedUsers[i].Score, 0.01)
		assert.Equal(t, u.IsActive, importedUsers[i].IsActive)
	}
}

func TestExportToBytes(t *testing.T) {
	users := []TestUser{
		{ID: 1, Name: "张三", Age: 25, Email: "zhangsan@test.com"},
	}

	// 测试导出到字节数组
	data, err := ExportToBytes(users)
	require.NoError(t, err)
	assert.Greater(t, len(data), 0)

	// 测试从字节数组导入
	var importedUsers []TestUser
	err = ImportFromReader(bytes.NewReader(data), &importedUsers)
	require.NoError(t, err)
	assert.Equal(t, 1, len(importedUsers))
	assert.Equal(t, "张三", importedUsers[0].Name)
}

func TestWriter(t *testing.T) {
	tmpFile := "test_writer.xlsx"
	defer os.Remove(tmpFile)

	w := NewWriter()

	// 测试写入表头
	err := w.WriteHeader("列1", "列2", "列3")
	require.NoError(t, err)

	// 测试写入数据行
	err = w.WriteRow("a1", "b1", "c1")
	require.NoError(t, err)
	err = w.WriteRow("a2", "b2", "c2")
	require.NoError(t, err)

	// 测试写入结构体
	err = w.WriteStruct(TestUser{ID: 1, Name: "测试", Age: 20, Email: "test@test.com"})
	require.NoError(t, err)

	// 保存文件
	err = w.SaveAs(tmpFile)
	require.NoError(t, err)
	w.Close()

	// 验证文件
	_, err = os.Stat(tmpFile)
	require.NoError(t, err)
}

func TestWriterStructs(t *testing.T) {
	tmpFile := "test_writer_structs.xlsx"
	defer os.Remove(tmpFile)

	users := []TestUser{
		{ID: 1, Name: "张三", Age: 25},
		{ID: 2, Name: "李四", Age: 30},
	}

	w := NewWriter()
	err := w.WriteStructs(users)
	require.NoError(t, err)

	err = w.SaveAs(tmpFile)
	require.NoError(t, err)
	w.Close()

	// 验证
	var imported []TestUser
	err = Import(tmpFile, &imported)
	require.NoError(t, err)
	assert.Equal(t, 2, len(imported))
}

func TestReader(t *testing.T) {
	tmpFile := "test_reader.xlsx"
	defer os.Remove(tmpFile)

	// 创建测试文件
	w := NewWriter()
	w.WriteHeader("A", "B", "C")
	w.WriteRow("1", "2", "3")
	w.WriteRow("4", "5", "6")
	w.SaveAs(tmpFile)
	w.Close()

	// 读取测试
	r, err := Open(tmpFile)
	require.NoError(t, err)
	defer r.Close()

	// 验证表头
	headers := r.GetHeaders()
	assert.Equal(t, []string{"A", "B", "C"}, headers)

	// 验证行数
	assert.Equal(t, 3, r.RowCount())
	assert.Equal(t, 2, r.DataRowCount())

	// 验证读取行
	row, ok := r.ReadRow()
	require.True(t, ok)
	assert.Equal(t, []string{"1", "2", "3"}, row)

	row, ok = r.ReadRow()
	require.True(t, ok)
	assert.Equal(t, []string{"4", "5", "6"}, row)

	_, ok = r.ReadRow()
	assert.False(t, ok)
}

func TestReaderReadStruct(t *testing.T) {
	tmpFile := "test_reader_struct.xlsx"
	defer os.Remove(tmpFile)

	// 创建测试文件
	w := NewWriter()
	w.WriteStructs([]TestUser{
		{ID: 1, Name: "张三", Age: 25, Email: "zhangsan@test.com", Score: 85.5, IsActive: true},
	})
	w.SaveAs(tmpFile)
	w.Close()

	// 读取
	r, err := Open(tmpFile)
	require.NoError(t, err)
	defer r.Close()

	var user TestUser
	err = r.ReadStruct(&user)
	require.NoError(t, err)

	assert.Equal(t, 1, user.ID)
	assert.Equal(t, "张三", user.Name)
	assert.Equal(t, 25, user.Age)
	assert.Equal(t, "zhangsan@test.com", user.Email)
	assert.InDelta(t, 85.5, user.Score, 0.01)
	assert.True(t, user.IsActive)
}

func TestReaderReadStructs(t *testing.T) {
	tmpFile := "test_reader_structs.xlsx"
	defer os.Remove(tmpFile)

	// 创建测试文件
	users := []TestUser{
		{ID: 1, Name: "张三", Age: 25},
		{ID: 2, Name: "李四", Age: 30},
		{ID: 3, Name: "王五", Age: 28},
	}

	w := NewWriter()
	w.WriteStructs(users)
	w.SaveAs(tmpFile)
	w.Close()

	// 读取
	r, err := Open(tmpFile)
	require.NoError(t, err)
	defer r.Close()

	var imported []TestUser
	err = r.ReadStructs(&imported)
	require.NoError(t, err)

	assert.Equal(t, 3, len(imported))
	assert.Equal(t, "张三", imported[0].Name)
	assert.Equal(t, "李四", imported[1].Name)
	assert.Equal(t, "王五", imported[2].Name)
}

func TestNestedStruct(t *testing.T) {
	tmpFile := "test_nested.xlsx"
	defer os.Remove(tmpFile)

	persons := []Person{
		{ID: 1, Name: "张三", Address: Address{City: "北京", Street: "长安街"}},
		{ID: 2, Name: "李四", Address: Address{City: "上海", Street: "南京路"}},
	}

	// 导出
	err := Export(persons, tmpFile)
	require.NoError(t, err)

	// 导入
	var imported []Person
	err = Import(tmpFile, &imported)
	require.NoError(t, err)

	assert.Equal(t, 2, len(imported))
	assert.Equal(t, "北京", imported[0].City)
	assert.Equal(t, "长安街", imported[0].Street)
}

func TestEmptyData(t *testing.T) {
	tmpFile := "test_empty.xlsx"
	defer os.Remove(tmpFile)

	// 空切片导出
	var emptyUsers []TestUser
	err := Export(emptyUsers, tmpFile)
	require.NoError(t, err)

	// 导入空文件（只有表头）
	var imported []TestUser
	err = Import(tmpFile, &imported)
	require.NoError(t, err)
	assert.Equal(t, 0, len(imported))
}

func TestPointerSlice(t *testing.T) {
	tmpFile := "test_pointer.xlsx"
	defer os.Remove(tmpFile)

	// 指针切片
	users := []*TestUser{
		{ID: 1, Name: "张三"},
		{ID: 2, Name: "李四"},
	}

	w := NewWriter()
	err := w.WriteStructs(users)
	require.NoError(t, err)
	err = w.SaveAs(tmpFile)
	require.NoError(t, err)
	w.Close()

	var imported []TestUser
	err = Import(tmpFile, &imported)
	require.NoError(t, err)
	assert.Equal(t, 2, len(imported))
}

func TestCustomOptions(t *testing.T) {
	tmpFile := "test_custom.xlsx"
	defer os.Remove(tmpFile)

	// 自定义工作表名称
	w := NewWriter(WithSheetName("自定义表"))
	w.WriteHeader("A", "B")
	w.WriteRow("1", "2")
	err := w.SaveAs(tmpFile)
	require.NoError(t, err)
	w.Close()

	// 使用自定义选项读取
	r, err := Open(tmpFile, WithReaderSheetName("自定义表"))
	require.NoError(t, err)
	defer r.Close()

	assert.Equal(t, "自定义表", r.sheetName)
}

func TestColumnWidthAndRowHeight(t *testing.T) {
	tmpFile := "test_style.xlsx"
	defer os.Remove(tmpFile)

	w := NewWriter()

	// 设置列宽
	err := w.SetColWidth("A", "C", 20)
	require.NoError(t, err)

	// 设置行高
	err = w.SetRowHeight(1, 30)
	require.NoError(t, err)

	w.WriteHeader("A", "B", "C")
	err = w.SaveAs(tmpFile)
	require.NoError(t, err)
	w.Close()

	// 验证文件创建成功
	_, err = os.Stat(tmpFile)
	require.NoError(t, err)
}

func TestMergeCell(t *testing.T) {
	tmpFile := "test_merge.xlsx"
	defer os.Remove(tmpFile)

	w := NewWriter()

	w.WriteRow("合并单元格测试")
	err := w.MergeCell("A1", "C1")
	require.NoError(t, err)

	w.WriteHeader("A", "B", "C")
	w.WriteRow("1", "2", "3")

	err = w.SaveAs(tmpFile)
	require.NoError(t, err)
	w.Close()

	_, err = os.Stat(tmpFile)
	require.NoError(t, err)
}

func TestValueToString(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{"string", "test", "test"},
		{"int", 42, "42"},
		{"int64", int64(9223372036854775807), "9223372036854775807"},
		{"float64", 3.14159, "3.14159"},
		{"bool", true, "true"},
		{"slice", []int{1, 2, 3}, "[1,2,3]"},
		{"slice_empty", []int{}, "[]"},
		{"slice_nil", []int(nil), ""},
		{"array", [3]int{1, 2, 3}, "[1,2,3]"},
		{"map", map[string]int{"a": 1, "b": 2}, `{"a":1,"b":2}`},
		{"map_nil", map[string]int(nil), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := reflect.ValueOf(tt.value)
			result, err := valueToString(v)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStringToValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		target   reflect.Type
		expected interface{}
	}{
		{"string", "test", reflect.TypeOf(""), "test"},
		{"int", "42", reflect.TypeOf(0), 42},
		{"int64", "9223372036854775807", reflect.TypeOf(int64(0)), int64(9223372036854775807)},
		{"float64", "3.14159", reflect.TypeOf(float64(0)), 3.14159},
		{"bool", "true", reflect.TypeOf(true), true},
		{"slice", "[1,2,3]", reflect.TypeOf([]int{}), []int{1, 2, 3}},
		{"slice_empty", "", reflect.TypeOf([]int{}), []int(nil)},
		{"map", `{"a":1,"b":2}`, reflect.TypeOf(map[string]int{}), map[string]int{"a": 1, "b": 2}},
		{"map_empty", "", reflect.TypeOf(map[string]int{}), map[string]int(nil)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := stringToValue(tt.input, tt.target)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result.Interface())
		})
	}
}

func TestGetColumns(t *testing.T) {
	type TestStruct struct {
		ID      int    `excel:"编号"`
		Name    string `excel:"名称"`
		Ignored string `excel:"-"`
		private string // 未导出字段
	}

	cols := getColumns(reflect.TypeOf(TestStruct{}), DefaultTag)
	assert.Equal(t, 2, len(cols))
	assert.Equal(t, "编号", cols[0].Name)
	assert.Equal(t, "名称", cols[1].Name)
}

func TestImportWithHeaderRow(t *testing.T) {
	tmpFile := "test_header_row.xlsx"
	defer os.Remove(tmpFile)

	// 创建一个带空行的文件
	w := NewWriter()
	w.WriteRow("", "", "")       // 空行
	w.WriteRow("ID", "姓名", "年龄") // 表头在第2行
	w.WriteRow("1", "张三", "25")
	w.WriteRow("2", "李四", "30")
	w.SaveAs(tmpFile)
	w.Close()

	// 从第2行开始读取
	var users []TestUser
	err := ImportWithHeader(tmpFile, &users, 2)
	require.NoError(t, err)
	assert.Equal(t, 2, len(users))
}

func BenchmarkExport(b *testing.B) {
	// 准备大量数据
	users := make([]TestUser, 1000)
	for i := 0; i < 1000; i++ {
		users[i] = TestUser{
			ID:       i + 1,
			Name:     fmt.Sprintf("用户%d", i+1),
			Age:      20 + i%50,
			Email:    fmt.Sprintf("user%d@test.com", i+1),
			Score:    float64(i) * 0.5,
			IsActive: i%2 == 0,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := new(bytes.Buffer)
		ExportToWriter(users, buf)
	}
}

func BenchmarkImport(b *testing.B) {
	// 先创建测试文件
	tmpFile := "bench_import.xlsx"
	defer os.Remove(tmpFile)

	users := make([]TestUser, 1000)
	for i := 0; i < 1000; i++ {
		users[i] = TestUser{
			ID:    i + 1,
			Name:  fmt.Sprintf("用户%d", i+1),
			Age:   20 + i%50,
			Email: fmt.Sprintf("user%d@test.com", i+1),
		}
	}
	Export(users, tmpFile)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var imported []TestUser
		Import(tmpFile, &imported)
	}
}
