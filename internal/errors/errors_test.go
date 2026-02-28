package errors

import (
	"errors"
	"testing"
)

type customError struct {
	Msg string
}

func (e customError) Error() string { return e.Msg }

func TestNew(t *testing.T) {
	err := New("test error")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "test error" {
		t.Errorf("expected 'test error', got '%s'", err.Error())
	}
}

func TestWrap(t *testing.T) {
	baseErr := errors.New("base error")

	t.Run("wrap non-nil error", func(t *testing.T) {
		wrapped := Wrap(baseErr, "wrapped")
		if wrapped == nil {
			t.Fatal("expected wrapped error, got nil")
		}
		expected := "wrapped: base error"
		if wrapped.Error() != expected {
			t.Errorf("expected '%s', got '%s'", expected, wrapped.Error())
		}
		if !errors.Is(wrapped, baseErr) {
			t.Error("expected wrapped error to wrap baseErr")
		}
	})

	t.Run("wrap nil error", func(t *testing.T) {
		wrapped := Wrap(nil, "wrapped")
		if wrapped != nil {
			t.Errorf("expected nil, got %v", wrapped)
		}
	})
}

func TestWrapf(t *testing.T) {
	baseErr := errors.New("base error")

	t.Run("wrapf non-nil error", func(t *testing.T) {
		wrapped := Wrapf(baseErr, "wrapped %d", 123)
		if wrapped == nil {
			t.Fatal("expected wrapped error, got nil")
		}
		expected := "wrapped 123: base error"
		if wrapped.Error() != expected {
			t.Errorf("expected '%s', got '%s'", expected, wrapped.Error())
		}
		if !errors.Is(wrapped, baseErr) {
			t.Error("expected wrapped error to wrap baseErr")
		}
	})

	t.Run("wrapf nil error", func(t *testing.T) {
		wrapped := Wrapf(nil, "wrapped %d", 123)
		if wrapped != nil {
			t.Errorf("expected nil, got %v", wrapped)
		}
	})
}

func TestIs(t *testing.T) {
	if !Is(ErrNotFound, ErrNotFound) {
		t.Error("expected ErrNotFound to be ErrNotFound")
	}

	wrapped := Wrap(ErrNotFound, "context")
	if !Is(wrapped, ErrNotFound) {
		t.Error("expected wrapped ErrNotFound to be ErrNotFound")
	}

	if Is(ErrNotFound, ErrConflict) {
		t.Error("expected ErrNotFound NOT to be ErrConflict")
	}
}

func TestAs(t *testing.T) {
	custom := customError{Msg: "custom"}
	wrapped := Wrap(custom, "context")

	var target customError
	if !As(wrapped, &target) {
		t.Fatal("expected wrapped error to be able to extract target")
	}
	if target.Msg != "custom" {
		t.Errorf("expected 'custom', got '%s'", target.Msg)
	}
}

func TestStandardErrors(t *testing.T) {
	tests := []struct {
		err  error
		text string
	}{
		{ErrNotFound, "not found"},
		{ErrConflict, "conflict"},
		{ErrInvalidInput, "invalid input"},
		{ErrUnauthorized, "unauthorized"},
		{ErrForbidden, "forbidden"},
		{ErrLocked, "locked"},
	}

	for _, tt := range tests {
		if tt.err.Error() != tt.text {
			t.Errorf("expected text '%s' for error, got '%s'", tt.text, tt.err.Error())
		}
	}
}
