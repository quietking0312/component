# mcachedb - 带批量持久化的多级缓存

`mcachedb` 是一个高性能的多级缓存组件，支持 L1(内存) -> L2(Redis) -> L3(数据库) 三级缓存架构，有效防止服务重启导致的数据丢失。

## 三级缓存架构

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │
       ▼
┌─────────────┐     命中     ┌─────────────┐
│    L1       │─────────────>│   Handler   │
│  (Memory)   │              └─────────────┘
└──────┬──────┘
       │ 未命中
       ▼
┌─────────────┐     命中     ┌─────────────┐
│    L2       │─────────────>│   Handler   │
│   (Redis)   │   + 回填L1   └─────────────┘
└──────┬──────┘
       │ 未命中
       ▼
┌─────────────┐     命中     ┌─────────────┐
│    L3       │─────────────>│   Handler   │
│    (DB)     │   + 回填L1/L2└─────────────┘
└─────────────┘
```

## 数据流转

### 写入流程
```
Handler -> Set -> L1(内存) -> 异步 -> L2(Redis) -> 异步批量 -> L3(数据库)
                (立即返回)    (500ms)              (1s/100条)
```

### 读取流程
```
Handler -> Get -> L1? -> L2? -> L3?
                  ↓      ↓      ↓
                命中   回填L1  回填L1/L2
```

### 服务重启保护
```
重启前: L1(内存) + L2(Redis) + L3(数据库)
         ↓ 重启丢失
重启后:        L2(Redis) + L3(数据库)
                ↓ 自动恢复
              回填 L1
```

## 快速开始

### 基础用法（三级缓存）

```go
import "go_admin_element/admin_server/component/mcachedb"

// 1. 定义实体
type User struct {
    mcachedb.BaseEntity
    Username string `json:"username"`
    Email    string `json:"email"`
}

func (u *User) Copy() mcachedb.Entity {
    return &User{
        BaseEntity: *u.BaseEntity.Copy().(*mcachedb.BaseEntity),
        Username:   u.Username,
        Email:      u.Email,
    }
}

// 2. 创建多级缓存
dbStore, _ := mcachedb.NewSQLXStore(dsn, &User{})

config := &mcachedb.MultiCacheConfig{
    L1MaxSize:      100000,              // L1最大条目数
    L2RedisConfig: &mcachedb.RedisConfig{
        Addr:      "localhost:6379",
        KeyPrefix: "myapp:",
    },
    SyncInterval:   500 * time.Millisecond, // L1->L2同步间隔
    FlushInterval:  1 * time.Second,        // L1->L3刷盘间隔
}

cache, _ := mcachedb.NewMultiCache(dbStore, config)
defer cache.Close()

// 3. Handler中使用
func (h *Handler) UpdateUser(c *gin.Context) {
    // 读：先L1，再L2，再L3，逐级回填
    entity, _ := h.cache.Get(userID)
    user := entity.(*User)
    
    // 修改
    user.Username = newName
    
    // 写：写L1，后台同步L2，批量刷L3
    h.cache.Set(user)
    
    // 立即返回
    c.JSON(200, Success)
}
```

## 配置说明

### MultiCacheConfig

| 参数 | 默认值 | 说明 |
|-----|-------|------|
| L1MaxSize | 100000 | L1内存缓存最大条目数 |
| L1CleanupInterval | 5m | L1过期数据清理间隔 |
| L2RedisConfig | 默认配置 | Redis配置 |
| WriteToL2OnSet | false | 写入时是否同步写入L2 |
| SyncInterval | 500ms | L1->L2同步间隔 |
| FlushInterval | 1s | L1->L3刷盘间隔 |
| L2Downgrade | true | L2故障时是否降级 |
| WriteMode | WriteModeAsync | L1写入模式：异步/同步/直写/缓存旁路 |
| L2EntityType | nil | L2反序列化实体原型；配置后可保持Redis中的具体类型 |

## 数据一致性保证

### 1. 三级缓存一致性

```go
// 写入时
L1.Set(entity)           // 立即写入
↓ 500ms
L2.Set(entity)           // 后台同步
↓ 1s / 100条
L3.Insert/Update(entity) // 批量刷盘
```

### 2. 服务重启恢复

```go
// 重启后首次读取
cache.Get("user:123")
// 1. L1未命中
// 2. L2命中 -> 回填L1 -> 返回
// 3. 如果L2也未命中 -> 查L3 -> 回填L1和L2 -> 返回
```

### 3. Redis故障降级

```go
// L2故障时自动降级
if mc.isL2Down() {
    // 跳过L2，直接L1 <-> L3
    // 定期重试连接L2
}
```

## 使用模式

### 模式1：标准模式（推荐）

```go
config := &mcachedb.MultiCacheConfig{
    SyncInterval:  500 * time.Millisecond,  // L1->L2 500ms
    FlushInterval: 1 * time.Second,          // L1->L3 1s
}
// 最多丢失 1s 数据
```

### 模式2：高可靠模式

```go
config := &mcachedb.MultiCacheConfig{
    WriteToL2OnSet: true,  // 同步写入L2
    FlushInterval:  100 * time.Millisecond,  // 快速刷盘
}
// 最多丢失 100ms 数据，性能略有下降
```

### 模式3：纯内存+数据库（无Redis）

```go
config := &mcachedb.MultiCacheConfig{
    L2RedisConfig: nil,  // 不配置Redis
}
// 重启丢失数据，但最简单
```

### 模式4：先写数据库后删缓存（强一致）

```go
config := &mcachedb.MultiCacheConfig{
    WriteMode: mcachedb.WriteModeCacheAside,  // 先写数据库，成功后删除缓存
}
// 写：先写 L3 数据库，成功后删除 L1/L2 缓存
// 读：未命中时从 L3 回填 L1/L2
// 一致性最好，但写入延迟取决于数据库
```

## 在项目中集成

### Service 层改造

```go
// user_service.go
type UserService struct {
    cache *mcachedb.MultiCache
}

