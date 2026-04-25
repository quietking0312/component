package mexcel

import (
	"fmt"
	"io"
	"reflect"

	"github.com/xuri/excelize/v2"
)

// Reader Excel 读取器
type Reader struct {
	file      *excelize.File
	sheetName string
	rows      [][]string
	rowIndex  int
	tagName   string
	headers   []string
	headerMap map[string]int // 表头到列索引的映射
}

// ReaderOption 读取器选项
type ReaderOption func(*Reader)

// WithReaderSheetName 设置工作表名称
func WithReaderSheetName(name string) ReaderOption {
	return func(r *Reader) {
		r.sheetName = name
	}
}

// WithReaderTagName 设置结构体标签名
func WithReaderTagName(name string) ReaderOption {
	return func(r *Reader) {
		r.tagName = name
	}
}

// WithHeaderRow 指定表头行号（从1开始）
func WithHeaderRow(rowNum int) ReaderOption {
	return func(r *Reader) {
		if rowNum > 0 {
			r.rowIndex = rowNum - 1
		}
	}
}

// Open 打开 Excel 文件
func Open(filename string, opts ...ReaderOption) (*Reader, error) {
	file, err := excelize.OpenFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	r := &Reader{
		file:      file,
		sheetName: "",
		rowIndex:  0,
		tagName:   DefaultTag,
		headerMap: make(map[string]int),
	}

	for _, opt := range opts {
		opt(r)
	}

	// 如果没有指定工作表，使用第一个
	if r.sheetName == "" {
		sheets := file.GetSheetList()
		if len(sheets) == 0 {
			file.Close()
			return nil, fmt.Errorf("no sheets found in file")
		}
		r.sheetName = sheets[0]
	}

	// 读取所有行
	rows, err := file.GetRows(r.sheetName)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to get rows: %w", err)
	}
	r.rows = rows

	// 解析表头
	if r.rowIndex < len(rows) {
		r.headers = rows[r.rowIndex]
		for i, h := range r.headers {
			r.headerMap[h] = i
		}
		r.rowIndex++
	}

	return r, nil
}

// OpenReader 从 io.Reader 打开 Excel
func OpenReader(reader io.Reader, opts ...ReaderOption) (*Reader, error) {
	file, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to open reader: %w", err)
	}

	r := &Reader{
		file:      file,
		sheetName: "",
		rowIndex:  0,
		tagName:   DefaultTag,
		headerMap: make(map[string]int),
	}

	for _, opt := range opts {
		opt(r)
	}

	// 如果没有指定工作表，使用第一个
	if r.sheetName == "" {
		sheets := file.GetSheetList()
		if len(sheets) == 0 {
			file.Close()
			return nil, fmt.Errorf("no sheets found")
		}
		r.sheetName = sheets[0]
	}

	// 读取所有行
	rows, err := file.GetRows(r.sheetName)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to get rows: %w", err)
	}
	r.rows = rows

	// 解析表头
	if r.rowIndex < len(rows) {
		r.headers = rows[r.rowIndex]
		for i, h := range r.headers {
			r.headerMap[h] = i
		}
		r.rowIndex++
	}

	return r, nil
}

// Close 关闭读取器
func (r *Reader) Close() error {
	if r.file != nil {
		return r.file.Close()
	}
	return nil
}

// GetHeaders 获取表头
func (r *Reader) GetHeaders() []string {
	return r.headers
}

// GetRows 获取所有数据行（不包括表头）
func (r *Reader) GetRows() [][]string {
	if r.rowIndex >= len(r.rows) {
		return nil
	}
	return r.rows[r.rowIndex:]
}

// ReadRow 读取下一行
func (r *Reader) ReadRow() ([]string, bool) {
	if r.rowIndex >= len(r.rows) {
		return nil, false
	}
	row := r.rows[r.rowIndex]
	r.rowIndex++
	return row, true
}

// GetCellByHeader 根据表头名称获取单元格值
func (r *Reader) GetCellByHeader(row []string, header string) (string, bool) {
	idx, ok := r.headerMap[header]
	if !ok || idx >= len(row) {
		return "", false
	}
	return row[idx], true
}

