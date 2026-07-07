package mcachedb

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis 层默认常量
const (
	defaultRedisAddr         = "localhost:6379"
	defaultRedisKeyPrefix    = "mcachedb:"
	defaultRedisTTL          = 5 * time.Minute
	defaultRedisPoolSize     = 10
	defaultRedisMinIdleConns = 2
	redisConnectTimeout      = 5 * time.Second
)

// redisTypedWrapper 带格式标记的 Redis 存储包装
// 用于在配置了 entityType 时，与旧版 EntityWrapper 格式做区分
type redisTypedWrapper struct {
	Format string          `json:"_mc_format_"` // 固定为 "typed"
	Data   json.RawMessage `json:"data"`        // 原始实体 JSON
}

// RedisStore Redis存储实现（支持单机和集群模式）
type RedisStore struct {
	cmdable    redis.Cmdable
	closer     io.Closer
	pipeliner  func() redis.Pipeliner
	subscriber redisSubscribable

	keyPrefix  string
	defaultTTL time.Duration
	entityType Entity // 可选：反序列化时恢复的具体实体类型
}

// redisSubscribable 订阅接口（仅单机模式支持）
type redisSubscribable interface {
	Subscribe(ctx context.Context, channels ...string) *redis.PubSub
}

// RedisConfig Redis配置
type RedisConfig struct {
	Addr         string
	Addrs        []string // 集群地址列表（如 ["127.0.0.1:7000", "127.0.0.1:7001"]）
	Password     string
	DB           int
	KeyPrefix    string
	DefaultTTL   time.Duration
	PoolSize     int
	MinIdleConns int
	ClusterMode  bool // 显式开启集群模式
}

// DefaultRedisConfig 默认Redis配置
func DefaultRedisConfig() *RedisConfig {
	return &RedisConfig{
		Addr:         defaultRedisAddr,
		Password:     "",
		DB:           0,
		KeyPrefix:    defaultRedisKeyPrefix,
		DefaultTTL:   defaultRedisTTL,
		PoolSize:     defaultRedisPoolSize,
		MinIdleConns: defaultRedisMinIdleConns,
	}
}

// NewRedisStore 创建Redis存储（自动识别单机/集群模式）
func NewRedisStore(config *RedisConfig, entityType ...Entity) (*RedisStore, error) {
	if config == nil {
		config = DefaultRedisConfig()
	}

	rs := &RedisStore{
		keyPrefix:  config.KeyPrefix,
		defaultTTL: config.DefaultTTL,
	}
	if len(entityType) > 0 && entityType[0] != nil {
		rs.entityType = entityType[0]
	}

	// 判断是否为集群模式：显式开启或提供了多个地址
	isCluster := config.ClusterMode || len(config.Addrs) > 1

	ctx, cancel := context.WithTimeout(context.Background(), redisConnectTimeout)
	defer cancel()

	if isCluster {
		addrs := config.Addrs
		if len(addrs) == 0 && config.Addr != "" {
			addrs = []string{config.Addr}
		}
		if len(addrs) == 0 {
			return nil, fmt.Errorf("redis cluster addrs is empty")
		}

		clusterClient := redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:        addrs,
			Password:     config.Password,
			PoolSize:     config.PoolSize,
			MinIdleConns: config.MinIdleConns,
		})

		if err := clusterClient.Ping(ctx).Err(); err != nil {
			_ = clusterClient.Close()
			return nil, fmt.Errorf("redis cluster connect failed: %w", err)
		}

		rs.cmdable = clusterClient
		rs.closer = clusterClient
		rs.pipeliner = func() redis.Pipeliner { return clusterClient.Pipeline() }
		// 集群模式下不直接支持通用 Subscribe（channel 只存在于单个节点，需业务层自行处理或使用 hash tag）
	} else {
		addr := config.Addr
		if addr == "" && len(config.Addrs) == 1 {
			addr = config.Addrs[0]
		}
		if addr == "" {
			addr = defaultRedisAddr
		}

		singleClient := redis.NewClient(&redis.Options{
			Addr:         addr,
			Password:     config.Password,
			DB:           config.DB,
			PoolSize:     config.PoolSize,
			MinIdleConns: config.MinIdleConns,
		})

		if err := singleClient.Ping(ctx).Err(); err != nil {
			_ = singleClient.Close()
			return nil, fmt.Errorf("redis connect failed: %w", err)
		}

		rs.cmdable = singleClient
		rs.closer = singleClient
		rs.pipeliner = func() redis.Pipeliner { return singleClient.Pipeline() }
		rs.subscriber = singleClient
	}

	return rs, nil
}

// makeKey 生成带前缀的key
func (s *RedisStore) makeKey(key string) string {
	return s.keyPrefix + key
}

