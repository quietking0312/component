package mredis

import (
	"context"
	"crypto/tls"
	"github.com/redis/go-redis/v9"
	"time"
)

const (
	// ModeCluster 集群模式：多主多从，数据按 slot 分片，适合大规模高可用场景
	ModeCluster = "cluster"
	// ModeSingle 单机模式：单个 Redis 实例，默认模式
	ModeSingle = "single"
	// ModeSentinel 哨兵模式：通过 Sentinel 自动发现主节点，主宕机时自动故障转移
	ModeSentinel = "sentinel"
)

type Client interface {
	redis.Cmdable
	Subscribe
	Close() error
}

type Subscribe interface {
	SSubscribe(ctx context.Context, channels ...string) *redis.PubSub
	PSubscribe(ctx context.Context, channels ...string) *redis.PubSub
	Subscribe(ctx context.Context, channel ...string) *redis.PubSub
}

type Options struct {
	// Addrs 服务地址列表。单机填一个，集群填所有节点，哨兵模式填所有 Sentinel 地址
	Addrs []string
	// Username ACL 用户名，Redis 6.0+ 启用 ACL 时使用
	Username string
	// Password 认证密码
	Password string
	// DB 数据库编号（0-15），仅单机和哨兵模式有效，集群模式固定为 0
	DB int
	// Mode 连接模式：ModeSingle（默认）、ModeCluster、ModeSentinel
	Mode string
	// ReadTimeout 读超时，默认 3s，-1 表示不超时
	ReadTimeout time.Duration
	// WriteTimeout 写超时，默认 3s，-1 表示不超时
	WriteTimeout time.Duration
	// PoolSize 每个节点的最大连接数，默认 NumCPU*10
	PoolSize int
	// MinIdleConns 最小空闲连接数，避免流量突增时建连延迟
	MinIdleConns int
	// TLSConfig TLS 配置，nil 表示不启用
	TLSConfig *tls.Config
	// MasterName 哨兵模式下的主节点名称，须与 Sentinel 配置一致
	MasterName string
	// SentinelAddrs 哨兵节点地址列表，仅哨兵模式使用
	SentinelAddrs []string
	// SentinelPassword 哨兵节点的认证密码，与 Redis 节点密码相互独立
	SentinelPassword string
}

type Option func(cfg *Options)

func defaultOptions() *Options {
	return &Options{}
}

func NewClient(opts ...Option) (Client, error) {
	cfg := defaultOptions()
	for _, opt := range opts {
		opt(cfg)
	}
	switch cfg.Mode {
	case ModeCluster:
		return NewRedisClusterClient(func(c *redis.ClusterOptions) {
			if len(cfg.Addrs) > 0 {
				c.Addrs = cfg.Addrs
			}
			if cfg.Username != "" {
				c.Username = cfg.Username
			}
			if cfg.Password != "" {
				c.Password = cfg.Password
			}
			if cfg.ReadTimeout != 0 {
				c.ReadTimeout = cfg.ReadTimeout
			}
			if cfg.WriteTimeout != 0 {
				c.WriteTimeout = cfg.WriteTimeout
			}
			if cfg.PoolSize != 0 {
				c.PoolSize = cfg.PoolSize
			}
			if cfg.MinIdleConns != 0 {
				c.MinIdleConns = cfg.MinIdleConns
			}
			if cfg.TLSConfig != nil {
				c.TLSConfig = cfg.TLSConfig
			}
		})
	case ModeSentinel:
		return NewRedisSentinelClient(func(c *redis.FailoverOptions) {
			if len(cfg.SentinelAddrs) > 0 {
				c.SentinelAddrs = cfg.SentinelAddrs
			}
			if cfg.MasterName != "" {
				c.MasterName = cfg.MasterName
			}
			if cfg.Username != "" {
				c.Username = cfg.Username
			}
			if cfg.Password != "" {
				c.Password = cfg.Password
			}
			if cfg.SentinelPassword != "" {
				c.SentinelPassword = cfg.SentinelPassword
			}
			if cfg.DB != 0 {
				c.DB = cfg.DB
			}
			if cfg.ReadTimeout != 0 {
				c.ReadTimeout = cfg.ReadTimeout
			}
			if cfg.WriteTimeout != 0 {
				c.WriteTimeout = cfg.WriteTimeout
			}
			if cfg.PoolSize != 0 {
				c.PoolSize = cfg.PoolSize
			}
			if cfg.MinIdleConns != 0 {
				c.MinIdleConns = cfg.MinIdleConns
			}
			if cfg.TLSConfig != nil {
				c.TLSConfig = cfg.TLSConfig
			}
		})
	default: // ModeSingle
		return NewRedisClient(func(c *redis.Options) {
			if len(cfg.Addrs) > 0 {
				c.Addr = cfg.Addrs[0]
			}
			if cfg.Username != "" {
				c.Username = cfg.Username
			}
			if cfg.Password != "" {
				c.Password = cfg.Password
			}
			if cfg.DB != 0 {
				c.DB = cfg.DB
			}
			if cfg.ReadTimeout != 0 {
				c.ReadTimeout = cfg.ReadTimeout
			}
			if cfg.WriteTimeout != 0 {
				c.WriteTimeout = cfg.WriteTimeout
			}
			if cfg.PoolSize != 0 {
				c.PoolSize = cfg.PoolSize
			}
			if cfg.MinIdleConns != 0 {
				c.MinIdleConns = cfg.MinIdleConns
			}
			if cfg.TLSConfig != nil {
				c.TLSConfig = cfg.TLSConfig
			}
		})
	}
}

func SetAddrs(addrs []string) Option {
	return func(cfg *Options) { cfg.Addrs = addrs }
}

func SetAuth(username, password string) Option {
	return func(cfg *Options) {
		cfg.Username = username
		cfg.Password = password
	}
}

func SetDB(db int) Option {
	return func(cfg *Options) { cfg.DB = db }
}

func SetMode(mode string) Option {
	return func(cfg *Options) { cfg.Mode = mode }
}

func SetReadTimeout(readTimeout time.Duration) Option {
	return func(cfg *Options) { cfg.ReadTimeout = readTimeout }
}

func SetWriteTimeout(writeTimeout time.Duration) Option {
	return func(cfg *Options) { cfg.WriteTimeout = writeTimeout }
}

func SetPoolSize(poolSize int) Option {
	return func(cfg *Options) { cfg.PoolSize = poolSize }
}

func SetMinIdleConns(minIdle int) Option {
	return func(cfg *Options) { cfg.MinIdleConns = minIdle }
}

func SetTLSConfig(tlsConfig *tls.Config) Option {
	return func(cfg *Options) { cfg.TLSConfig = tlsConfig }
}

func SetMasterName(masterName string) Option {
	return func(cfg *Options) { cfg.MasterName = masterName }
}

func SetSentinelAddrs(addrs []string) Option {
	return func(cfg *Options) { cfg.SentinelAddrs = addrs }
}

func SetSentinelPassword(password string) Option {
	return func(cfg *Options) { cfg.SentinelPassword = password }
}
