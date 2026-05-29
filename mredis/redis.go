package mredis

import (
	"context"
	redis "github.com/redis/go-redis/v9"
	"runtime"
	"time"
)

func defaultRedisOption() *redis.Options {
	return &redis.Options{
		Network:         "tcp",
		Addr:            "127.0.0.1:6379",
		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		PoolSize:        runtime.NumCPU() * 10,
		MinIdleConns:    5,
		PoolTimeout:     4 * time.Second,
	}
}

type RedisOption func(cfg *redis.Options)

func NewRedisClient(opts ...RedisOption) (*redis.Client, error) {
	redisCfg := defaultRedisOption()
	for _, opt := range opts {
		opt(redisCfg)
	}
	redisClient := redis.NewClient(redisCfg)
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		_ = redisClient.Close()
		return nil, err
	}
	return redisClient, nil
}

func defaultRedisClusterOption() *redis.ClusterOptions {
	return &redis.ClusterOptions{
		Addrs:           []string{"127.0.0.1:6379"},
		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		PoolSize:        runtime.NumCPU() * 10,
		MinIdleConns:    5,
		PoolTimeout:     4 * time.Second,
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
		_ = redisClusterClient.Close()
		return nil, err
	}
	return redisClusterClient, nil
}

func defaultRedisSentinelOption() *redis.FailoverOptions {
	return &redis.FailoverOptions{
		MasterName:      "mymaster",
		SentinelAddrs:   []string{"127.0.0.1:26379"},
		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		PoolSize:        runtime.NumCPU() * 10,
		MinIdleConns:    5,
		PoolTimeout:     4 * time.Second,
	}
}

type SentinelOption func(cfg *redis.FailoverOptions)

func NewRedisSentinelClient(opts ...SentinelOption) (*redis.Client, error) {
	cfg := defaultRedisSentinelOption()
	for _, opt := range opts {
		opt(cfg)
	}
	client := redis.NewFailoverClient(cfg)
	if err := client.Ping(context.Background()).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}
	return client, nil
}
