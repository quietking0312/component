# mmongo - MongoDB 客户端组件

基于官方 `go.mongodb.org/mongo-driver/v2`（v2 驱动）封装的 MongoDB 客户端连接层，仅负责连接创建、配置和生命周期管理，业务侧直接使用官方 `*mongo.Database` / `*mongo.Collection` API。

## 特性

- ⚙️ **灵活配置**：通过 Option 模式配置地址、认证、连接池、超时等参数
- 🔗 **URI / 参数两种连接方式**：可直接使用连接串，也可用离散参数拼装
- 🔐 **TLS 支持**：支持 TLS 加密连接
- 🧩 **零额外封装成本**：`Client` 直接嵌入官方 `*mongo.Client`，所有官方方法（`StartSession`、`Watch` 等）均可直接调用
- 🔤 **操作符常量**：内置 `$set`/`$push`/`$match` 等 BSON 操作符常量，避免手写字符串拼错字母
- 🏷️ **按 tag 建索引**：`Collection[T]` 可根据结构体 `mindex` tag 自动创建单字段/唯一/复合/TTL/文本索引
- 🗂️ **Collection 缓存**：`Registry` 可在启动时批量注册 `Collection[T]`，业务代码按类型直接取用，避免重复解析集合名

## 快速开始

```go
package main

import (
    "context"
    "fmt"

    "github.com/quietking0312/component/mmongo"
    "go.mongodb.org/mongo-driver/v2/bson"
)

func main() {
    client, err := mmongo.NewClient(
        mmongo.SetHosts([]string{"127.0.0.1:27017"}),
        mmongo.SetAuth("user", "password"),
        mmongo.SetDatabase("mydb"),
    )
    if err != nil {
        panic(err)
    }
    defer client.Close(context.Background())

    coll := client.Database().Collection("users")
    _, err = coll.InsertOne(context.Background(), bson.M{"name": "alice"})
    if err != nil {
        panic(err)
    }

    // 使用操作符常量代替手写 "$set"，避免拼错
    _, err = coll.UpdateOne(context.Background(),
        bson.M{"name": "alice"},
        bson.M{mmongo.OpSet: bson.M{"age": 18}},
    )
    if err != nil {
        panic(err)
    }

    var result bson.M
    err = coll.FindOne(context.Background(), bson.M{"name": "alice"}).Decode(&result)
    if err != nil {
        panic(err)
    }
    fmt.Println(result)
}
```

### 使用连接串

```go
client, err := mmongo.NewClient(
    mmongo.SetURI("mongodb://user:password@host1:27017,host2:27017/?replicaSet=rs0"),
    mmongo.SetDatabase("mydb"),
)
```

## API 文档

### 配置选项

| 函数 | 说明 |
|------|------|
| `SetURI(uri string) Option` | 设置完整连接串，设置后忽略 Hosts/Auth/AuthSource/ReplicaSet |
| `SetHosts(hosts []string) Option` | 设置服务地址列表（未设置 URI 时使用） |
| `SetAuth(username, password string) Option` | 设置认证用户名密码 |
| `SetAuthSource(authSource string) Option` | 设置认证数据库，默认 `admin` |
| `SetDatabase(database string) Option` | 设置默认数据库名，`Database()` 不传参时使用 |
| `SetReplicaSet(replicaSet string) Option` | 设置副本集名称 |
| `SetMaxPoolSize(size uint64) Option` | 设置连接池最大连接数 |
| `SetMinPoolSize(size uint64) Option` | 设置连接池最小连接数 |
| `SetConnectTimeout(d time.Duration) Option` | 设置连接超时，同时用作 `NewClient` 的 Ping 超时 |
| `SetTimeout(d time.Duration) Option` | 设置单次操作的整体超时（v2 驱动统一超时机制） |
| `SetServerSelectionTimeout(d time.Duration) Option` | 设置服务节点选择超时 |
| `SetTLSConfig(cfg *tls.Config) Option` | 设置 TLS 配置 |

### 客户端

| 方法 | 说明 |
|------|------|
| `NewClient(opts ...Option) (*Client, error)` | 创建客户端，内部会执行 Ping 检测连通性 |
| `client.Database(name ...string) *mongo.Database` | 获取数据库，不传参时使用 `SetDatabase` 配置的默认库 |
| `client.Close(ctx context.Context) error` | 断开连接，释放连接池资源 |

`Client` 直接嵌入官方 `*mongo.Client`，因此 `StartSession`、`Watch`、`Ping` 等官方方法均可直接在 `client` 上调用。

### 操作符常量

[operator.go](operator.go) 提供了常用 BSON 操作符常量，统一使用 `Op` 前缀，业务代码写更新/查询/聚合语句时用常量替代手写字符串：

| 分类 | 常量示例 |
|------|------|
| Update | `OpSet`、`OpUnset`、`OpInc`、`OpPush`、`OpPull`、`OpAddToSet`、`OpEach`、`OpPosition`、`OpSlice` 等 |
| Query/比较 | `OpEq`、`OpNe`、`OpGt`、`OpGte`、`OpLt`、`OpLte`、`OpIn`、`OpNin`、`OpExists`、`OpRegex`、`OpElemMatch` 等 |
| 逻辑 | `OpAnd`、`OpOr`、`OpNot`、`OpNor` |
| 聚合管道 | `OpMatch`、`OpGroup`、`OpProject`、`OpSort`、`OpLookup`、`OpUnwind`、`OpFacet`、`OpBucket` 等 |

### 按 tag 自动建索引

