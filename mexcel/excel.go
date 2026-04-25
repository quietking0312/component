// Package mexcel Excel 读写组件
// 基于 github.com/xuri/excelize/v2 封装的 Excel 处理工具
//
// 基本使用示例:
//
// 写入 Excel:
//
//	// 定义数据结构
//	type User struct {
//	    Name   string `excel:"姓名"`
//	    Age    int    `excel:"年龄"`
//	    Email  string `excel:"邮箱"`
//	}
//
//	// 创建数据
//	users := []User{
//	    {Name: "张三", Age: 25, Email: "zhangsan@example.com"},
//	    {Name: "李四", Age: 30, Email: "lisi@example.com"},
//	}
//
//	// 写入文件
//	err := mexcel.Export(users, "users.xlsx")
//
// 读取 Excel:
//
//	var users []User
//	err := mexcel.Import("users.xlsx", &users)
//
// 使用 Writer 自定义写入:
//
//	w := mexcel.NewWriter()
//	defer w.Close()
//
//	// 写入表头
//	w.WriteHeader("姓名", "年龄", "邮箱")
//
//	// 写入数据行
//	w.WriteRow("张三", 25, "zhangsan@example.com")
//	w.WriteRow("李四", 30, "lisi@example.com")
//
//	// 保存文件
//	err := w.SaveAs("users.xlsx")
//
// 使用 Reader 自定义读取:
//
//	r, err := mexcel.Open("users.xlsx")
//	if err != nil {
//	    return err
//	}
//	defer r.Close()
//
//	// 读取所有行
//	rows, err := r.GetRows()
//	for _, row := range rows {
//	    // 处理每一行数据
//	}
package mexcel

import (
	"fmt"
	"reflect"
	"strconv"
)

// Version 版本号
const Version = "v1.0.0"

// DefaultTag 默认结构体标签名
const DefaultTag = "excel"

// Convertible 类型转换接口
type Convertible interface {
	ToCell() (string, error)
}

// Parseable 单元格解析接口
type Parseable interface {
	FromCell(string) error
}

// columnInfo 列信息
type columnInfo struct {
	Index []int               // 字段索引路径（支持嵌套结构体）
	Name  string              // 列名（来自标签或字段名）
	Field reflect.StructField // 字段信息
}

// parseTag 解析结构体标签
func parseTag(field reflect.StructField, tagName string) (string, bool) {
	tag := field.Tag.Get(tagName)
	if tag == "" {
		return "", false
	}
	// 处理 tag 选项，如 `excel:"姓名,omitempty"`
	if idx := len(tag); idx > 0 {
		for i, c := range tag {
			if c == ',' {
				return tag[:i], true
			}
		}
	}
	return tag, true
}

// getColumns 获取结构体的列信息
func getColumns(t reflect.Type, tagName string) []columnInfo {
	return getColumnsRecursive(t, tagName, nil)
}

// getColumnsRecursive 递归获取列信息（支持嵌套结构体）
func getColumnsRecursive(t reflect.Type, tagName string, parentIndex []int) []columnInfo {
	var columns []columnInfo
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// 跳过未导出字段
		if !field.IsExported() {
			continue
		}

		// 构建完整的索引路径
		index := append([]int{}, parentIndex...)
		index = append(index, i)

		// 处理嵌套结构体（匿名嵌入）
		if field.Anonymous && field.Type.Kind() == reflect.Struct {
			nestedCols := getColumnsRecursive(field.Type, tagName, index)
			columns = append(columns, nestedCols...)
			continue
		}

		// 解析标签
		name, ok := parseTag(field, tagName)
		if !ok {
			// 没有标签，使用字段名
			name = field.Name
		}
		// 跳过标记为 `-` 的字段
		if name == "-" {
			continue
		}

		columns = append(columns, columnInfo{
			Index: index,
			Name:  name,
			Field: field,
		})
	}
	return columns
}

