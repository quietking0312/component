package mexcel

import (
	"fmt"
	"io"
	"reflect"

	"github.com/xuri/excelize/v2"
)

// Writer Excel 写入器
type Writer struct {
	file      *excelize.File
	sheetName string
	rowNum    int
	tagName   string
}

// WriterOption 写入器选项
type WriterOption func(*Writer)

// WithSheetName 设置工作表名称
func WithSheetName(name string) WriterOption {
	return func(w *Writer) {
		w.sheetName = name
	}
}

// WithTagName 设置结构体标签名
func WithTagName(name string) WriterOption {
	return func(w *Writer) {
		w.tagName = name
	}
}

// NewWriter 创建新的 Excel 写入器
func NewWriter(opts ...WriterOption) *Writer {
	w := &Writer{
		file:      excelize.NewFile(),
		sheetName: "Sheet1",
		rowNum:    1,
		tagName:   DefaultTag,
	}

	for _, opt := range opts {
		opt(w)
	}

	// 创建默认工作表
	if w.sheetName != "Sheet1" {
		w.file.NewSheet(w.sheetName)
		w.file.DeleteSheet("Sheet1")
	}

	return w
}

// WriteHeader 写入表头
func (w *Writer) WriteHeader(headers ...string) error {
	for i, header := range headers {
		cell, err := excelize.CoordinatesToCellName(i+1, w.rowNum)
		if err != nil {
			return fmt.Errorf("failed to get cell name: %w", err)
		}
		if err := w.file.SetCellValue(w.sheetName, cell, header); err != nil {
			return fmt.Errorf("failed to set header: %w", err)
		}
	}
	w.rowNum++
	return nil
}

// WriteRow 写入一行数据
func (w *Writer) WriteRow(values ...interface{}) error {
	for i, val := range values {
		cell, err := excelize.CoordinatesToCellName(i+1, w.rowNum)
		if err != nil {
			return fmt.Errorf("failed to get cell name: %w", err)
		}
		if err := w.file.SetCellValue(w.sheetName, cell, val); err != nil {
			return fmt.Errorf("failed to set cell value: %w", err)
		}
	}
	w.rowNum++
	return nil
}

// WriteStruct 写入结构体数据（自动提取表头和数据）
func (w *Writer) WriteStruct(data interface{}) error {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return fmt.Errorf("data must be a struct, got %s", v.Kind())
	}

	t := v.Type()
	columns := getColumns(t, w.tagName)

	// 写入表头（只在第一行写入）
	if w.rowNum == 1 {
		headers := make([]string, len(columns))
		for i, col := range columns {
			headers[i] = col.Name
		}
		if err := w.WriteHeader(headers...); err != nil {
			return err
		}
	}

	// 写入数据
	values := make([]interface{}, len(columns))
	for i, col := range columns {
		fieldVal := v.FieldByIndex(col.Index)
		str, err := valueToString(fieldVal)
		if err != nil {
			return fmt.Errorf("failed to convert field %s: %w", col.Field.Name, err)
		}
		values[i] = str
	}

	return w.WriteRow(values...)
}

// WriteStructs 写入结构体切片
func (w *Writer) WriteStructs(data interface{}) error {
	v := reflect.ValueOf(data)
	if v.Kind() != reflect.Slice {
		return fmt.Errorf("data must be a slice, got %s", v.Kind())
	}

	// 写入表头
	if v.Len() > 0 {
		elem := v.Index(0)
		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}
		if elem.Kind() == reflect.Struct {
			t := elem.Type()
			columns := getColumns(t, w.tagName)
			headers := make([]string, len(columns))
			for i, col := range columns {
				headers[i] = col.Name
			}
			if err := w.WriteHeader(headers...); err != nil {
				return err
			}
		}
	}

	// 写入数据
	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i)
		if elem.Kind() == reflect.Ptr {
			if elem.IsNil() {
				continue
			}
			elem = elem.Elem()
		}
		if err := w.WriteStruct(elem.Interface()); err != nil {
			return fmt.Errorf("failed to write row %d: %w", i+1, err)
		}
	}

	return nil
}

// SetCellStyle 设置单元格样式
func (w *Writer) SetCellStyle(hCell, vCell string, style *excelize.Style) error {
	styleID, err := w.file.NewStyle(style)
	if err != nil {
		return fmt.Errorf("failed to create style: %w", err)
	}
	return w.file.SetCellStyle(w.sheetName, hCell, vCell, styleID)
}

// SetColWidth 设置列宽
func (w *Writer) SetColWidth(startCol, endCol string, width float64) error {
	return w.file.SetColWidth(w.sheetName, startCol, endCol, width)
}

// SetRowHeight 设置行高
func (w *Writer) SetRowHeight(row int, height float64) error {
	return w.file.SetRowHeight(w.sheetName, row, height)
}

// MergeCell 合并单元格
func (w *Writer) MergeCell(hCell, vCell string) error {
	return w.file.MergeCell(w.sheetName, hCell, vCell)
}

// SetCellValue 设置单个单元格的值
func (w *Writer) SetCellValue(cell string, value interface{}) error {
	return w.file.SetCellValue(w.sheetName, cell, value)
}

// SetCellFormula 设置单元格公式
func (w *Writer) SetCellFormula(cell, formula string) error {
	return w.file.SetCellFormula(w.sheetName, cell, formula)
}

// AddSheet 添加工作表
func (w *Writer) AddSheet(name string) error {
	_, err := w.file.NewSheet(name)
	return err
}

// SetActiveSheet 设置活动工作表
func (w *Writer) SetActiveSheet(name string) {
	idx, _ := w.file.GetSheetIndex(name)
	if idx != -1 {
		w.file.SetActiveSheet(idx)
		w.sheetName = name
	}
}

// WriteTo 写入到 io.Writer，实现 io.WriterTo 接口
func (w *Writer) WriteTo(writer io.Writer) (int64, error) {
	buf, err := w.file.WriteToBuffer()
	if err != nil {
		return 0, err
	}
	return buf.WriteTo(writer)
}

// SaveAs 保存到文件
func (w *Writer) SaveAs(filename string) error {
	return w.file.SaveAs(filename)
}

// Close 关闭写入器
func (w *Writer) Close() error {
	return w.file.Close()
}

// Bytes 返回字节数组
func (w *Writer) Bytes() ([]byte, error) {
	buf, err := w.file.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ==================== 便捷函数 ====================

// Export 导出数据到 Excel 文件
// data 必须是结构体切片或指向结构体切片的指针
func Export(data interface{}, filename string, opts ...WriterOption) error {
	w := NewWriter(opts...)
	defer w.Close()

	if err := w.WriteStructs(data); err != nil {
		return err
	}

	return w.SaveAs(filename)
}

// ExportToBytes 导出数据到字节数组
func ExportToBytes(data interface{}, opts ...WriterOption) ([]byte, error) {
	w := NewWriter(opts...)
	defer w.Close()

	if err := w.WriteStructs(data); err != nil {
		return nil, err
	}

	return w.Bytes()
}

// ExportToWriter 导出数据到 io.Writer
func ExportToWriter(data interface{}, writer io.Writer, opts ...WriterOption) error {
	w := NewWriter(opts...)
	defer w.Close()

	if err := w.WriteStructs(data); err != nil {
		return err
	}

	_, err := w.WriteTo(writer)
	return err
}
