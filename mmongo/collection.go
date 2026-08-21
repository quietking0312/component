package mmongo

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Collection 包装了一个绑定到结构体类型 T 的 *mongo.Collection，
// 使得 EnsureIndexes 可以根据 T 上的索引 tag 推导出该集合的索引。
type Collection[T any] struct {
	*mongo.Collection
	indexTagName string
}

// namedCollection 让模型类型可以覆盖其默认的集合名。
type namedCollection interface {
	CollectionName() string
}

// collectionConfig 保存 NewCollection 的配置，通过 CollectionOption 填充。
// 把它保持在每个实例上（而不是像旧的 IndexTagName 那样用包级变量），
// 意味着设置不同的并发 Collection[T] 实例之间永远不会发生数据竞争。
type collectionConfig struct {
	name         string
	indexTagName string
}

// CollectionOption 配置通过 NewCollection 创建的 Collection[T]。
type CollectionOption func(cfg *collectionConfig)

// SetCollectionName 显式设置底层 MongoDB 集合名，优先级高于 CollectionName()
// 方法以及结构体类型名兜底方案。
func SetCollectionName(name string) CollectionOption {
	return func(cfg *collectionConfig) { cfg.name = name }
}

// SetIndexTagName 覆盖 EnsureIndexes 在该 Collection 实例上读取的结构体 tag
// 名（默认是 "mindex"）。当 "mindex" 与你模型上已经使用的 tag 名冲突时使用。
func SetIndexTagName(tagName string) CollectionOption {
	return func(cfg *collectionConfig) { cfg.indexTagName = tagName }
}

// NewCollection 为给定的 database 返回一个 Collection[T]。集合名按以下顺序解析：
//   - 如果传入了 SetCollectionName，直接使用；
//   - 否则如果 T（或 *T）实现了 CollectionName() string，使用其返回值；
//   - 否则使用 T 的结构体类型名（全小写）。
//
// T 必须是结构体类型或指向结构体的指针；Go 泛型没有语法能把这一点表达成
// 编译期约束，所以这里通过反射在运行时校验，并在 T 不满足条件时让
// NewCollection 立即 panic，而不是把失败推迟到第一次调用 EnsureIndexes 时。
func NewCollection[T any](db *mongo.Database, opts ...CollectionOption) *Collection[T] {
	if _, err := structTypeOf[T](); err != nil {
		panic(err)
	}

	cfg := &collectionConfig{indexTagName: defaultIndexTagName}
	for _, opt := range opts {
		opt(cfg)
	}

	return &Collection[T]{
		Collection:   db.Collection(nameFromConfig[T](cfg)),
		indexTagName: cfg.indexTagName,
	}
}

// nameFromConfig 按照 NewCollection 文档中同样的优先级从 cfg 解析出最终的
// 集合名：显式的 SetCollectionName 优先，否则回退到 resolveCollectionName[T]。
// Registry 复用了这个函数，从而保证查找时解析出的名字与 NewCollection 完全一致。
func nameFromConfig[T any](cfg *collectionConfig) string {
	if cfg.name != "" {
		return cfg.name
	}
	return resolveCollectionName[T]()
}

// structTypeOf 返回 T 底层的结构体 reflect.Type，T 本身必须是结构体或指向
// 结构体的指针。对其他任何 kind（例如 T = string、T = map[string]any）返回错误。
func structTypeOf[T any]() (reflect.Type, error) {
	typ := reflect.TypeOf((*T)(nil)).Elem()
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return nil, fmt.Errorf("mmongo: type parameter must be a struct or pointer to struct, got %s", typ)
	}
	return typ, nil
}

func resolveCollectionName[T any]() string {
	var zero T
	switch named := any(zero).(type) {
	case namedCollection:
		return named.CollectionName()
	}
	if named, ok := any(&zero).(namedCollection); ok {
		return named.CollectionName()
	}

	typ := reflect.TypeOf(zero)
	if typ == nil {
		typ = reflect.TypeOf((*T)(nil)).Elem()
	}
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	return strings.ToLower(typ.Name())
}

// EnsureIndexes 创建 T 上所有通过 `mindex` 结构体 tag 声明的索引。
// 每次启动都调用是安全的：当完全相同的索引已存在时 MongoDB 会直接忽略，
// 只有当同名索引的定义发生冲突时才会返回错误。
func (c *Collection[T]) EnsureIndexes(ctx context.Context) error {
	models, err := indexModelsFor[T](c.indexTagName)
	if err != nil {
		return err
	}
	if len(models) == 0 {
		return nil
	}
	_, err = c.Indexes().CreateMany(ctx, models)
	return err
}
