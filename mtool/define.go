package mtool

import (
	"runtime"
)

const (
	CapitalLetter = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	LowerLetter   = "abcdefghijklmnopqrstuvwxyz"
	Numbers       = "0123456789"
	Letters       = LowerLetter + CapitalLetter
	AlphaNumeric  = Numbers + Letters
)

// 获取协程调用函数
func runFuncName() (string, string, int) {
	pc := make([]uintptr, 1)
	runtime.Callers(3, pc)
	f := runtime.FuncForPC(pc[0])
	fileName, line := f.FileLine(pc[0])
	return fileName, f.Name(), line
}

func GetMapKeys[K comparable, V comparable](m map[K]V) []K {
	var keys = make([]K, 0)
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func GetMapValues[K comparable, V comparable](m map[K]V) []V {
	var values = make([]V, 0)
	for _, v := range m {
		values = append(values, v)
	}
	return values
}

/*
IndexOf 使用方法
a := []string{"a", "b", "c"}

	IndexOf(a, "b", func(a interface{}, b interface{}) int {
		s1 := a.(string)
		s2 := b.(string)
		return strings.Compare(s1, s2)
	}
*/
func IndexOf[T comparable](list []T, i T) int {
	for a := 0; a < len(list); a++ {
		if i == list[a] {
			return a
		}
	}
	return -1
}

func SliceSplit[T comparable](slice []T, n int) [][]T {
	g := make([][]T, 0)
	if n < 1 {
		return g
	}
	sliceLen := len(slice)
	for i := 0; i < sliceLen; i += n {
		end := i + n
		if end > sliceLen {
			end = sliceLen
		}
		g = append(g, slice[i:end])
	}
	return g
}

func SliceGetKeys[T any, X any](slice []T, fc func(T) (X, bool)) []X {
	l := make([]X, 0)
	for _, v := range slice {
		x, ok := fc(v)
		if ok {
			l = append(l, x)
		}
	}
	return l
}
