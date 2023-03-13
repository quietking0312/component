package mredis

import (
	"context"
	"github.com/redis/go-redis/v9"
)

type RedisClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	GetRange(ctx context.Context, key string, start, end int64) *redis.StringCmd

	Do(ctx context.Context, args ...interface{}) *redis.Cmd
	Watch(ctx context.Context, fn func(tx *redis.Tx) error, keys ...string) error
}

func NewClient(a string) (RedisClient, error) {
	if a == "" {
		return NewRedisClient()
	} else {
		return NewRedisClusterClient()
	}
}