// valueToString 将值转换为字符串
func valueToString(v reflect.Value) (string, error) {
	if !v.IsValid() {
		return "", nil
	}

	// 实现 Convertible 接口
	if v.CanInterface() {
		if c, ok := v.Interface().(Convertible); ok {
			return c.ToCell()
		}
	}

	switch v.Kind() {
	case reflect.String:
		return v.String(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10), nil
	case reflect.Float32:
		return strconv.FormatFloat(v.Float(), 'f', -1, 32), nil
	case reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'f', -1, 64), nil
	case reflect.Bool:
		return strconv.FormatBool(v.Bool()), nil
	default:
		// 处理指针
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				return "", nil
			}
			return valueToString(v.Elem())
		}
		// 其他类型使用 fmt.Sprint
		return fmt.Sprint(v.Interface()), nil
	}
}

// stringToValue 将字符串转换为值
func stringToValue(s string, t reflect.Type) (reflect.Value, error) {
	// 实现 Parseable 接口
	ptr := reflect.New(t)
	if p, ok := ptr.Interface().(Parseable); ok {
		if err := p.FromCell(s); err != nil {
			return reflect.Value{}, err
		}
		return ptr.Elem(), nil
	}

	switch t.Kind() {
	case reflect.String:
		return reflect.ValueOf(s), nil
	case reflect.Int:
		i, err := strconv.ParseInt(s, 10, 64)
		return reflect.ValueOf(int(i)), err
	case reflect.Int8:
		i, err := strconv.ParseInt(s, 10, 8)
		return reflect.ValueOf(int8(i)), err
	case reflect.Int16:
		i, err := strconv.ParseInt(s, 10, 16)
		return reflect.ValueOf(int16(i)), err
	case reflect.Int32:
		i, err := strconv.ParseInt(s, 10, 32)
		return reflect.ValueOf(int32(i)), err
	case reflect.Int64:
		i, err := strconv.ParseInt(s, 10, 64)
		return reflect.ValueOf(i), err
	case reflect.Uint:
		u, err := strconv.ParseUint(s, 10, 64)
		return reflect.ValueOf(uint(u)), err
	case reflect.Uint8:
		u, err := strconv.ParseUint(s, 10, 8)
		return reflect.ValueOf(uint8(u)), err
	case reflect.Uint16:
		u, err := strconv.ParseUint(s, 10, 16)
		return reflect.ValueOf(uint16(u)), err
	case reflect.Uint32:
		u, err := strconv.ParseUint(s, 10, 32)
		return reflect.ValueOf(uint32(u)), err
	case reflect.Uint64:
		u, err := strconv.ParseUint(s, 10, 64)
		return reflect.ValueOf(u), err
	case reflect.Float32:
		f, err := strconv.ParseFloat(s, 32)
		return reflect.ValueOf(float32(f)), err
	case reflect.Float64:
		f, err := strconv.ParseFloat(s, 64)
		return reflect.ValueOf(f), err
	case reflect.Bool:
		b, err := strconv.ParseBool(s)
		return reflect.ValueOf(b), err
	default:
		// 处理指针
		if t.Kind() == reflect.Ptr {
			if s == "" {
				return reflect.Zero(t), nil
			}
			elem, err := stringToValue(s, t.Elem())
			if err != nil {
				return reflect.Value{}, err
			}
			ptr := reflect.New(t.Elem())
			ptr.Elem().Set(elem)
			return ptr, nil
		}
		return reflect.Value{}, fmt.Errorf("unsupported type: %s", t.Kind())
	}
}

// isSlicePtr 检查是否为指向切片的指针
func isSlicePtr(v interface{}) bool {
	t := reflect.TypeOf(v)
	return t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Slice
}

// getElemType 获取切片的元素类型
func getElemType(v interface{}) reflect.Type {
	t := reflect.TypeOf(v).Elem().Elem()
	// 如果元素是指针，返回指针指向的类型
	if t.Kind() == reflect.Ptr {
		return t.Elem()
	}
	return t
}
