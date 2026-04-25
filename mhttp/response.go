package mhttp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Response HTTP 响应封装
type Response struct {
	response *http.Response
	body     []byte
	err      error
}

// doRequest 执行 HTTP 请求
func doRequest(client *http.Client, req *http.Request) (*Response, error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request failed: %w", err)
	}

	body, err := io.ReadAll(resp.Body)
	if closeErr := resp.Body.Close(); closeErr != nil {
		// 忽略关闭错误，但优先返回读取错误
		if err == nil {
			err = closeErr
		}
	}

	return &Response{
		response: resp,
		body:     body,
		err:      err,
	}, nil
}

// StatusCode 返回 HTTP 状态码
func (r *Response) StatusCode() int {
	if r.response == nil {
		return 0
	}
	return r.response.StatusCode
}

// Status 返回 HTTP 状态文本
func (r *Response) Status() string {
	if r.response == nil {
		return ""
	}
	return r.response.Status
}

// Header 返回响应头
func (r *Response) Header() http.Header {
	if r.response == nil {
		return nil
	}
	return r.response.Header
}

// GetHeader 获取单个响应头值
func (r *Response) GetHeader(key string) string {
	if r.response == nil {
		return ""
	}
	return r.response.Header.Get(key)
}

// ContentType 返回响应的 Content-Type
func (r *Response) ContentType() string {
	return r.GetHeader("Content-Type")
}

// ContentLength 返回响应体长度
func (r *Response) ContentLength() int64 {
	if r.response == nil {
		return 0
	}
	return r.response.ContentLength
}

// Body 返回响应体字节数组（可重复读取）
func (r *Response) Body() []byte {
	return r.body
}

// Bytes 返回响应体字节数组（别名）
func (r *Response) Bytes() []byte {
	return r.body
}

// String 返回响应体字符串
func (r *Response) String() string {
	return string(r.body)
}

// IsSuccess 判断请求是否成功（2xx）
func (r *Response) IsSuccess() bool {
	return r.StatusCode() >= 200 && r.StatusCode() < 300
}

// IsError 判断请求是否失败（4xx/5xx）
func (r *Response) IsError() bool {
	return r.StatusCode() >= 400
}

// IsOK 判断状态码是否为 200
func (r *Response) IsOK() bool {
	return r.StatusCode() == http.StatusOK
}

// JSON 将响应体 JSON 反序列化到 v
func (r *Response) JSON(v interface{}) error {
	if r.err != nil {
		return r.err
	}
	if len(r.body) == 0 {
		return nil
	}
	return json.Unmarshal(r.body, v)
}

// Error 返回读取响应体时的错误
func (r *Response) Error() error {
	return r.err
}

// Close 空操作，保持与 io.ReadCloser 兼容
// 实际上响应体在 doRequest 中已经被读取并关闭
func (r *Response) Close() error {
	return nil
}

// RawResponse 返回原始的 http.Response
func (r *Response) RawResponse() *http.Response {
	return r.response
}

// Cookies 返回响应中的 Cookies
func (r *Response) Cookies() []*http.Cookie {
	if r.response == nil {
		return nil
	}
	return r.response.Cookies()
}
