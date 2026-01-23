package mpubsub

import "sync"

// 基础的数据组
// 可以根据实际情况， 使用各种数据类型
var _ GroupIface[any] = (*Group[any])(nil)

type Group[T any] struct {
	data sync.Map
}

func (g *Group[T]) Set(h HandlerIface[T]) {
	g.data.Store(h.ID(), h)
}

func (g *Group[T]) GetSubList(t T) []HandlerIface[T] {
	v := make([]HandlerIface[T], 0)
	g.data.Range(func(key, value any) bool {
		val, ok := value.(HandlerIface[T])
		if ok {
			v = append(v, val)
		}
		return true
	})
	return v
}

func (g *Group[T]) Delete(key string) {
	g.data.Delete(key)
}
