package unit

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestNewError(t *testing.T) {
	err := NewError(ErrCodeInvalidRequest, "invalid request")
	if err.Code != ErrCodeInvalidRequest {
		t.Errorf("Code = %q, want %q", err.Code, ErrCodeInvalidRequest)
	}
	if err.Message != "invalid request" {
		t.Errorf("Message = %q, want %q", err.Message, "invalid request")
	}
	if err.Details == nil {
		t.Error("Details should not be nil")
	}
}

func TestNewDomainError(t *testing.T) {
	err := NewDomainError("model", ErrCodeModelNotFound, "model not found")
	if err.Code != ErrCodeModelNotFound {
		t.Errorf("Code = %q, want %q", err.Code, ErrCodeModelNotFound)
	}
	if err.Domain != "model" {
		t.Errorf("Domain = %q, want %q", err.Domain, "model")
	}
	if err.Message != "model not found" {
		t.Errorf("Message = %q, want %q", err.Message, "model not found")
	}
}

func TestWrapError(t *testing.T) {
	original := errors.New("original error")
	wrapped := WrapError(original, ErrCodeInternalError, "wrapped")

	if wrapped.Cause != original {
		t.Error("Cause should be the original error")
	}
	if !errors.Is(wrapped, original) {
		t.Error("errors.Is should return true for the original error")
	}
}

func TestUnitError_Error(t *testing.T) {
	t.Run("without cause", func(t *testing.T) {
		err := NewError(ErrCodeNotFound, "not found")
		expected := "[00004] not found"
		if got := err.Error(); got != expected {
			t.Errorf("Error() = %q, want %q", got, expected)
		}
	})

	t.Run("with cause", func(t *testing.T) {
		cause := errors.New("underlying error")
		err := WrapError(cause, ErrCodeInternalError, "operation failed")
		expected := "[00008] operation failed: underlying error"
		if got := err.Error(); got != expected {
			t.Errorf("Error() = %q, want %q", got, expected)
		}
	})
}

func TestUnitError_WithDetails(t *testing.T) {
	err := NewError(ErrCodeValidationFailed, "validation failed").
		WithDetails("field", "name").
		WithDetails("reason", "required")

	if err.Details["field"] != "name" {
		t.Errorf("Details[field] = %v, want %v", err.Details["field"], "name")
	}
	if err.Details["reason"] != "required" {
		t.Errorf("Details[reason] = %v, want %v", err.Details["reason"], "required")
	}
}

func TestUnitError_Is(t *testing.T) {
	err1 := NewError(ErrCodeTimeout, "timeout 1")
	err2 := NewError(ErrCodeTimeout, "timeout 2")
	err3 := NewError(ErrCodeNotFound, "not found")

	if !err1.Is(err2) {
		t.Error("Same code errors should be equal")
	}
	if err1.Is(err3) {
		t.Error("Different code errors should not be equal")
	}

	if !errors.Is(err1, err2) {
		t.Error("errors.Is should return true for same code errors")
	}
}

func TestAsUnitError(t *testing.T) {
	t.Run("with UnitError", func(t *testing.T) {
		original := NewError(ErrCodeNotFound, "not found")
		ue, ok := AsUnitError(original)
		if !ok {
			t.Error("AsUnitError should return true for UnitError")
		}
		if ue.Code != ErrCodeNotFound {
			t.Errorf("Code = %q, want %q", ue.Code, ErrCodeNotFound)
		}
	})

	t.Run("with standard error", func(t *testing.T) {
		stdErr := errors.New("standard error")
		ue, ok := AsUnitError(stdErr)
		if ok {
			t.Error("AsUnitError should return false for standard error")
		}
		if ue != nil {
			t.Error("AsUnitError should return nil for standard error")
		}
	})

	t.Run("with nil error", func(t *testing.T) {
		ue, ok := AsUnitError(nil)
		if ok {
			t.Error("AsUnitError should return false for nil error")
		}
		if ue != nil {
			t.Error("AsUnitError should return nil for nil error")
		}
	})

	t.Run("with wrapped UnitError", func(t *testing.T) {
		original := NewError(ErrCodeNotFound, "not found")
		wrapped := fmt.Errorf("wrapped: %w", original)
		ue, ok := AsUnitError(wrapped)
		if !ok {
			t.Error("AsUnitError should return true for wrapped UnitError")
		}
		if ue.Code != ErrCodeNotFound {
			t.Errorf("Code = %q, want %q", ue.Code, ErrCodeNotFound)
		}
	})
}

