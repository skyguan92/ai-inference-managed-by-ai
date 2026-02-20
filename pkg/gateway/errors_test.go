package gateway

import (
	"errors"
	"net/http"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
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

	t.Run("UnitError", func(t *testing.T) {
		ue := unit.NewError(unit.ErrCodeNotFound, "model not found")
		got := ToErrorInfo(ue)

		if got.Code != string(unit.ErrCodeNotFound) {
			t.Errorf("Code = %q, want %q", got.Code, unit.ErrCodeNotFound)
		}
		if got.Message != "model not found" {
			t.Errorf("Message = %q, want %q", got.Message, "model not found")
		}
	})

	t.Run("UnitError with details", func(t *testing.T) {
		ue := unit.NewDomainError("model", unit.ErrCodeModelNotFound, "model xyz not found").
			WithDetails("model_id", "xyz")
		got := ToErrorInfo(ue)

		if got.Code != string(unit.ErrCodeModelNotFound) {
			t.Errorf("Code = %q, want %q", got.Code, unit.ErrCodeModelNotFound)
		}
		if got.Details == nil {
			t.Error("Details should not be nil")
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

func TestErrorCodeToHTTPStatus(t *testing.T) {
	tests := []struct {
		code     string
		expected int
	}{
		// Legacy codes
		{ErrCodeInvalidRequest, http.StatusBadRequest},
		{ErrCodeUnitNotFound, http.StatusNotFound},
		{ErrCodeResourceNotFound, http.StatusNotFound},
		{ErrCodeValidationFailed, http.StatusBadRequest},
		{ErrCodeExecutionFailed, http.StatusInternalServerError},
		{ErrCodeTimeout, http.StatusRequestTimeout},
		{ErrCodeUnauthorized, http.StatusUnauthorized},
		{ErrCodeRateLimited, http.StatusTooManyRequests},
		{ErrCodeInternalError, http.StatusInternalServerError},
		// UnitError codes
		{string(unit.ErrCodeSuccess), http.StatusOK},
		{string(unit.ErrCodeNotFound), http.StatusNotFound},
		{string(unit.ErrCodeModelNotFound), http.StatusNotFound},
		{string(unit.ErrCodeAlreadyExists), http.StatusConflict},
		{string(unit.ErrCodeTimeout), http.StatusRequestTimeout},
		{string(unit.ErrCodeRateLimited), http.StatusTooManyRequests},
		{string(unit.ErrCodeUnknown), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			got := ErrorCodeToHTTPStatus(tt.code)
			if got != tt.expected {
				t.Errorf("ErrorCodeToHTTPStatus(%q) = %d, want %d", tt.code, got, tt.expected)
			}
		})
	}
}

func TestGetHTTPStatus(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{"nil error", nil, http.StatusOK},
		{"UnitError", unit.NewError(unit.ErrCodeNotFound, ""), http.StatusNotFound},
		{"ErrorInfo", NewErrorInfo(ErrCodeInvalidRequest, ""), http.StatusBadRequest},
		{"standard error", errors.New("error"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetHTTPStatus(tt.err)
			if got != tt.expected {
				t.Errorf("GetHTTPStatus() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"UnitError NotFound", unit.NewError(unit.ErrCodeNotFound, ""), true},
		{"UnitError ModelNotFound", unit.NewDomainError("model", unit.ErrCodeModelNotFound, ""), true},
		{"ErrorInfo ResourceNotFound", NewErrorInfo(ErrCodeResourceNotFound, ""), true},
		{"ErrorInfo UnitNotFound", NewErrorInfo(ErrCodeUnitNotFound, ""), true},
		{"UnitError Other", unit.NewError(unit.ErrCodeInvalidRequest, ""), false},
		{"ErrorInfo Other", NewErrorInfo(ErrCodeInvalidRequest, ""), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNotFound(tt.err)
			if got != tt.expected {
				t.Errorf("IsNotFound() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsAlreadyExists(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"UnitError AlreadyExists", unit.NewError(unit.ErrCodeAlreadyExists, ""), true},
		{"UnitError ModelAlreadyExists", unit.NewDomainError("model", unit.ErrCodeModelAlreadyExists, ""), true},
		{"UnitError Other", unit.NewError(unit.ErrCodeInvalidRequest, ""), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAlreadyExists(tt.err)
			if got != tt.expected {
				t.Errorf("IsAlreadyExists() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsTimeout(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"UnitError Timeout", unit.NewError(unit.ErrCodeTimeout, ""), true},
		{"ErrorInfo Timeout", NewErrorInfo(ErrCodeTimeout, ""), true},
		{"UnitError InferenceTimeout", unit.NewDomainError("inference", unit.ErrCodeInferenceTimeout, ""), true},
		{"UnitError Other", unit.NewError(unit.ErrCodeInvalidRequest, ""), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTimeout(tt.err)
			if got != tt.expected {
				t.Errorf("IsTimeout() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsRateLimited(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"UnitError RateLimited", unit.NewError(unit.ErrCodeRateLimited, ""), true},
		{"ErrorInfo RateLimited", NewErrorInfo(ErrCodeRateLimited, ""), true},
		{"UnitError InferenceRateLimited", unit.NewDomainError("inference", unit.ErrCodeInferenceRateLimited, ""), true},
		{"UnitError Other", unit.NewError(unit.ErrCodeInvalidRequest, ""), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRateLimited(tt.err)
			if got != tt.expected {
				t.Errorf("IsRateLimited() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewUnitError(t *testing.T) {
	err := NewUnitError(unit.ErrCodeModelNotFound, "model not found")
	if err.Code != string(unit.ErrCodeModelNotFound) {
		t.Errorf("Code = %q, want %q", err.Code, unit.ErrCodeModelNotFound)
	}
	if err.Message != "model not found" {
		t.Errorf("Message = %q, want %q", err.Message, "model not found")
	}
}

func TestWrapError(t *testing.T) {
	t.Run("with error", func(t *testing.T) {
		original := errors.New("original")
		wrapped := WrapError(original, "wrapped")

		if wrapped == nil {
			t.Fatal("WrapError should not return nil for non-nil error")
		}
		if wrapped.Code != ErrCodeInternalError {
			t.Errorf("Code = %q, want %q", wrapped.Code, ErrCodeInternalError)
		}
	})

	t.Run("with nil error", func(t *testing.T) {
		wrapped := WrapError(nil, "wrapped")
		if wrapped != nil {
			t.Error("WrapError should return nil for nil error")
		}
	})
}

func TestCommonErrorConstructors(t *testing.T) {
	tests := []struct {
		name         string
		err          *ErrorInfo
		expectedCode string
	}{
		{"NewInvalidRequestError", NewInvalidRequestError("test"), ErrCodeInvalidRequest},
		{"NewNotFoundError", NewNotFoundError("model"), ErrCodeResourceNotFound},
		{"NewValidationError", NewValidationError("test", nil), ErrCodeValidationFailed},
		{"NewTimeoutError", NewTimeoutError("test"), ErrCodeTimeout},
		{"NewUnauthorizedError", NewUnauthorizedError("test"), ErrCodeUnauthorized},
		{"NewRateLimitedError", NewRateLimitedError("test"), ErrCodeRateLimited},
		{"NewInternalError", NewInternalError("test"), ErrCodeInternalError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.expectedCode {
				t.Errorf("Code = %q, want %q", tt.err.Code, tt.expectedCode)
			}
		})
	}
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
	// Verify that *ErrorInfo satisfies the error interface at compile time.
	var _ error = NewErrorInfo(ErrCodeInternalError, "test")

	ei := NewErrorInfo(ErrCodeInternalError, "test")
	_ = ei.Error()
}
