package mhttp

import (
	"net"
	"net/http"
	"time"
)

// Client HTTP 客户端封装
type Client struct {
	client  *http.Client
	timeout time.Duration
}

// Option 客户端配置选项
type Option func(*Client)

// NewClient 创建一个新的 HTTP 客户端
func NewClient(opts ...Option) *Client {
	c := &Client{
		client: &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{
					Timeout:   5 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          100,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   5 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		},
		timeout: 30 * time.Second,
	}

	for _, opt := range opts {
		opt(c)
	}

	c.client.Timeout = c.timeout
	return c
}

// WithTimeout 设置请求超时时间
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.timeout = timeout
	}
}

// WithHTTPClient 设置自定义的 http.Client
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		c.client = client
	}
}

// HTTPClient 获取底层的 http.Client
func (c *Client) HTTPClient() *http.Client {
	return c.client
}

// Get 发起 GET 请求
func (c *Client) Get(url string) *Request {
	return NewRequest(c, http.MethodGet, url)
}

// Post 发起 POST 请求
func (c *Client) Post(url string) *Request {
	return NewRequest(c, http.MethodPost, url)
}

// Put 发起 PUT 请求
func (c *Client) Put(url string) *Request {
	return NewRequest(c, http.MethodPut, url)
}

// Delete 发起 DELETE 请求
func (c *Client) Delete(url string) *Request {
	return NewRequest(c, http.MethodDelete, url)
}

// Patch 发起 PATCH 请求
func (c *Client) Patch(url string) *Request {
	return NewRequest(c, http.MethodPatch, url)
}

// Head 发起 HEAD 请求
func (c *Client) Head(url string) *Request {
	return NewRequest(c, http.MethodHead, url)
}

// Options 发起 OPTIONS 请求
func (c *Client) Options(url string) *Request {
	return NewRequest(c, http.MethodOptions, url)
}

// Do 执行预构建好的 http.Request
func (c *Client) Do(req *http.Request) (*Response, error) {
	return doRequest(c.client, req)
}

// ========== 包级默认客户端 ==========

var defaultClient = NewClient()

// SetDefaultTimeout 设置默认客户端的超时时间
func SetDefaultTimeout(timeout time.Duration) {
	defaultClient.timeout = timeout
	defaultClient.client.Timeout = timeout
}

// Get 使用默认客户端发起 GET 请求
func Get(url string) *Request {
	return defaultClient.Get(url)
}

// Post 使用默认客户端发起 POST 请求
func Post(url string) *Request {
	return defaultClient.Post(url)
}

// Put 使用默认客户端发起 PUT 请求
func Put(url string) *Request {
	return defaultClient.Put(url)
}

// Delete 使用默认客户端发起 DELETE 请求
func Delete(url string) *Request {
	return defaultClient.Delete(url)
}

// Patch 使用默认客户端发起 PATCH 请求
func Patch(url string) *Request {
	return defaultClient.Patch(url)
}

// Head 使用默认客户端发起 HEAD 请求
func Head(url string) *Request {
	return defaultClient.Head(url)
}

// Options 使用默认客户端发起 OPTIONS 请求
func Options(url string) *Request {
	return defaultClient.Options(url)
}

// Do 使用默认客户端执行请求
func Do(req *http.Request) (*Response, error) {
	return defaultClient.Do(req)
}
