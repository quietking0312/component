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
