package mcachedb

import (
	"encoding/json"
	"reflect"
	"time"
)

// BaseEntity 基础实体实现
type BaseEntity struct {
	ID         string    `json:"id" db:"id"`
	Ver        int64     `json:"ver" db:"ver"`
	DelFlag    bool      `json:"del_flag" db:"del_flag"`
	CreateTime time.Time `json:"create_time" db:"create_time"`
	UpdateTime time.Time `json:"update_time" db:"update_time"`
}

// NewBaseEntity 创建基础实体
func NewBaseEntity(id string) *BaseEntity {
	now := time.Now()
	return &BaseEntity{
		ID:         id,
		Ver:        1,
		DelFlag:    false,
		CreateTime: now,
		UpdateTime: now,
	}
}

// CacheKey 返回缓存键
func (e *BaseEntity) CacheKey() string {
	return e.ID
}

// IsDeleted 是否已删除
func (e *BaseEntity) IsDeleted() bool {
	return e.DelFlag
}

// SetDeleted 设置删除标记
func (e *BaseEntity) SetDeleted(deleted bool) {
	e.DelFlag = deleted
}

// Version 获取版本号
func (e *BaseEntity) Version() int64 {
	return e.Ver
}

// IncrementVersion 增加版本号
func (e *BaseEntity) IncrementVersion() {
	e.Ver++
	e.UpdateTime = time.Now()
}

// Copy 复制（深拷贝）
func (e *BaseEntity) Copy() *BaseEntity {
	return &BaseEntity{
		ID:         e.ID,
		Ver:        e.Ver,
		DelFlag:    e.DelFlag,
		CreateTime: e.CreateTime,
		UpdateTime: e.UpdateTime,
	}
}

// 注意：BaseEntity 不再提供 Marshal/Unmarshal 的默认实现。
// 具体实体类型必须自己实现这两个方法，否则会因为方法提升而继承到错误的实现，
// 导致只序列化基础字段、丢失业务字段。例如：
//
//	func (u *User) Marshal() ([]byte, error) { return json.Marshal(u) }
//	func (u *User) Unmarshal(b []byte) error { return json.Unmarshal(b, u) }

// EntityWrapper 实体包装器，用于将任意结构体转为 Entity
type EntityWrapper struct {
	BaseEntity
	Data interface{} `json:"data"`
}

// NewEntityWrapper 创建实体包装器
func NewEntityWrapper(key string, data interface{}) *EntityWrapper {
	return &EntityWrapper{
		BaseEntity: *NewBaseEntity(key),
		Data:       data,
	}
}

// Copy 复制
func (e *EntityWrapper) Copy() Entity {
	return &EntityWrapper{
		BaseEntity: *e.BaseEntity.Copy(),
		Data:       deepCopy(e.Data),
	}
}

// Marshal JSON 序列化整个 wrapper
func (e *EntityWrapper) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

// Unmarshal JSON 反序列化整个 wrapper
func (e *EntityWrapper) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}

// deepCopy 深拷贝
func deepCopy(src interface{}) interface{} {
	if src == nil {
		return nil
	}

	data, err := json.Marshal(src)
	if err != nil {
		return src
	}

	dst := reflect.New(reflect.TypeOf(src)).Interface()
	if err := json.Unmarshal(data, dst); err != nil {
		return src
	}

	return reflect.ValueOf(dst).Elem().Interface()
}

// entityToJSON 实体转JSON
func entityToJSON(entity Entity) ([]byte, error) {
	return json.Marshal(entity)
}

// entityFromJSON JSON转实体
func entityFromJSON(data []byte, entity Entity) error {
	return json.Unmarshal(data, entity)
}

// GenericEntity 泛型实体
type GenericEntity[T any] struct {
	BaseEntity
	Payload T `json:"payload"`
}

// NewGenericEntity 创建泛型实体
func NewGenericEntity[T any](key string, payload T) *GenericEntity[T] {
	return &GenericEntity[T]{
		BaseEntity: *NewBaseEntity(key),
		Payload:    payload,
	}
}

// Copy 复制泛型实体
func (e *GenericEntity[T]) Copy() Entity {
	var payloadCopy T
	data, _ := json.Marshal(e.Payload)
	_ = json.Unmarshal(data, &payloadCopy)

	return &GenericEntity[T]{
		BaseEntity: *e.BaseEntity.Copy(),
		Payload:    payloadCopy,
	}
}

// Marshal JSON 序列化完整泛型实体
func (e *GenericEntity[T]) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

// Unmarshal JSON 反序列化完整泛型实体
func (e *GenericEntity[T]) Unmarshal(data []byte) error {
	return json.Unmarshal(data, e)
}

// 确保实现 Entity 接口
var _ Entity = (*EntityWrapper)(nil)
