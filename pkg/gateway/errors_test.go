package gateway

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrorInfo_Error(t *testing.T) {
	tests := []struct {
		name     string
		errInfo  *ErrorInfo
		expected string
	}{
		{
			name:     "without details",
			errInfo:  NewErrorInfo(ErrCodeInvalidRequest, "invalid input"),
			expected: "[INVALID_REQUEST] invalid input",
		},
		{
			name:     "with details",
			errInfo:  NewErrorInfoWithDetails(ErrCodeValidationFailed, "field required", map[string]string{"field": "name"}),
			expected: "[VALIDATION_FAILED] field required: map[field:name]",
		},
		{
			name:     "empty message",
			errInfo:  NewErrorInfo(ErrCodeInternalError, ""),
			expected: "[INTERNAL_ERROR] ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.errInfo.Error(); got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestNewErrorInfo(t *testing.T) {
	err := NewErrorInfo(ErrCodeUnitNotFound, "unit not found")

	if err.Code != ErrCodeUnitNotFound {
		t.Errorf("Code = %q, want %q", err.Code, ErrCodeUnitNotFound)
	}
	if err.Message != "unit not found" {
		t.Errorf("Message = %q, want %q", err.Message, "unit not found")
	}
	if err.Details != nil {
		t.Errorf("Details should be nil, got %v", err.Details)
	}
}

func TestNewErrorInfoWithDetails(t *testing.T) {
	details := map[string]any{"field": "model_id", "value": ""}
	err := NewErrorInfoWithDetails(ErrCodeValidationFailed, "validation failed", details)

	if err.Code != ErrCodeValidationFailed {
		t.Errorf("Code = %q, want %q", err.Code, ErrCodeValidationFailed)
	}
	if err.Message != "validation failed" {
		t.Errorf("Message = %q, want %q", err.Message, "validation failed")
	}
	if err.Details == nil {
		t.Error("Details should not be nil")
	}
}

func TestToErrorInfo(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		if got := ToErrorInfo(nil); got != nil {
			t.Errorf("ToErrorInfo(nil) = %v, want nil", got)
		}
	})

	t.Run("ErrorInfo pointer", func(t *testing.T) {
		original := NewErrorInfo(ErrCodeTimeout, "request timeout")
		got := ToErrorInfo(original)

		if got != original {
			t.Errorf("ToErrorInfo should return same pointer for *ErrorInfo")
		}
	})

	t.Run("standard error", func(t *testing.T) {
		stdErr := errors.New("something went wrong")
		got := ToErrorInfo(stdErr)

		if got.Code != ErrCodeInternalError {
			t.Errorf("Code = %q, want %q", got.Code, ErrCodeInternalError)
		}
		if got.Message != "something went wrong" {
			t.Errorf("Message = %q, want %q", got.Message, "something went wrong")
		}
	})
}

func TestErrorInfo_JSON(t *testing.T) {
	t.Run("without details", func(t *testing.T) {
		err := NewErrorInfo(ErrCodeRateLimited, "rate limit exceeded")
		got := err.JSON()
		expected := `{"code":"RATE_LIMITED","message":"rate limit exceeded"}`
		if got != expected {
			t.Errorf("JSON() = %q, want %q", got, expected)
		}
	})

	t.Run("with details", func(t *testing.T) {
		err := NewErrorInfoWithDetails(ErrCodeValidationFailed, "invalid field", map[string]string{"field": "name"})
		got := err.JSON()

		if got == "" {
			t.Error("JSON() should not be empty")
		}
	})
}

func TestErrorInfo_Is(t *testing.T) {
	t.Run("same code", func(t *testing.T) {
		err1 := NewErrorInfo(ErrCodeTimeout, "timeout 1")
		err2 := NewErrorInfo(ErrCodeTimeout, "timeout 2")

		if !errors.Is(err1, err2) {
			t.Error("errors.Is should return true for same error codes")
		}
	})

	t.Run("different code", func(t *testing.T) {
		err1 := NewErrorInfo(ErrCodeTimeout, "timeout")
		err2 := NewErrorInfo(ErrCodeInternalError, "internal error")

		if errors.Is(err1, err2) {
			t.Error("errors.Is should return false for different error codes")
		}
	})

	t.Run("non ErrorInfo target", func(t *testing.T) {
		err := NewErrorInfo(ErrCodeTimeout, "timeout")
		stdErr := errors.New("standard error")

		if errors.Is(err, stdErr) {
			t.Error("errors.Is should return false for non-ErrorInfo target")
		}
	})
}

func TestErrorCodeConstants(t *testing.T) {
	codes := map[string]string{
		"ErrCodeInvalidRequest":   ErrCodeInvalidRequest,
		"ErrCodeUnitNotFound":     ErrCodeUnitNotFound,
		"ErrCodeResourceNotFound": ErrCodeResourceNotFound,
		"ErrCodeValidationFailed": ErrCodeValidationFailed,
		"ErrCodeExecutionFailed":  ErrCodeExecutionFailed,
		"ErrCodeTimeout":          ErrCodeTimeout,
		"ErrCodeUnauthorized":     ErrCodeUnauthorized,
		"ErrCodeRateLimited":      ErrCodeRateLimited,
		"ErrCodeInternalError":    ErrCodeInternalError,
	}

	expected := map[string]string{
		"ErrCodeInvalidRequest":   "INVALID_REQUEST",
		"ErrCodeUnitNotFound":     "UNIT_NOT_FOUND",
		"ErrCodeResourceNotFound": "RESOURCE_NOT_FOUND",
		"ErrCodeValidationFailed": "VALIDATION_FAILED",
		"ErrCodeExecutionFailed":  "EXECUTION_FAILED",
		"ErrCodeTimeout":          "TIMEOUT",
		"ErrCodeUnauthorized":     "UNAUTHORIZED",
		"ErrCodeRateLimited":      "RATE_LIMITED",
		"ErrCodeInternalError":    "INTERNAL_ERROR",
	}

	for name, code := range codes {
		if code != expected[name] {
			t.Errorf("%s = %q, want %q", name, code, expected[name])
		}
	}
}

func TestErrorInfoImplementsErrorInterface(t *testing.T) {
	var err error = NewErrorInfo(ErrCodeInternalError, "test")
	if err == nil {
		t.Error("ErrorInfo should implement error interface")
	}

	_ = fmt.Sprintf("%s", err)
}
