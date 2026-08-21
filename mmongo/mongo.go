package mmongo

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Client 包装了官方的 mongo.Client，直接暴露其所有方法，
// 使调用方能够不加改动地使用标准驱动的 API。
type Client struct {
	*mongo.Client
	Config *Options
}

// NewClient 连接到 MongoDB，通过 Ping 校验连通性，并返回一个
// 包装了官方驱动 client 的 *Client。
func NewClient(opts ...Option) (*Client, error) {
	cfg := defaultOptions()
	for _, opt := range opts {
		opt(cfg)
	}

	client, err := mongo.Connect(cfg.clientOptions())
	if err != nil {
		return nil, err
	}

	pingCtx, pingCancel := context.WithTimeout(context.Background(), cfg.ConnectTimeout)
	defer pingCancel()
	if err := client.Ping(pingCtx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, err
	}

	return &Client{
		Client: client,
		Config: cfg,
	}, nil
}

// Database 返回给定名字的数据库；未传 name 时返回配置的默认数据库
// （Config.Database）。
func (c *Client) Database(name ...string) *mongo.Database {
	if len(name) > 0 && name[0] != "" {
		return c.Client.Database(name[0])
	}
	return c.Client.Database(c.Config.Database)
}

// Close 断开客户端连接，释放连接池中的所有连接。
func (c *Client) Close(ctx context.Context) error {
	return c.Client.Disconnect(ctx)
}
