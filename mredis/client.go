package mredis

import (
	"crypto/tls"
	"github.com/redis/go-redis/v9"
	"time"
)

const (
	ModeCluster = "cluster"
	ModeSingle  = "single"
)

type Client interface {
	redis.Cmdable
}

type Options struct {
	Addrs        []string
	Username     string
	Password     string
	Mode         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PoolSize     int
	MinIdleConns int
	TLSConfig    *tls.Config
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
	if cfg.Mode == ModeCluster {
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
	} else {
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
	return func(cfg *Options) {
		cfg.Addrs = addrs
	}
}

func SetAuth(username, password string) Option {
	return func(cfg *Options) {
		cfg.Username = username
		cfg.Password = password
	}
}

func SetReadTimeout(readTimeout time.Duration) Option {
	return func(cfg *Options) {
		cfg.ReadTimeout = readTimeout
	}
}

func SetWriteTimeout(writeTimeout time.Duration) Option {
	return func(cfg *Options) {
		cfg.WriteTimeout = writeTimeout
	}
}

func SetPoolSize(poolSize int) Option {
	return func(cfg *Options) {
		cfg.PoolSize = poolSize
	}
}
func SetMinIdleConns(minIdle int) Option {
	return func(cfg *Options) {
		cfg.MinIdleConns = minIdle
	}
}

func SetTLSConfig(tlsConfig *tls.Config) Option {
	return func(cfg *Options) {
		cfg.TLSConfig = tlsConfig
	}
}
