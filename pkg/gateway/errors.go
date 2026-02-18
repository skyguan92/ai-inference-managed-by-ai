package gateway

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

const (
	ErrCodeInvalidRequest   = "INVALID_REQUEST"
	ErrCodeUnitNotFound     = "UNIT_NOT_FOUND"
	ErrCodeResourceNotFound = "RESOURCE_NOT_FOUND"
	ErrCodeValidationFailed = "VALIDATION_FAILED"
	ErrCodeExecutionFailed  = "EXECUTION_FAILED"
	ErrCodeTimeout          = "TIMEOUT"
	ErrCodeUnauthorized     = "UNAUTHORIZED"
	ErrCodeRateLimited      = "RATE_LIMITED"
	ErrCodeInternalError    = "INTERNAL_ERROR"
)

type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func NewErrorInfo(code string, message string) *ErrorInfo {
	return &ErrorInfo{
		Code:    code,
		Message: message,
	}
}

func NewErrorInfoWithDetails(code string, message string, details any) *ErrorInfo {
	return &ErrorInfo{
		Code:    code,
		Message: message,
		Details: details,
	}
}

// ToErrorInfo converts any error to ErrorInfo
// It supports UnitError from pkg/unit and creates appropriate error responses
func ToErrorInfo(err error) *ErrorInfo {
	if err == nil {
		return nil
	}

	// Check if it's already an ErrorInfo
	if ei, ok := err.(*ErrorInfo); ok {
		return ei
	}

	// Check if it's a UnitError
	if ue, ok := unit.AsUnitError(err); ok {
		return &ErrorInfo{
			Code:    string(ue.Code),
			Message: ue.Message,
			Details: ue.Details,
		}
	}

	// Default: treat as internal error
	return &ErrorInfo{
		Code:    ErrCodeInternalError,
		Message: err.Error(),
	}
}

func (e *ErrorInfo) Error() string {
	if e.Details != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *ErrorInfo) JSON() string {
	b, _ := json.Marshal(e)
	return string(b)
}

func (e *ErrorInfo) Is(target error) bool {
	t, ok := target.(*ErrorInfo)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// ErrorCodeToHTTPStatus maps error codes to HTTP status codes
// It supports both legacy codes and new UnitError codes
func ErrorCodeToHTTPStatus(code string) int {
	// First try to map as UnitError code
	ueCode := unit.ErrorCode(code)
	status := unit.ErrorToHTTPStatus(ueCode)
	if status != http.StatusInternalServerError || ueCode == unit.ErrCodeInternalError {
		return status
	}

	// Fallback to legacy code mapping
	switch code {
	case ErrCodeInvalidRequest:
		return http.StatusBadRequest
	case ErrCodeUnitNotFound, ErrCodeResourceNotFound:
		return http.StatusNotFound
	case ErrCodeValidationFailed:
		return http.StatusBadRequest
	case ErrCodeExecutionFailed:
		return http.StatusInternalServerError
	case ErrCodeTimeout:
		return http.StatusRequestTimeout
	case ErrCodeUnauthorized:
		return http.StatusUnauthorized
	case ErrCodeRateLimited:
		return http.StatusTooManyRequests
	case ErrCodeInternalError:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// GetHTTPStatus returns the appropriate HTTP status code for an error
func GetHTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}

	// Try to extract UnitError code
	if ue, ok := unit.AsUnitError(err); ok {
		return unit.ErrorToHTTPStatus(ue.Code)
	}

	// Try to extract ErrorInfo code
	if ei, ok := err.(*ErrorInfo); ok {
		return ErrorCodeToHTTPStatus(ei.Code)
	}

	// Default to internal server error
	return http.StatusInternalServerError
}

// IsNotFound checks if an error is a "not found" error
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}

	// Check UnitError
	if unit.IsNotFound(err) {
		return true
	}

	// Check ErrorInfo
	if ei, ok := err.(*ErrorInfo); ok {
		return ei.Code == ErrCodeUnitNotFound || ei.Code == ErrCodeResourceNotFound ||
			ei.Code == string(unit.ErrCodeNotFound) || ei.Code == string(unit.ErrCodeModelNotFound)
	}

	return errors.Is(err, unit.ErrNotFound)
}

// IsAlreadyExists checks if an error is an "already exists" error
func IsAlreadyExists(err error) bool {
	if err == nil {
		return false
	}

	// Check UnitError
	if unit.IsAlreadyExists(err) {
		return true
	}

	return false
}

// IsTimeout checks if an error is a timeout error
func IsTimeout(err error) bool {
	if err == nil {
		return false
	}

	// Check UnitError
	if unit.IsTimeout(err) {
		return true
	}

	// Check ErrorInfo
	if ei, ok := err.(*ErrorInfo); ok {
		return ei.Code == ErrCodeTimeout || ei.Code == string(unit.ErrCodeTimeout)
	}

	return errors.Is(err, unit.ErrTimeout)
}

// IsRateLimited checks if an error is a rate limit error
func IsRateLimited(err error) bool {
	if err == nil {
		return false
	}

	// Check UnitError
	if unit.IsRateLimited(err) {
		return true
	}

	// Check ErrorInfo
	if ei, ok := err.(*ErrorInfo); ok {
		return ei.Code == ErrCodeRateLimited || ei.Code == string(unit.ErrCodeRateLimited)
	}

	return errors.Is(err, unit.ErrRateLimited)
}

// NewUnitError creates a UnitError and converts it to ErrorInfo
func NewUnitError(code unit.ErrorCode, message string) *ErrorInfo {
	return ToErrorInfo(unit.NewError(code, message))
}

// WrapError wraps an error with additional context and converts to ErrorInfo
func WrapError(err error, message string) *ErrorInfo {
	if err == nil {
		return nil
	}

	wrapped := fmt.Errorf("%s: %w", message, err)
	return ToErrorInfo(wrapped)
}

// Common error constructors
func NewInvalidRequestError(message string) *ErrorInfo {
	return NewErrorInfo(ErrCodeInvalidRequest, message)
}

func NewNotFoundError(resource string) *ErrorInfo {
	return NewErrorInfo(ErrCodeResourceNotFound, fmt.Sprintf("%s not found", resource))
}

func NewValidationError(message string, details any) *ErrorInfo {
	return NewErrorInfoWithDetails(ErrCodeValidationFailed, message, details)
}

func NewTimeoutError(message string) *ErrorInfo {
	return NewErrorInfo(ErrCodeTimeout, message)
}

func NewUnauthorizedError(message string) *ErrorInfo {
	return NewErrorInfo(ErrCodeUnauthorized, message)
}

func NewRateLimitedError(message string) *ErrorInfo {
	return NewErrorInfo(ErrCodeRateLimited, message)
}

func NewInternalError(message string) *ErrorInfo {
	return NewErrorInfo(ErrCodeInternalError, message)
}
