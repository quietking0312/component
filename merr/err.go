package merr

import (
	"errors"
	"fmt"
)

// CodedError 带错误码的错误接口
// 兼容标准 error 接口，同时支持错误码和链式 unwrap
type CodedError interface {
	error
	// Code 返回错误码
	Code() string
	// Unwrap 返回被包装的错误，兼容 errors.Is / errors.As
	Unwrap() error
}

// 确保 MErr 实现了 CodedError 和标准 error 接口
var _ CodedError = (*MErr)(nil)

// MErr 具体的带码错误实现
type MErr struct {
	code string // 错误码，如 "ERR_NOT_FOUND"
	msg  string // 错误描述
	err  error  // 被包装的错误（链式）
}

// New 创建一个新的带码错误
//   - code: 错误码，建议使用大写常量定义
//   - format: 错误描述格式字符串
//   - a: 格式参数
func New(code string, format string, a ...any) *MErr {
	return &MErr{
		code: code,
		msg:  fmt.Sprintf(format, a...),
	}
}

// Wrap 包装一个已有错误，附加错误码
func Wrap(code string, err error) *MErr {
	return &MErr{
		code: code,
		msg:  err.Error(),
		err:  err,
	}
}

// Wrapf 包装一个已有错误，附加错误码和自定义消息
func Wrapf(code string, err error, format string, a ...any) *MErr {
	return &MErr{
		code: code,
		msg:  fmt.Sprintf(format, a...),
		err:  err,
	}
}

// Error 实现标准 error 接口
func (e *MErr) Error() string {
	if e.err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.code, e.msg, e.err)
	}
	return fmt.Sprintf("[%s] %s", e.code, e.msg)
}

// Code 返回错误码
func (e *MErr) Code() string {
	return e.code
}

// Unwrap 返回被包装的错误，支持 errors.Is / errors.As
func (e *MErr) Unwrap() error {
	return e.err
}

// Is 判断目标错误是否与本错误匹配（错误码相同即匹配）
func (e *MErr) Is(target error) bool {
	if t, ok := target.(*MErr); ok {
		return e.code == t.code
	}
	return false
}

// ========== 预定义通用错误码 ==========

const (
	CodeOK             = "OK"
	CodeInvalidParam   = "INVALID_PARAM"
	CodeNotFound       = "NOT_FOUND"
	CodeUnauthorized   = "UNAUTHORIZED"
	CodeForbidden      = "FORBIDDEN"
	CodeInternalError  = "INTERNAL_ERROR"
	CodeTimeout        = "TIMEOUT"
	CodeUnavailable    = "UNAVAILABLE"
	CodeDuplicate      = "DUPLICATE"
	CodeTooManyRequest = "TOO_MANY_REQUESTS"
)

// ========== 便捷构造函数 ==========

// InvalidParam 参数错误
func InvalidParam(format string, a ...any) *MErr { return New(CodeInvalidParam, format, a...) }

// NotFound 资源未找到
func NotFound(format string, a ...any) *MErr { return New(CodeNotFound, format, a...) }

// Unauthorized 未授权
func Unauthorized(format string, a ...any) *MErr { return New(CodeUnauthorized, format, a...) }

// Forbidden 禁止访问
func Forbidden(format string, a ...any) *MErr { return New(CodeForbidden, format, a...) }

// Internal 内部错误
func Internal(format string, a ...any) *MErr { return New(CodeInternalError, format, a...) }

// Timeout 超时
func Timeout(format string, a ...any) *MErr { return New(CodeTimeout, format, a...) }

// Unavailable 服务不可用
func Unavailable(format string, a ...any) *MErr { return New(CodeUnavailable, format, a...) }

// Duplicate 重复/冲突
func Duplicate(format string, a ...any) *MErr { return New(CodeDuplicate, format, a...) }

// TooManyRequests 请求过多
func TooManyRequests(format string, a ...any) *MErr { return New(CodeTooManyRequest, format, a...) }

// IsCode 判断错误是否匹配指定错误码
// 支持标准 error 和 CodedError
func IsCode(err error, code string) bool {
	if err == nil {
		return false
	}
	var ce *MErr
	if errors.As(err, &ce) {
		return ce.code == code
	}
	return false
}
