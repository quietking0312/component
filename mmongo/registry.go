package mmongo

import (
	"fmt"
	"sync"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Registry 按 Collection[T] 解析出的 MongoDB 集合名（与 Collection.Name()
// 返回的名字一致）缓存 Collection[T] 实例，这样启动之后的重复查找可以跳过
// 重新解析名字 / 重新用反射校验 T 的过程。用名字而不是 T 做 key，也意味着
// 两个本来就映射到同一个物理集合的模型类型会共享同一个缓存项，而不是
// 悄悄产生重复项。调用方自己持有 *Registry（不是包级单例），需要自行
// 决定如何在应用内共享/注入。注册预期只在启动时发生一次，之后是大量的
// 并发读取，这正是 sync.Map 优化的访问模式。
type Registry struct {
	items sync.Map // collection name (string) -> any (*Collection[T])
}

// NewRegistry 返回一个空的、可直接使用的 Registry。
func NewRegistry() *Registry {
	return &Registry{}
}

// RegisterCollection 通过 NewCollection 创建一个 Collection[T]，并以其解析出
// 的集合名（即返回值 coll.Name() 报告的那个字符串）为 key 存入 r。
// 通常在启动阶段每个模型调用一次；用同一个名字重复注册会覆盖之前的条目。
func RegisterCollection[T any](r *Registry, db *mongo.Database, opts ...CollectionOption) *Collection[T] {
	coll := NewCollection[T](db, opts...)
	r.items.Store(coll.Name(), coll)
	return coll
}

// CollectionFrom 返回之前通过 RegisterCollection 存入的 Collection[T]；
// 如果解析出的名字下没有注册任何内容，返回 false。如果注册时用
// SetCollectionName 之类的 CollectionOption 覆盖了集合名，这里也要传入
// 同样的 CollectionOption，才能解析出一致的名字。
func CollectionFrom[T any](r *Registry, opts ...CollectionOption) (*Collection[T], bool) {
	cfg := &collectionConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	v, ok := r.items.Load(nameFromConfig[T](cfg))
	if !ok {
		return nil, false
	}
	coll, ok := v.(*Collection[T])
	return coll, ok
}

// MustCollectionFrom 与 CollectionFrom 类似，但在解析出的名字下没有任何注册
// 时会 panic——未注册属于启动期 bug，不应该要求业务代码在每个调用点都处理。
func MustCollectionFrom[T any](r *Registry, opts ...CollectionOption) *Collection[T] {
	coll, ok := CollectionFrom[T](r, opts...)
	if !ok {
		cfg := &collectionConfig{}
		for _, opt := range opts {
			opt(cfg)
		}
		panic(fmt.Sprintf("mmongo: no Collection registered under name %q; call RegisterCollection first", nameFromConfig[T](cfg)))
	}
	return coll
}