// Get 从Redis获取
func (s *RedisStore) Get(ctx context.Context, key string) (Entity, error) {
	data, err := s.cmdable.Get(ctx, s.makeKey(key)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return s.unmarshalEntity(data)
}

// MGet 批量获取（集群模式下 go-redis 会自动按 slot 分片执行）
func (s *RedisStore) MGet(ctx context.Context, keys []string) (map[string]Entity, error) {
	if len(keys) == 0 {
		return make(map[string]Entity), nil
	}

	prefixedKeys := make([]string, len(keys))
	for i, key := range keys {
		prefixedKeys[i] = s.makeKey(key)
	}

	results, err := s.cmdable.MGet(ctx, prefixedKeys...).Result()
	if err != nil {
		return nil, err
	}

	entities := make(map[string]Entity)
	for i, result := range results {
		if result == nil {
			continue
		}

		data, ok := result.(string)
		if !ok {
			continue
		}

		entity, err := s.unmarshalEntity([]byte(data))
		if err != nil {
			continue
		}

		entities[keys[i]] = entity
	}

	return entities, nil
}

// Set 设置到Redis
func (s *RedisStore) Set(ctx context.Context, entity Entity) error {
	data, err := s.marshalEntity(entity)
	if err != nil {
		return err
	}

	key := s.makeKey(entity.CacheKey())

	ttl := s.defaultTTL
	if ttl > 0 {
		return s.cmdable.Set(ctx, key, data, ttl).Err()
	}
	return s.cmdable.Set(ctx, key, data, 0).Err()
}

// MSet 批量设置（集群模式下 pipeline 会自动按 slot 分片到各节点执行）
func (s *RedisStore) MSet(ctx context.Context, entities []Entity) error {
	if len(entities) == 0 {
		return nil
	}

	pipe := s.pipeliner()

	for _, entity := range entities {
		data, err := s.marshalEntity(entity)
		if err != nil {
			return err
		}

		key := s.makeKey(entity.CacheKey())
		pipe.Set(ctx, key, data, s.defaultTTL)
	}

	_, err := pipe.Exec(ctx)
	return err
}

// unmarshalEntity 根据是否配置了 entityType 选择反序列化方式
func (s *RedisStore) unmarshalEntity(data []byte) (Entity, error) {
	// 若配置了 entityType，先尝试解析带格式标记的 typed wrapper
	if s.entityType != nil {
		var typed redisTypedWrapper
		if err := json.Unmarshal(data, &typed); err == nil && typed.Format == "typed" {
			entity := s.entityType.Copy()
			if err := json.Unmarshal(typed.Data, entity); err == nil {
				return entity, nil
			}
		}
		// 解析失败可能是旧格式（EntityWrapper），继续降级解析
	}

	var wrapper EntityWrapper
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, err
	}

	return &wrapper, nil
}

// marshalEntity 根据是否配置了 entityType 选择序列化方式
func (s *RedisStore) marshalEntity(entity Entity) ([]byte, error) {
	if s.entityType != nil {
		// 配置了具体类型：用 typed wrapper 存储原始实体 JSON，保持类型信息
		rawData, err := json.Marshal(entity)
		if err != nil {
			return nil, err
		}
		wrapper := redisTypedWrapper{
			Format: "typed",
			Data:   rawData,
		}
		return json.Marshal(wrapper)
	}

	// 未配置类型：使用 EntityWrapper 保持兼容
	wrapper := &EntityWrapper{
		BaseEntity: *NewBaseEntity(entity.CacheKey()),
		Data:       entity,
	}
	wrapper.Ver = entity.Version()
	wrapper.DelFlag = entity.IsDeleted()

	return json.Marshal(wrapper)
}

// Delete 从Redis删除
func (s *RedisStore) Delete(ctx context.Context, key string) error {
	return s.cmdable.Del(ctx, s.makeKey(key)).Err()
}

// MDelete 批量删除（集群模式下 go-redis 会自动按 slot 分片执行）
func (s *RedisStore) MDelete(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}

	prefixedKeys := make([]string, len(keys))
	for i, key := range keys {
		prefixedKeys[i] = s.makeKey(key)
	}

	return s.cmdable.Del(ctx, prefixedKeys...).Err()
}

// Close 关闭连接
func (s *RedisStore) Close() error {
	if s.closer != nil {
		return s.closer.Close()
	}
	return nil
}

// Publish 发布消息
func (s *RedisStore) Publish(ctx context.Context, channel string, message string) error {
	return s.cmdable.Publish(ctx, channel, message).Err()
}

// Subscribe 订阅消息（集群模式下暂不支持，返回 nil）
func (s *RedisStore) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	if s.subscriber == nil {
		return nil
	}
	return s.subscriber.Subscribe(ctx, channels...)
}
