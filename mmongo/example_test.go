package mmongo

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/v2/bson"
	"time"
)

// Address 是 User.Address 的嵌套结构，整体以子文档形式存入 "address" 字段。
type Address struct {
	City   string `bson:"city"`
	Street string `bson:"street"`
}

// User 演示 mindex tag：email 唯一索引，bio 加入全文索引，
// tenant+name 组成复合索引，created_at 是 1 小时后过期的 TTL 索引。
type User struct {
	ID        string    `bson:"_id,omitempty"`
	Tenant    string    `bson:"tenant" mindex:"group=tenant_name,seq=1"`
	Name      string    `bson:"name" mindex:"group=tenant_name,seq=2"`
	Email     string    `bson:"email" mindex:"unique"`
	Bio       string    `bson:"bio" mindex:"text"`
	Address   Address   `bson:"address"`
	CreatedAt time.Time `bson:"created_at" mindex:"expire=3600"`
}

// CollectionName 显式指定集合名，否则会退化为结构体名全小写（"user"）。
func (User) CollectionName() string { return "users" }

// Example 演示了 mmongo 的完整用法：建立连接、使用操作符常量拼装更新语句、
// 通过 Collection[T] 按 mindex tag 建索引，并调用官方驱动方法读写数据。
//
// 本示例未附带 "// Output:" 注释，因此 go test 只会编译它，不会连接真实
// MongoDB 执行；如需运行，请确保本机有可访问的 MongoDB 实例。
func Example() {
	client, err := NewClient(
		SetHosts([]string{"127.0.0.1:27017"}),
		SetAuth("user", "password"),
		SetDatabase("mydb"),
	)
	if err != nil {
		panic(err)
	}
	defer client.Close(context.Background())

	ctx := context.Background()

	// Collection[T] 会通过 User.CollectionName() 解析出集合名 "users"，
	// EnsureIndexes 按 mindex tag 批量创建索引，重复调用是安全的。
	users := NewCollection[User](client.Database())
	if err := users.EnsureIndexes(ctx); err != nil {
		panic(err)
	}

	// users.Collection 就是官方 *mongo.Collection，直接用官方 API 增删改查。
	// InsertOne 直接写入嵌套结构体，Address 会作为子文档存入 "address" 字段。
	_, err = users.InsertOne(ctx, User{
		Tenant:  "acme",
		Name:    "alice",
		Email:   "alice@example.com",
		Bio:     "MongoDB enthusiast",
		Address: Address{City: "Hangzhou", Street: "Wangjiang Rd"},
	})
	if err != nil {
		panic(err)
	}

	// 使用操作符常量代替手写 "$set"，避免拼错字母。
	_, err = users.UpdateOne(ctx,
		bson.M{"email": "alice@example.com"},
		bson.M{OpSet: bson.M{"name": "Alice"}},
	)
	if err != nil {
		panic(err)
	}

	// 用点号路径 "address.city" 只更新嵌套结构体里的单个字段，
	// 不会覆盖 Address 的其它字段（如 street）。
	_, err = users.UpdateOne(ctx,
		bson.M{"email": "alice@example.com"},
		bson.M{OpSet: bson.M{"address.city": "Shanghai"}},
	)
	if err != nil {
		panic(err)
	}

	var result User
	err = users.FindOne(ctx, bson.M{"email": "alice@example.com"}).Decode(&result)
	if err != nil {
		panic(err)
	}
	fmt.Println(result.Name, result.Address.City)

	// 删除文档：DeleteOne 按过滤条件删除单条记录。
	_, err = users.DeleteOne(ctx, bson.M{"email": "alice@example.com"})
	if err != nil {
		panic(err)
	}
}