func NewUserService(db *sqlx.DB, redisAddr string) (*UserService, error) {
    // 数据库存储
    dbStore, err := mcachedb.NewSQLXStoreFromDB(db, &User{})
    if err != nil {
        return nil, err
    }
    
    // 多级缓存
    cache, err := mcachedb.NewMultiCache(dbStore, &mcachedb.MultiCacheConfig{
        L2RedisConfig: &mcachedb.RedisConfig{
            Addr: redisAddr,
        },
        L2EntityType: &User{},  // 保持 Redis 中的类型，避免 L2 命中后类型丢失
    })
    if err != nil {
        return nil, err
    }
    
    return &UserService{cache: cache}, nil
}

func (s *UserService) GetUser(id string) (*User, error) {
    entity, err := s.cache.Get(id)
    if err != nil || entity == nil {
        return nil, err
    }
    return entity.(*User), nil
}

func (s *UserService) UpdateUser(user *User) error {
    return s.cache.Set(user)
}

func (s *UserService) DeleteUser(id string) error {
    return s.cache.Delete(id)
}
```

### Handler 层使用

```go
// user_handler.go
func (h *UserHandler) GetUser(c *gin.Context) {
    id := c.Param("id")
    
    user, err := h.service.GetUser(id)
    if err != nil {
        c.JSON(500, Error(err))
        return
    }
    if user == nil {
        c.JSON(404, NotFound())
        return
    }
    
    c.JSON(200, OK(user))
}

func (h *UserHandler) UpdateUser(c *gin.Context) {
    var req UpdateUserReq
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, Error(err))
        return
    }
    
    // 获取现有数据
    user, _ := h.service.GetUser(req.ID)
    if user == nil {
        c.JSON(404, NotFound())
        return
    }
    
    // 修改
    user.Username = req.Username
    
    // 保存（异步写入，立即返回）
    if err := h.service.UpdateUser(user); err != nil {
        c.JSON(500, Error(err))
        return
    }
    
    c.JSON(200, OK(user))
}
```

## 监控指标

```go
stats := cache.Stats()

fmt.Printf("L1命中率: %.2f%%\n", float64(stats.L1Hits)/float64(stats.TotalHits())*100)
fmt.Printf("L2命中率: %.2f%%\n", float64(stats.L2Hits)/float64(stats.TotalHits())*100)
fmt.Printf("总命中率: %.2f%%\n", stats.HitRate()*100)
```

## 性能对比

| 模式 | 读取延迟 | 写入延迟 | 数据安全 |
|-----|---------|---------|---------|
| 纯内存 | ~0.1μs | ~0.1μs | 重启丢失 |
| 三级缓存 | ~0.1μs(L1) | ~0.1μs(L1) | 最多丢1s |
| 同步写入 | ~0.1μs(L1) | ~50μs(L2) | 最多丢100ms |
| 先写库后删缓存 | ~0.1μs(L1)/~5ms(L3) | ~10ms | 数据库成功后即一致 |
| 纯数据库 | ~5ms | ~10ms | 100%安全 |

## 注意事项

1. **Redis容量**：L2使用Redis，注意设置合理的过期时间和内存限制
2. **数据倾斜**：热点数据会常驻L1和L2，冷数据只会在L3
3. **最终一致性**：L1/L2/L3之间存在延迟，强一致性场景需要特殊处理
4. **故障恢复**：L2故障时会降级到L1+L3，恢复后自动重新连接
