package mtool

import (
	"runtime"
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