func TestErrorToHTTPStatus(t *testing.T) {
	tests := []struct {
		code     ErrorCode
		expected int
	}{
		{ErrCodeSuccess, http.StatusOK},
		{ErrCodeInvalidRequest, http.StatusBadRequest},
		{ErrCodeInvalidInput, http.StatusBadRequest},
		{ErrCodeValidationFailed, http.StatusBadRequest},
		{ErrCodeUnauthorized, http.StatusUnauthorized},
		{ErrCodeNotFound, http.StatusNotFound},
		{ErrCodeAlreadyExists, http.StatusConflict},
		{ErrCodeTimeout, http.StatusRequestTimeout},
		{ErrCodeRateLimited, http.StatusTooManyRequests},
		{ErrCodeModelNotFound, http.StatusNotFound},
		{ErrCodeEngineAlreadyRunning, http.StatusConflict},
		{ErrCodeRemoteNotEnabled, http.StatusServiceUnavailable},
		{ErrCodeUnknown, http.StatusInternalServerError},
		{ErrCodeInternalError, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			got := ErrorToHTTPStatus(tt.code)
			if got != tt.expected {
				t.Errorf("ErrorToHTTPStatus(%q) = %d, want %d", tt.code, got, tt.expected)
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
		{"UnitError NotFound", NewError(ErrCodeNotFound, ""), true},
		{"UnitError ModelNotFound", NewDomainError("model", ErrCodeModelNotFound, ""), true},
		{"UnitError Other", NewError(ErrCodeInvalidRequest, ""), false},
		{"Standard error", errors.New("not found"), false},
		{"Nil error", nil, false},
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
		{"AlreadyExists", NewError(ErrCodeAlreadyExists, ""), true},
		{"ModelAlreadyExists", NewDomainError("model", ErrCodeModelAlreadyExists, ""), true},
		{"Other", NewError(ErrCodeInvalidRequest, ""), false},
		{"Standard error", errors.New("already exists"), false},
		{"Nil error", nil, false},
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
		{"Timeout", NewError(ErrCodeTimeout, ""), true},
		{"InferenceTimeout", NewDomainError("inference", ErrCodeInferenceTimeout, ""), true},
		{"Other", NewError(ErrCodeInvalidRequest, ""), false},
		{"Standard error", errors.New("timeout"), false},
		{"Nil error", nil, false},
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
		{"RateLimited", NewError(ErrCodeRateLimited, ""), true},
		{"InferenceRateLimited", NewDomainError("inference", ErrCodeInferenceRateLimited, ""), true},
		{"Other", NewError(ErrCodeInvalidRequest, ""), false},
		{"Standard error", errors.New("rate limited"), false},
		{"Nil error", nil, false},
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

func TestErrorCodeConstants(t *testing.T) {
	tests := []struct {
		name  string
		code  ErrorCode
		value string
	}{
		{"ErrCodeSuccess", ErrCodeSuccess, "00000"},
		{"ErrCodeUnknown", ErrCodeUnknown, "00001"},
		{"ErrCodeInvalidRequest", ErrCodeInvalidRequest, "00002"},
		{"ErrCodeUnauthorized", ErrCodeUnauthorized, "00003"},
		{"ErrCodeNotFound", ErrCodeNotFound, "00004"},
		{"ErrCodeAlreadyExists", ErrCodeAlreadyExists, "00005"},
		{"ErrCodeTimeout", ErrCodeTimeout, "00006"},
		{"ErrCodeRateLimited", ErrCodeRateLimited, "00007"},
		{"ErrCodeInternalError", ErrCodeInternalError, "00008"},
		{"ErrCodeInvalidInput", ErrCodeInvalidInput, "00009"},
		{"ErrCodeValidationFailed", ErrCodeValidationFailed, "00010"},
		{"ErrCodeModelNotFound", ErrCodeModelNotFound, "00100"},
		{"ErrCodeEngineNotFound", ErrCodeEngineNotFound, "00200"},
		{"ErrCodeInferenceTimeout", ErrCodeInferenceTimeout, "00303"},
		{"ErrCodeResourceInsufficient", ErrCodeResourceInsufficient, "00400"},
		{"ErrCodeDeviceNotFound", ErrCodeDeviceNotFound, "00500"},
		{"ErrCodeServiceNotFound", ErrCodeServiceNotFound, "00600"},
		{"ErrCodeAppNotFound", ErrCodeAppNotFound, "00700"},
		{"ErrCodePipelineNotFound", ErrCodePipelineNotFound, "00800"},
		{"ErrCodeAlertRuleNotFound", ErrCodeAlertRuleNotFound, "00900"},
		{"ErrCodeRemoteNotEnabled", ErrCodeRemoteNotEnabled, "01000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.code) != tt.value {
				t.Errorf("%s = %q, want %q", tt.name, tt.code, tt.value)
			}
		})
	}
}

func TestCommonErrors(t *testing.T) {
	if ErrUnknown.Code != ErrCodeUnknown {
		t.Errorf("ErrUnknown.Code = %q, want %q", ErrUnknown.Code, ErrCodeUnknown)
	}
	if ErrInvalidInput.Code != ErrCodeInvalidInput {
		t.Errorf("ErrInvalidInput.Code = %q, want %q", ErrInvalidInput.Code, ErrCodeInvalidInput)
	}
	if ErrNotFound.Code != ErrCodeNotFound {
		t.Errorf("ErrNotFound.Code = %q, want %q", ErrNotFound.Code, ErrCodeNotFound)
	}
	if ErrAlreadyExists.Code != ErrCodeAlreadyExists {
		t.Errorf("ErrAlreadyExists.Code = %q, want %q", ErrAlreadyExists.Code, ErrCodeAlreadyExists)
	}
	if ErrTimeout.Code != ErrCodeTimeout {
		t.Errorf("ErrTimeout.Code = %q, want %q", ErrTimeout.Code, ErrCodeTimeout)
	}
	if ErrRateLimited.Code != ErrCodeRateLimited {
		t.Errorf("ErrRateLimited.Code = %q, want %q", ErrRateLimited.Code, ErrCodeRateLimited)
	}
	if ErrInternal.Code != ErrCodeInternalError {
		t.Errorf("ErrInternal.Code = %q, want %q", ErrInternal.Code, ErrCodeInternalError)
	}
	if ErrUnauthorized.Code != ErrCodeUnauthorized {
		t.Errorf("ErrUnauthorized.Code = %q, want %q", ErrUnauthorized.Code, ErrCodeUnauthorized)
	}
	if ErrValidation.Code != ErrCodeValidationFailed {
		t.Errorf("ErrValidation.Code = %q, want %q", ErrValidation.Code, ErrCodeValidationFailed)
	}
	if ErrProviderNotSet.Code != ErrCodeInternalError {
		t.Errorf("ErrProviderNotSet.Code = %q, want %q", ErrProviderNotSet.Code, ErrCodeInternalError)
	}
}

func TestUnitErrorImplementsError(t *testing.T) {
	var err error = NewError(ErrCodeInternalError, "test")
	if err == nil {
		t.Error("UnitError should implement error interface")
	}

	_ = fmt.Sprintf("%s", err)
}
