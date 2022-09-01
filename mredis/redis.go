package mredis

import (
	"context"
	redis "github.com/go-redis/redis/v8"
	"runtime"
	"time"
)

func defaultRedisOption() *redis.Options {
	return &redis.Options{
		Network:         "tcp",
		Addr:            "127.0.0.1:6379",
		Username:        "",
		Password:        "",
		DB:              0,
		MaxRetries:      3,                // 最大重试测试； -1 禁用重试
		MinRetryBackoff: time.Duration(8), // 重试直接的最小
		MaxRetryBackoff: time.Duration(512),
		// 新连接超时时间
		DialTimeout: 5 * time.Second,
		// 读取超时
		ReadTimeout: 3 * time.Second,
		//写入超时
		WriteTimeout: 3 * time.Second,
		// 连接池大小
		PoolSize: runtime.NumCPU() * 10,
		// 空闲连接数
		MinIdleConns: 5,
		// 连接过时 时间
		MaxConnAge: 0 * time.Hour,
		// 池超时
		PoolTimeout: 4 * time.Second,
		// 空闲超时时间
		IdleTimeout: 5 * time.Minute,
		// 空闲连接超时检测 频率
		IdleCheckFrequency: 1 * time.Minute,
		// TLS 设置
		TLSConfig: nil,
		// 限制器
		Limiter: nil,
	}
}

type RedisOption func(cfg *redis.Options)

func NewRedisClient(opts ...RedisOption) (*redis.Client, error) {
	redisCfg := defaultRedisOption()
	for _, opt := range opts {
		opt(redisCfg)
	}

	redisClient := redis.NewClient(redisCfg)
	for i := 0; i < redisCfg.PoolSize; i++ {
		_, err := redisClient.Ping(context.Background()).Result()
		if err != nil {
			return nil, err
		}
	}
	return redisClient, nil
}

func defaultRedisClusterOption() *redis.ClusterOptions {
	return &redis.ClusterOptions{
		Addrs:           []string{"127.0.0.1:6379"},
		Username:        "",
		Password:        "",
		MaxRetries:      3,                // 最大重试测试； -1 禁用重试
		MinRetryBackoff: time.Duration(8), // 重试直接的最小
		MaxRetryBackoff: time.Duration(512),
		// 新连接超时时间
		DialTimeout: 5 * time.Second,
		// 读取超时
		ReadTimeout: 3 * time.Second,
		//写入超时
		WriteTimeout: 3 * time.Second,
		// 连接池大小
		PoolSize: runtime.NumCPU() * 10,
		// 空闲连接数
		MinIdleConns: 5,
		// 连接过时 时间
		MaxConnAge: 0 * time.Hour,
		// 池超时
		PoolTimeout: 4 * time.Second,
		// 空闲超时时间
		IdleTimeout: 5 * time.Minute,
		// 空闲连接超时检测 频率
		IdleCheckFrequency: 1 * time.Minute,
		// TLS 设置
		TLSConfig: nil,
	}
}

type ClusterOption func(cfg *redis.ClusterOptions)

func NewRedisClusterClient(opts ...ClusterOption) (*redis.ClusterClient, error) {
	redisCfg := defaultRedisClusterOption()
	for _, opt := range opts {
		opt(redisCfg)
	}
	redisClusterClient := redis.NewClusterClient(redisCfg)
	err := redisClusterClient.ForEachShard(context.Background(), func(ctx context.Context, client *redis.Client) error {
		return client.Ping(ctx).Err()
	})
	if err != nil {
		return nil, err
	}

	return redisClusterClient, nil
}
