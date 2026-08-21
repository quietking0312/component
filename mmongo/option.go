package mmongo

import (
	"crypto/tls"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Options 保存 MongoDB 客户端的连接配置。
type Options struct {
	// URI 完整连接串（如 "mongodb://user:pass@host1,host2/?replicaSet=rs0"）。
	// 设置后优先使用 URI，忽略 Hosts/Username/Password/AuthSource/ReplicaSet
	URI string
	// Hosts 服务地址列表，未设置 URI 时使用
	Hosts []string
	// Username 认证用户名
	Username string
	// Password 认证密码
	Password string
	// AuthSource 认证数据库，默认 "admin"
	AuthSource string
	// Database 默认操作的数据库名，Database() 未传参时使用
	Database string
	// ReplicaSet 副本集名称
	ReplicaSet string
	// MaxPoolSize 连接池最大连接数
	MaxPoolSize uint64
	// MinPoolSize 连接池最小连接数
	MinPoolSize uint64
	// ConnectTimeout 建立连接超时，同时用作 NewClient 的 Ping 超时
	ConnectTimeout time.Duration
	// Timeout 单次操作的整体超时（v2 驱动统一超时机制，取代旧版 SocketTimeout）
	Timeout time.Duration
	// ServerSelectionTimeout 选择可用服务节点的超时
	ServerSelectionTimeout time.Duration
	// TLSConfig TLS 配置，nil 表示不启用
	TLSConfig *tls.Config
}

// Option 配置 Options。
type Option func(cfg *Options)

func defaultOptions() *Options {
	return &Options{
		Hosts:                  []string{"127.0.0.1:27017"},
		AuthSource:             "admin",
		MaxPoolSize:            100,
		MinPoolSize:            5,
		ConnectTimeout:         10 * time.Second,
		Timeout:                30 * time.Second,
		ServerSelectionTimeout: 10 * time.Second,
	}
}

// SetURI 设置完整的 MongoDB 连接 URI。
func SetURI(uri string) Option {
	return func(cfg *Options) { cfg.URI = uri }
}

// SetHosts 设置 MongoDB 服务地址列表。
func SetHosts(hosts []string) Option {
	return func(cfg *Options) { cfg.Hosts = hosts }
}

// SetAuth 设置用于认证的用户名和密码。
func SetAuth(username, password string) Option {
	return func(cfg *Options) {
		cfg.Username = username
		cfg.Password = password
	}
}

// SetAuthSource 设置认证数据库。
func SetAuthSource(authSource string) Option {
	return func(cfg *Options) { cfg.AuthSource = authSource }
}

// SetDatabase 设置默认数据库名。
func SetDatabase(database string) Option {
	return func(cfg *Options) { cfg.Database = database }
}

// SetReplicaSet 设置副本集名称。
func SetReplicaSet(replicaSet string) Option {
	return func(cfg *Options) { cfg.ReplicaSet = replicaSet }
}

// SetMaxPoolSize 设置连接池最大连接数。
func SetMaxPoolSize(size uint64) Option {
	return func(cfg *Options) { cfg.MaxPoolSize = size }
}

// SetMinPoolSize 设置连接池最小连接数。
func SetMinPoolSize(size uint64) Option {
	return func(cfg *Options) { cfg.MinPoolSize = size }
}

// SetConnectTimeout 设置连接超时。
func SetConnectTimeout(d time.Duration) Option {
	return func(cfg *Options) { cfg.ConnectTimeout = d }
}

// SetTimeout 设置单次操作的整体超时（v2 驱动统一的超时机制，
// 取代了旧版的 SocketTimeout）。
func SetTimeout(d time.Duration) Option {
	return func(cfg *Options) { cfg.Timeout = d }
}

// SetServerSelectionTimeout 设置服务节点选择超时。
func SetServerSelectionTimeout(d time.Duration) Option {
	return func(cfg *Options) { cfg.ServerSelectionTimeout = d }
}

// SetTLSConfig 设置 TLS 配置。
func SetTLSConfig(cfg *tls.Config) Option {
	return func(o *Options) { o.TLSConfig = cfg }
}

// clientOptions 根据 Options 构建驱动所需的 *options.ClientOptions。
func (cfg *Options) clientOptions() *options.ClientOptions {
	clientOpts := options.Client()

	if cfg.URI != "" {
		clientOpts.ApplyURI(cfg.URI)
	} else {
		if len(cfg.Hosts) > 0 {
			clientOpts.SetHosts(cfg.Hosts)
		}
		if cfg.Username != "" {
			clientOpts.SetAuth(options.Credential{
				AuthSource: cfg.AuthSource,
				Username:   cfg.Username,
				Password:   cfg.Password,
			})
		}
		if cfg.ReplicaSet != "" {
			clientOpts.SetReplicaSet(cfg.ReplicaSet)
		}
	}

	if cfg.MaxPoolSize != 0 {
		clientOpts.SetMaxPoolSize(cfg.MaxPoolSize)
	}
	if cfg.MinPoolSize != 0 {
		clientOpts.SetMinPoolSize(cfg.MinPoolSize)
	}
	if cfg.ConnectTimeout != 0 {
		clientOpts.SetConnectTimeout(cfg.ConnectTimeout)
	}
	if cfg.Timeout != 0 {
		clientOpts.SetTimeout(cfg.Timeout)
	}
	if cfg.ServerSelectionTimeout != 0 {
		clientOpts.SetServerSelectionTimeout(cfg.ServerSelectionTimeout)
	}
	if cfg.TLSConfig != nil {
		clientOpts.SetTLSConfig(cfg.TLSConfig)
	}

	return clientOpts
}
