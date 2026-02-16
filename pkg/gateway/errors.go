package gateway

import (
	"encoding/json"
	"fmt"
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

func ToErrorInfo(err error) *ErrorInfo {
	if err == nil {
		return nil
	}

	if ei, ok := err.(*ErrorInfo); ok {
		return ei
	}

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
