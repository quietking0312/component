package mhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Request HTTP 请求构建器
type Request struct {
	client  *Client
	method  string
	url     string
	headers map[string]string
	queries map[string]string
	body    io.Reader
	ctx     context.Context
}

// NewRequest 创建一个新的请求构建器
func NewRequest(client *Client, method, url string) *Request {
	return &Request{
		client:  client,
		method:  method,
		url:     url,
		headers: make(map[string]string),
		queries: make(map[string]string),
		ctx:     context.Background(),
	}
}

// SetHeader 设置单个请求头
func (r *Request) SetHeader(key, value string) *Request {
	r.headers[key] = value
	return r
}

// SetHeaders 批量设置请求头
func (r *Request) SetHeaders(headers map[string]string) *Request {
	for k, v := range headers {
		r.headers[k] = v
	}
	return r
}

// SetContentType 设置 Content-Type
func (r *Request) SetContentType(contentType string) *Request {
	r.headers["Content-Type"] = contentType
	return r
}

// SetAuthorization 设置 Authorization 请求头
func (r *Request) SetAuthorization(token string) *Request {
	r.headers["Authorization"] = token
	return r
}

// SetBearerToken 设置 Bearer Token
func (r *Request) SetBearerToken(token string) *Request {
	r.headers["Authorization"] = "Bearer " + token
	return r
}

// Query 设置单个查询参数
func (r *Request) Query(key, value string) *Request {
	r.queries[key] = value
	return r
}

// Queries 批量设置查询参数
func (r *Request) Queries(queries map[string]string) *Request {
	for k, v := range queries {
		r.queries[k] = v
	}
	return r
}

// Body 设置原始请求体
func (r *Request) Body(body io.Reader) *Request {
	r.body = body
	return r
}

// BodyBytes 设置字节数组请求体
func (r *Request) BodyBytes(body []byte) *Request {
	r.body = bytes.NewReader(body)
	return r
}

// BodyString 设置字符串请求体
func (r *Request) BodyString(body string) *Request {
	r.body = strings.NewReader(body)
	return r
}

// BodyJSON 设置 JSON 请求体（自动序列化并设置 Content-Type）
func (r *Request) BodyJSON(body interface{}) *Request {
	data, err := json.Marshal(body)
	if err != nil {
		// 序列化失败时，存储错误标记，在 Do 时处理
		r.body = &marshalErrorReader{err: err}
		return r
	}
	r.body = bytes.NewReader(data)
	r.headers["Content-Type"] = "application/json"
	return r
}

// WithContext 设置请求的 context
func (r *Request) WithContext(ctx context.Context) *Request {
	r.ctx = ctx
	return r
}

// Do 执行请求并返回响应
func (r *Request) Do() (*Response, error) {
	start := time.Now()

	// 构建 URL（附加查询参数）
	reqURL := r.url
	if len(r.queries) > 0 {
		u, err := url.Parse(r.url)
		if err != nil {
			return nil, fmt.Errorf("parse url failed: %w", err)
		}
		q := u.Query()
		for k, v := range r.queries {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
		reqURL = u.String()
	}

	// 创建请求
	req, err := http.NewRequestWithContext(r.ctx, r.method, reqURL, r.body)
	if err != nil {
		return nil, fmt.Errorf("create request failed: %w", err)
	}

	// 检查 body 序列化错误
	if mer, ok := r.body.(*marshalErrorReader); ok {
		return nil, fmt.Errorf("marshal body failed: %w", mer.err)
	}

	// 设置请求头
	for k, v := range r.headers {
		req.Header.Set(k, v)
	}

	// 执行请求
	resp, err := doRequest(r.client.client, req)

	// 记录日志
	duration := time.Since(start)
	if err != nil {
		r.client.logger.Error("http request failed",
			"method", r.method,
			"url", reqURL,
			"duration", duration,
			"error", err,
		)
		return nil, err
	}
	r.client.logger.Info("http request completed",
		"method", r.method,
		"url", reqURL,
		"duration", duration,
		"status", resp.StatusCode(),
	)

	return resp, nil
}

// DoAndBindJSON 执行请求并将响应体 JSON 反序列化到 v
func (r *Request) DoAndBindJSON(v interface{}) error {
	resp, err := r.Do()
	if err != nil {
		return err
	}
	defer resp.Close()
	return resp.JSON(v)
}

// DoAndBindString 执行请求并返回响应体字符串
func (r *Request) DoAndBindString() (string, error) {
	resp, err := r.Do()
	if err != nil {
		return "", err
	}
	defer resp.Close()
	return resp.String(), nil
}

// DoAndBindBytes 执行请求并返回响应体字节数组
func (r *Request) DoAndBindBytes() ([]byte, error) {
	resp, err := r.Do()
	if err != nil {
		return nil, err
	}
	defer resp.Close()
	return resp.Bytes(), nil
}

// marshalErrorReader 用于标记 JSON 序列化失败的 Reader
type marshalErrorReader struct {
	err error
}

func (r *marshalErrorReader) Read(p []byte) (n int, err error) {
	return 0, r.err
}
