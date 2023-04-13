package mtool

import (
	"errors"
	"reflect"
)

func CopyStruct(src any, dst any) error {
	srcValue := reflect.ValueOf(src)
	srcType := srcValue.Type()
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() != reflect.Ptr {
		return errors.New("dst must be a pointer")
	}
	dstElem := dstValue.Elem()
	if srcType.Kind() != reflect.Struct || dstElem.Kind() != reflect.Struct {
		return errors.New("src and dst must be struct")
	}
	for i := 0; i < srcType.NumField(); i++ {
		srcField := srcValue.Field(i)
		dstField := dstElem.FieldByName(srcType.Field(i).Name)
		if dstField.IsValid() && dstField.CanSet() {
			if srcField.Type() == dstField.Type() {
				dstField.Set(srcField)
			}
		}
	}
	return nil
}

type Options func(srcFiled, dstFiled reflect.StructField) bool

func CopyStruct2(src any, dst any, opts ...Options) error {
	srcValue := reflect.ValueOf(src)
	srcType := srcValue.Type()
	dstValue := reflect.ValueOf(dst)
	dstType := dstValue.Type()
	if dstValue.Kind() != reflect.Ptr {
		return errors.New("dst must be a pointer")
	}
	dstElem := dstValue.Elem()
	if srcType.Kind() != reflect.Struct || dstElem.Kind() != reflect.Struct {
		return errors.New("src and dst must be struct")
	}
	for i := 0; i < srcType.NumField(); i++ {
		srcField := srcValue.Field(i)
		for j := 0; j < dstElem.NumField(); j++ {
			dstField := dstElem.Field(j)
			if dstField.IsValid() && dstField.CanSet() {
				isOK := false
				if len(opts) > 0 {
					for _, o := range opts {
						isOK = o(srcType.Field(i), dstType.Elem().Field(j))
					}
				} else {
					isOK = srcType.Field(i).Name == dstType.Elem().Field(j).Name && srcField.Type() == dstField.Type()
				}

				if isOK {
					dstField.Set(srcField)
				}
			}

		}

	}
	return nil
}

type A struct {
	Name string `json:"name"`
	Age  int
}

type B struct {
	Name  string `json:"name"`
	Age   int
	Phone int
}
