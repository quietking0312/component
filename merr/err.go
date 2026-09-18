package merr

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"sync"
)

// Code 错误码约束：只能是字符串或 int32（包含它们的具名类型，如 proto 生成的枚举）
// 传入其他类型（如 int64、struct）会在编译期报错。
type Code interface {
	~string | ~int32
}

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
//   - code: 错误码，只能是 string 或 int32（含具名类型，如 proto 生成的枚举）
//   - format: 错误描述格式字符串
//   - a: 格式参数
func New[T Code](code T, format string, a ...any) *MErr {
	return &MErr{
		code: codeToString(code),
		msg:  fmt.Sprintf(format, a...),
	}
}

// Wrap 包装一个已有错误，附加错误码
func Wrap[T Code](code T, err error) *MErr {
	return &MErr{
		code: codeToString(code),
		msg:  err.Error(),
		err:  err,
	}
}

// Wrapf 包装一个已有错误，附加错误码和自定义消息
func Wrapf[T Code](code T, err error, format string, a ...any) *MErr {
	return &MErr{
		code: codeToString(code),
		msg:  fmt.Sprintf(format, a...),
		err:  err,
	}
}

// Preset 使用通过 RegisterCode 预先注册的错误码消息创建错误
// 常用于兼容 proto 生成的枚举数字错误码：只需注册一次消息，之后按码直接构造错误
func Preset[T Code](code T) *MErr {
	c := codeToString(code)
	msg, _ := lookupPreset(c)
	return &MErr{code: c, msg: msg}
}

// Error 实现标准 error 接口
// 若该错误码通过 RegisterCode 注册了预设消息，会自动拼接到最终消息中
func (e *MErr) Error() string {
	msg := e.msg
	if preset, ok := lookupPreset(e.code); ok && preset != "" && preset != msg {
		if msg != "" {
			msg = fmt.Sprintf("%s: %s", preset, msg)
		} else {
			msg = preset
		}
	}
	if e.err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.code, msg, e.err)
	}
	return fmt.Sprintf("[%s] %s", e.code, msg)
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
// 支持标准 error 和 CodedError，code 也可以直接传入 proto 生成的枚举错误码
func IsCode[T Code](err error, code T) bool {
	if err == nil {
		return false
	}
	var ce *MErr
	if errors.As(err, &ce) {
		return ce.code == codeToString(code)
	}
	return false
}

// ========== 预设错误码消息注册 ==========

var (
	presetMu  sync.RWMutex
	presetReg = map[string]string{}
)

// RegisterCode 注册预设错误码及对应的消息
// 常用于将 proto 生成的枚举错误码统一注册消息，之后可直接通过 New/Preset 按码构造错误，
// 无需每次手写消息文案；Error() 输出时会自动拼接该预设消息。
// 重复注册同一错误码会覆盖之前的消息。
func RegisterCode[T Code](code T, message string) {
	c := codeToString(code)
	presetMu.Lock()
	presetReg[c] = message
	presetMu.Unlock()
}

// RegisterCodes 批量注册预设错误码及消息
func RegisterCodes[T Code](codeMessages map[T]string) {
	presetMu.Lock()
	for c, msg := range codeMessages {
		presetReg[codeToString(c)] = msg
	}
	presetMu.Unlock()
}

// lookupPreset 查找错误码对应的预设消息
func lookupPreset(code string) (string, bool) {
	presetMu.RLock()
	msg, ok := presetReg[code]
	presetMu.RUnlock()
	return msg, ok
}

// codeToString 将 string 或 int32（含具名类型）错误码归一化为字符串
// 数字类型直接取其数值，以保留原始数字错误码，便于与 proto 定义保持一致；
// 消息文案统一通过 RegisterCode 维护。
func codeToString[T Code](code T) string {
	if s, ok := any(code).(string); ok {
		return s
	}
	return strconv.FormatInt(reflect.ValueOf(code).Int(), 10)
}
