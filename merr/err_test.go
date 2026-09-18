package merr

import (
	"errors"
	"fmt"
	"testing"
)

func TestMErr_Basic(t *testing.T) {
	err := New("ERR_TEST", "something went wrong: %s", "detail")
	if err.Code() != "ERR_TEST" {
		t.Fatalf("expected code ERR_TEST, got %s", err.Code())
	}
	if err.Error() != "[ERR_TEST] something went wrong: detail" {
		t.Fatalf("unexpected error message: %s", err.Error())
	}
}

func TestMErr_Wrap(t *testing.T) {
	inner := errors.New("database connection failed")
	err := Wrap(CodeInternalError, inner)

	if !errors.Is(err, inner) {
		t.Fatal("expected errors.Is to match wrapped error")
	}
	if !IsCode(err, CodeInternalError) {
		t.Fatal("expected IsCode to match")
	}
}

func TestMErr_Predefined(t *testing.T) {
	err := NotFound("user %d not found", 42)
	if err.Code() != CodeNotFound {
		t.Fatalf("expected code %s, got %s", CodeNotFound, err.Code())
	}
	fmt.Println(err.Error())
}

// pbErrCode 模拟 proto 生成的数字枚举错误码类型
type pbErrCode int32

func (c pbErrCode) String() string {
	switch c {
	case 1001:
		return "USER_NOT_FOUND"
	default:
		return "UNKNOWN"
	}
}

const pbErrCodeUserNotFound pbErrCode = 1001

func TestMErr_RegisterCode(t *testing.T) {
	RegisterCode(pbErrCodeUserNotFound, "用户不存在")

	// 数字枚举错误码应归一化为其数值，而非 String() 返回的名字
	if got := codeToString(pbErrCodeUserNotFound); got != "1001" {
		t.Fatalf("expected code 1001, got %s", got)
	}

	err := Preset(pbErrCodeUserNotFound)
	if err.Code() != "1001" {
		t.Fatalf("expected code 1001, got %s", err.Code())
	}
	if err.Error() != "[1001] 用户不存在" {
		t.Fatalf("unexpected error message: %s", err.Error())
	}

	// New 附加自定义信息时，预设消息应自动拼接进去
	err2 := New(pbErrCodeUserNotFound, "id=%d", 42)
	if err2.Error() != "[1001] 用户不存在: id=42" {
		t.Fatalf("unexpected error message: %s", err2.Error())
	}

	if !IsCode(err2, pbErrCodeUserNotFound) {
		t.Fatal("expected IsCode to match numeric enum code")
	}
}

func TestMErr_Is(t *testing.T) {
	err1 := New("ERR_SAME", "first")
	err2 := New("ERR_SAME", "second")
	err3 := New("ERR_DIFF", "third")

	if !errors.Is(err1, err2) {
		t.Fatal("expected errors.Is to match same code")
	}
	if errors.Is(err1, err3) {
		t.Fatal("expected errors.Is to not match different code")
	}
}
