package mtool

type CompareFunc func(any, any) int

/*
使用方法
a := []string{"a", "b", "c"}
IndexOf(a, "b", func(a interface{}, b interface{}) int {
	s1 := a.(string)
	s2 := b.(string)
	return strings.Compare(s1, s2)
}
*/

func IndexOf(list []any, i any, cmp CompareFunc) int {
	for a := 0; a < len(list); a++ {
		if cmp(i, list[a]) == 0 {
			return a
		}
	}
	return -1
}
