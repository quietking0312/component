package mtool

type MList[K comparable, T any] []T

func (m MList[K, T]) Map(key func(T) K) map[K]T {
	result := make(map[K]T, len(m))
	for _, v := range m {
		result[key(v)] = v
	}
	return result
}
