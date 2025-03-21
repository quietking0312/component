package mtool

import (
	"errors"
	"reflect"
	"strings"
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

type Opt struct {
	Tag string
}

// GetFieldsTagValueMap 结构体转map
func GetFieldsTagValueMap(v any, opts ...func(opt *Opt)) map[string]any {
	opt := &Opt{
		Tag: "json",
	}
	for _, o := range opts {
		o(opt)
	}

	r := reflect.ValueOf(v)
	if r.Kind() == reflect.Ptr {
		r = r.Elem()
	}
	refType := r.Type()
	ret := make(map[string]any)
	for i := 0; i < r.NumField(); i++ {
		tag := refType.Field(i).Tag.Get(opt.Tag)
		if tag == "-" {
			continue
		}
		value := r.Field(i).Interface()
		ret[tag] = value
	}
	return ret
}

func GetFilesTag(v any, opts ...func(opt *Opt)) []string {
	opt := &Opt{
		Tag: "json",
	}
	for _, o := range opts {
		o(opt)
	}
	r := reflect.ValueOf(v)
	if r.Kind() == reflect.Ptr {
		r = r.Elem()
	} else if r.Kind() != reflect.Struct {
		return nil
	}
	refType := r.Type()
	s := make([]string, 0)
	for i := 0; i < r.NumField(); i++ {
		tag := refType.Field(i).Tag.Get(opt.Tag)
		if tag == "-" {
			continue
		}
		t := strings.Split(tag, ",")[0]
		if t != "" {
			s = append(s, t)
		}

	}
	return s
}