// ReadStruct 读取一行并映射到结构体
func (r *Reader) ReadStruct(dest interface{}) error {
	row, ok := r.ReadRow()
	if !ok {
		return io.EOF
	}

	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("dest must be a pointer to struct")
	}

	elem := v.Elem()
	t := elem.Type()
	columns := getColumns(t, r.tagName)

	for _, col := range columns {
		// 查找对应的列索引
		idx, ok := r.headerMap[col.Name]
		if !ok {
			// 表头中没有这个字段，跳过
			continue
		}

		if idx >= len(row) {
			continue
		}

		cellValue := row[idx]

		// 转换值
		val, err := stringToValue(cellValue, col.Field.Type)
		if err != nil {
			return fmt.Errorf("failed to parse field %s: %w", col.Name, err)
		}

		// 获取目标字段（处理嵌套结构体）
		field, err := r.getField(elem, col.Index)
		if err != nil {
			continue
		}

		if !field.CanSet() {
			continue
		}

		field.Set(val)
	}

	return nil
}

// getField 根据索引路径获取字段值（支持嵌套结构体）
func (r *Reader) getField(v reflect.Value, index []int) (reflect.Value, error) {
	if len(index) == 0 {
		return reflect.Value{}, fmt.Errorf("empty index")
	}

	// 处理嵌套路径
	current := v
	for i := 0; i < len(index)-1; i++ {
		field := current.Field(index[i])
		// 如果是指针且为 nil，创建实例
		if field.Kind() == reflect.Ptr && field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		// 解引用指针
		if field.Kind() == reflect.Ptr {
			field = field.Elem()
		}
		current = field
	}

	return current.Field(index[len(index)-1]), nil
}

// ReadStructs 读取所有行并映射到结构体切片
// dest 必须是指向结构体切片的指针，如: &[]User
func (r *Reader) ReadStructs(dest interface{}) error {
	if !isSlicePtr(dest) {
		return fmt.Errorf("dest must be a pointer to slice")
	}

	elemType := getElemType(dest)
	sliceVal := reflect.ValueOf(dest).Elem()

	for {
		// 创建元素实例
		var elem reflect.Value
		if elemType.Kind() == reflect.Ptr {
			// 元素类型是指针，如 *User
			elem = reflect.New(elemType.Elem())
		} else {
			// 元素类型是值，如 User
			elem = reflect.New(elemType)
		}

		if err := r.ReadStruct(elem.Interface()); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		// 根据元素类型决定如何添加到切片
		if elemType.Kind() == reflect.Ptr {
			sliceVal.Set(reflect.Append(sliceVal, elem))
		} else {
			sliceVal.Set(reflect.Append(sliceVal, elem.Elem()))
		}
	}

	return nil
}

// GetSheetList 获取所有工作表名称
func (r *Reader) GetSheetList() []string {
	return r.file.GetSheetList()
}

// SetSheet 切换到指定工作表
func (r *Reader) SetSheet(name string) error {
	rows, err := r.file.GetRows(name)
	if err != nil {
		return fmt.Errorf("failed to get rows: %w", err)
	}

	r.sheetName = name
	r.rows = rows
	r.rowIndex = 0
	r.headers = nil
	r.headerMap = make(map[string]int)

	// 重新解析表头
	if len(rows) > 0 {
		r.headers = rows[0]
		for i, h := range r.headers {
			r.headerMap[h] = i
		}
		r.rowIndex = 1
	}

	return nil
}

// RowCount 返回总行数（包括表头）
func (r *Reader) RowCount() int {
	return len(r.rows)
}

// DataRowCount 返回数据行数（不包括表头）
func (r *Reader) DataRowCount() int {
	count := len(r.rows) - r.rowIndex
	if count < 0 {
		return 0
	}
	return count
}

// Reset 重置到数据开始位置
func (r *Reader) Reset() {
	if len(r.rows) > 0 {
		r.rowIndex = 1
	} else {
		r.rowIndex = 0
	}
}

// ==================== 便捷函数 ====================

// Import 从 Excel 文件导入数据
// dest 必须是指向结构体切片的指针，如: &[]User
func Import(filename string, dest interface{}, opts ...ReaderOption) error {
	r, err := Open(filename, opts...)
	if err != nil {
		return err
	}
	defer r.Close()

	return r.ReadStructs(dest)
}

// ImportFromReader 从 io.Reader 导入数据
func ImportFromReader(reader io.Reader, dest interface{}, opts ...ReaderOption) error {
	r, err := OpenReader(reader, opts...)
	if err != nil {
		return err
	}
	defer r.Close()

	return r.ReadStructs(dest)
}

// ImportWithHeader 从 Excel 文件导入数据（自定义表头行）
func ImportWithHeader(filename string, dest interface{}, headerRow int, opts ...ReaderOption) error {
	opts = append(opts, WithHeaderRow(headerRow))
	return Import(filename, dest, opts...)
}