[collection.go](collection.go) 提供了 `Collection[T]`，将 `*mongo.Collection` 绑定到一个结构体类型 `T`，`EnsureIndexes` 会读取 `T` 上的 `mindex` tag 并批量创建索引。`mindex` 的 tag 语法见 [index_tag.go](index_tag.go)：

| tag 写法 | 说明 |
|------|------|
| `mindex:"index"` | 单字段升序索引 |
| `mindex:"unique"` | 单字段唯一索引（隐含 index） |
| `mindex:"index,desc"` | 单字段降序索引 |
| `mindex:"index,sparse"` | 稀疏索引 |
| `mindex:"text"` | 加入全文索引（多个 text 字段会合并成同一个文本索引） |
| `mindex:"expire=3600"` | TTL 索引，`expireAfterSeconds=3600`（字段需为日期/时间戳） |
| `mindex:"group=name,seq=1"` | 加入名为 `name` 的复合索引，按 `seq` 升序排列字段顺序，可再叠加 `unique`/`desc`/`sparse` |

字段的文档键名始终读官方 `bson` tag（不受影响）；索引描述用的 tag 名默认是 `mindex`，如果与项目里已有的 tag 冲突，可以在创建 `Collection[T]` 时用 `SetIndexTagName` 按实例覆盖（tag 名保存在 `Collection[T]` 实例上，而不是包级变量，不同模型/并发调用之间互不影响，不存在数据竞争）：

```go
coll := mmongo.NewCollection[User](client.Database(), mmongo.SetIndexTagName("index"))
// 之后 coll.EnsureIndexes 只识别 `index:"unique"` 这种写法
```

```go
type User struct {
    ID        string `bson:"_id,omitempty"`
    Email     string `bson:"email" mindex:"unique"`
    Bio       string `bson:"bio" mindex:"text"`
    Tenant    string `bson:"tenant" mindex:"group=tenant_name,seq=1"`
    Name      string `bson:"name" mindex:"group=tenant_name,seq=2"`
    CreatedAt int64  `bson:"created_at" mindex:"expire=3600"`
}

// 不传 name：User 没有 CollectionName() 方法，落到集合名 "user"（结构体名全小写）
coll := mmongo.NewCollection[User](client.Database())
if err := coll.EnsureIndexes(context.Background()); err != nil {
    panic(err)
}
// coll.Collection 就是官方 *mongo.Collection，可直接用于增删改查
```

`EnsureIndexes` 内部调用 `CreateMany`，重复调用是安全的：索引已存在且定义相同时 MongoDB 会直接忽略；若同名索引的定义发生冲突，会返回错误。

`NewCollection[T]` 的集合名解析顺序：

1. 显式传入 `SetCollectionName`（如 `mmongo.NewCollection[User](db, mmongo.SetCollectionName("users"))`）；
2. `T`（或 `*T`）实现了 `CollectionName() string` 方法，使用其返回值；
3. 否则使用结构体类型名全小写（如 `User` → `"user"`）。

```go
type Order struct {
    ID string `bson:"_id,omitempty"`
}

func (Order) CollectionName() string { return "orders" }

coll := mmongo.NewCollection[Order](client.Database()) // 使用 "orders"
```

### 缓存 Collection（Registry）

每次调用 `NewCollection[T]` 都会做一遍反射（类型校验 + 集合名解析）。如果模型类型数量较多，可以用 `Registry` 在启动时一次性注册所有 `Collection[T]`，业务代码使用时直接按类型取用，不必重复解析：

```go
registry := mmongo.NewRegistry()
mmongo.RegisterCollection[User](registry, client.Database())
mmongo.RegisterCollection[Order](registry, client.Database())

// 业务代码里直接取用
users := mmongo.MustCollectionFrom[User](registry)
orders, ok := mmongo.CollectionFrom[Order](registry)
```

- `RegisterCollection[T](r, db, opts...)`：等价于 `NewCollection[T](db, opts...)` 再存入 `r`，缓存 key 是解析出来的**集合名字符串**（就是 `coll.Name()` 返回的那个名字），不是 `T` 的类型。
- `CollectionFrom[T](r, opts...)`：按同样的规则（`SetCollectionName` 优先，否则 `CollectionName()` 方法，否则结构体名全小写）重新解析出集合名，再去缓存里取。默认情况下只传结构体类型就能拿到正确的 key，不需要手写字符串，也就不存在拼错的问题；如果注册时用了 `SetCollectionName` 覆盖了名字，取用时也要传同样的 `SetCollectionName`，否则解析出的默认名对不上会返回 `(nil, false)`。
- `MustCollectionFrom[T](r, opts...)`：取不到时直接 panic，适合"未注册即视为启动期 bug"的场景。

因为 key 是集合名而不是 Go 类型，`User` 和 `*User` 如果都用默认命名规则，会解析成同一个名字（`"user"`），后注册的会覆盖前一个——这与实际情况是一致的：它们本来就代表同一个 MongoDB 集合。

`Registry` 内部用 `sync.Map` 存储（注册通常只发生一次、之后大量并发读取，正是 `sync.Map` 的设计场景），是调用方自己创建、持有的实例，不是包级全局变量，多个 `Registry` 实例之间互不影响，也不存在并发访问下的数据竞争。

## 测试

```bash
cd mmongo
go test -v
```

依赖本地可访问的 MongoDB 实例（默认 `127.0.0.1:27017`）。

## 注意事项

1. 客户端创建成功后会执行 Ping 检测，请确保网络可达
2. `SetURI` 与离散参数（`SetHosts`/`SetAuth`/`SetAuthSource`/`SetReplicaSet`）互斥，设置 URI 后离散参数会被忽略
3. 生产环境建议开启 TLS 和认证
4. 使用完毕后务必调用 `Close` 释放连接池资源
