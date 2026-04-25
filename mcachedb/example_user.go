package mcachedb

import (
	"fmt"
)

// User 用户实体示例
type User struct {
	BaseEntity
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Age      int      `json:"age"`
	Tags     []string `json:"tags"`
}

// NewUser 创建用户
func NewUser(id, username, email string, age int) *User {
	return &User{
		BaseEntity: *NewBaseEntity(id),
		Username:   username,
		Email:      email,
		Age:        age,
		Tags:       make([]string, 0),
	}
}

// Copy 深拷贝
func (u *User) Copy() Entity {
	return &User{
		BaseEntity: *u.BaseEntity.Copy().(*BaseEntity),
		Username:   u.Username,
		Email:      u.Email,
		Age:        u.Age,
		Tags:       append([]string{}, u.Tags...),
	}
}

// Order 订单实体示例
type Order struct {
	BaseEntity
	UserID    string      `json:"user_id"`
	ProductID string      `json:"product_id"`
	Amount    float64     `json:"amount"`
	Status    string      `json:"status"`
	Items     []OrderItem `json:"items"`
}

// OrderItem 订单项
type OrderItem struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	Price     float64 `json:"price"`
}

// NewOrder 创建订单
func NewOrder(id, userID, productID string, amount float64) *Order {
	return &Order{
		BaseEntity: *NewBaseEntity(id),
		UserID:     userID,
		ProductID:  productID,
		Amount:     amount,
		Status:     "pending",
		Items:      make([]OrderItem, 0),
	}
}

// Copy 深拷贝
func (o *Order) Copy() Entity {
	items := make([]OrderItem, len(o.Items))
	copy(items, o.Items)

	return &Order{
		BaseEntity: *o.BaseEntity.Copy().(*BaseEntity),
		UserID:     o.UserID,
		ProductID:  o.ProductID,
		Amount:     o.Amount,
		Status:     o.Status,
		Items:      items,
	}
}

// CacheWithUser 带有用户实体类型的缓存
func NewUserCache(store DBStore, opts ...Option) (*Cache, error) {
	return New(store, opts...)
}

// UserService 用户服务示例
type UserService struct {
	cache *Cache
}

// NewUserService 创建用户服务
func NewUserService(cache *Cache) *UserService {
	return &UserService{cache: cache}
}

// GetUser 获取用户
func (s *UserService) GetUser(userID string) (*User, error) {
	entity, err := s.cache.Get(userID)
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, nil
	}

	user, ok := entity.(*User)
	if !ok {
		return nil, fmt.Errorf("invalid user type")
	}
	return user, nil
}

// SaveUser 保存用户
func (s *UserService) SaveUser(user *User) error {
	return s.cache.Set(user)
}

// DeleteUser 删除用户
func (s *UserService) DeleteUser(userID string) error {
	return s.cache.Delete(userID)
}

// BatchUpdateUsers 批量更新用户
func (s *UserService) BatchUpdateUsers(users []*User) error {
	pipe := s.cache.Pipeline()
	for _, user := range users {
		pipe.Set(user)
	}
	return pipe.Exec()
}
