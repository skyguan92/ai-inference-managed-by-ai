package unit

import (
	"errors"
	"fmt"
	"net/http"
)

// ErrorCode 错误码类型
type ErrorCode string

// 通用错误码 (000-099)
const (
	ErrCodeSuccess          ErrorCode = "00000"
	ErrCodeUnknown          ErrorCode = "00001"
	ErrCodeInvalidRequest   ErrorCode = "00002"
	ErrCodeUnauthorized     ErrorCode = "00003"
	ErrCodeNotFound         ErrorCode = "00004"
	ErrCodeAlreadyExists    ErrorCode = "00005"
	ErrCodeTimeout          ErrorCode = "00006"
	ErrCodeRateLimited      ErrorCode = "00007"
	ErrCodeInternalError    ErrorCode = "00008"
	ErrCodeInvalidInput     ErrorCode = "00009"
	ErrCodeValidationFailed ErrorCode = "00010"
)

// 模型领域错误码 (100-199)
const (
	ErrCodeModelNotFound      ErrorCode = "00100"
	ErrCodeModelAlreadyExists ErrorCode = "00101"
	ErrCodeModelPullFailed    ErrorCode = "00102"
	ErrCodeModelVerifyFailed  ErrorCode = "00103"
	ErrCodeModelImportFailed  ErrorCode = "00104"
	ErrCodeModelDeleteFailed  ErrorCode = "00105"
)

// 引擎领域错误码 (200-299)
const (
	ErrCodeEngineNotFound       ErrorCode = "00200"
	ErrCodeEngineAlreadyRunning ErrorCode = "00201"
	ErrCodeEngineNotRunning     ErrorCode = "00202"
	ErrCodeEngineStartFailed    ErrorCode = "00203"
	ErrCodeEngineStopFailed     ErrorCode = "00204"
	ErrCodeEngineInstallFailed  ErrorCode = "00205"
)

// 推理领域错误码 (300-399)
const (
	ErrCodeInferenceModelNotLoaded ErrorCode = "00300"
	ErrCodeInferenceInvalidParams  ErrorCode = "00301"
	ErrCodeInferenceEngineError    ErrorCode = "00302"
	ErrCodeInferenceTimeout        ErrorCode = "00303"
	ErrCodeInferenceRateLimited    ErrorCode = "00304"
)

// 资源领域错误码 (400-499)
const (
	ErrCodeResourceInsufficient     ErrorCode = "00400"
	ErrCodeResourceAllocationFailed ErrorCode = "00401"
	ErrCodeResourceSlotNotFound     ErrorCode = "00402"
	ErrCodeResourcePressureHigh     ErrorCode = "00403"
)

// 设备领域错误码 (500-599)
const (
	ErrCodeDeviceNotFound     ErrorCode = "00500"
	ErrCodeDeviceUnreachable  ErrorCode = "00501"
	ErrCodeDeviceMetricsError ErrorCode = "00502"
)

// 服务领域错误码 (600-699)
const (
	ErrCodeServiceNotFound    ErrorCode = "00600"
	ErrCodeServiceStartFailed ErrorCode = "00601"
	ErrCodeServiceScaleFailed ErrorCode = "00602"
)

// 应用领域错误码 (700-799)
const (
	ErrCodeAppNotFound    ErrorCode = "00700"
	ErrCodeAppStartFailed ErrorCode = "00701"
	ErrCodeAppOOM         ErrorCode = "00702"
)

// 管道领域错误码 (800-899)
const (
	ErrCodePipelineNotFound         ErrorCode = "00800"
	ErrCodePipelineValidationFailed ErrorCode = "00801"
	ErrCodePipelineStepFailed       ErrorCode = "00802"
)

// 告警领域错误码 (900-999)
const (
	ErrCodeAlertRuleNotFound ErrorCode = "00900"
)

// 远程领域错误码 (1000-1099)
const (
	ErrCodeRemoteNotEnabled ErrorCode = "01000"
	ErrCodeRemoteExecFailed ErrorCode = "01001"
)

// Catalog 领域错误码 (1100-1199)
const (
	ErrCodeRecipeNotFound      ErrorCode = "01100"
	ErrCodeRecipeAlreadyExists ErrorCode = "01101"
	ErrCodeRecipeInvalid       ErrorCode = "01102"
	ErrCodeRecipeApplyFailed   ErrorCode = "01103"
)

// Skill 领域错误码 (1200-1299)
const (
	ErrCodeSkillNotFound         ErrorCode = "01200"
	ErrCodeSkillAlreadyExists    ErrorCode = "01201"
	ErrCodeSkillInvalid          ErrorCode = "01202"
	ErrCodeBuiltinSkillImmutable ErrorCode = "01203"
)

// Agent 领域错误码 (1300-1399)
const (
	ErrCodeAgentNotEnabled      ErrorCode = "01300"
	ErrCodeAgentLLMError        ErrorCode = "01301"
	ErrCodeConversationNotFound ErrorCode = "01302"
)

// UnitError 统一的错误类型
type UnitError struct {
	Code    ErrorCode
	Domain  string
	Message string
	Details map[string]any
	Cause   error
}

// Error 实现 error 接口
func (e *UnitError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap 返回原始错误，用于 errors.Is 和 errors.As
func (e *UnitError) Unwrap() error {
	return e.Cause
}

// WithDetails 添加错误详情
func (e *UnitError) WithDetails(key string, value any) *UnitError {
	if e.Details == nil {
		e.Details = make(map[string]any)
	}
	e.Details[key] = value
	return e
}

// WithCause 设置原始错误
func (e *UnitError) WithCause(err error) *UnitError {
	e.Cause = err
	return e
}

// Is 实现 errors.Is 接口
func (e *UnitError) Is(target error) bool {
	t, ok := target.(*UnitError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// NewError 创建通用错误
func NewError(code ErrorCode, message string) *UnitError {
	return &UnitError{
		Code:    code,
		Message: message,
		Details: make(map[string]any),
	}
}

// NewDomainError 创建领域错误
func NewDomainError(domain string, code ErrorCode, message string) *UnitError {
	return &UnitError{
		Code:    code,
		Domain:  domain,
		Message: message,
		Details: make(map[string]any),
	}
}

// WrapError 包装现有错误
func WrapError(err error, code ErrorCode, message string) *UnitError {
	return &UnitError{
		Code:    code,
		Message: message,
		Cause:   err,
		Details: make(map[string]any),
	}
}

// WrapDomainError 包装领域错误
func WrapDomainError(err error, domain string, code ErrorCode, message string) *UnitError {
	return &UnitError{
		Code:    code,
		Domain:  domain,
		Message: message,
		Cause:   err,
		Details: make(map[string]any),
	}
}

// AsUnitError 将错误转换为 UnitError
func AsUnitError(err error) (*UnitError, bool) {
	if err == nil {
		return nil, false
	}
	var ue *UnitError
	if errors.As(err, &ue) {
		return ue, true
	}
	return nil, false
}

// ErrorToHTTPStatus 将错误码映射为 HTTP 状态码
func ErrorToHTTPStatus(code ErrorCode) int {
	switch code {
	case ErrCodeSuccess:
		return http.StatusOK
	case ErrCodeInvalidRequest, ErrCodeInvalidInput, ErrCodeValidationFailed:
		return http.StatusBadRequest
	case ErrCodeUnauthorized:
		return http.StatusUnauthorized
	case ErrCodeNotFound:
		return http.StatusNotFound
	case ErrCodeAlreadyExists:
		return http.StatusConflict
	case ErrCodeTimeout:
		return http.StatusRequestTimeout
	case ErrCodeRateLimited:
		return http.StatusTooManyRequests
	case ErrCodeModelNotFound, ErrCodeEngineNotFound, ErrCodeServiceNotFound,
		ErrCodeAppNotFound, ErrCodePipelineNotFound, ErrCodeAlertRuleNotFound,
		ErrCodeDeviceNotFound, ErrCodeResourceSlotNotFound,
		ErrCodeRecipeNotFound, ErrCodeSkillNotFound, ErrCodeConversationNotFound:
		return http.StatusNotFound
	case ErrCodeModelAlreadyExists, ErrCodeEngineAlreadyRunning,
		ErrCodeRecipeAlreadyExists, ErrCodeSkillAlreadyExists:
		return http.StatusConflict
	case ErrCodeRecipeInvalid, ErrCodeSkillInvalid, ErrCodeBuiltinSkillImmutable:
		return http.StatusBadRequest
	case ErrCodeAgentNotEnabled, ErrCodeAgentLLMError:
		return http.StatusServiceUnavailable
	case ErrCodeRecipeApplyFailed:
		return http.StatusInternalServerError
	case ErrCodeRemoteNotEnabled, ErrCodeRemoteExecFailed:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// IsNotFound 检查是否为资源未找到错误
func IsNotFound(err error) bool {
	if ue, ok := AsUnitError(err); ok {
		return ue.Code == ErrCodeNotFound ||
			ue.Code == ErrCodeModelNotFound ||
			ue.Code == ErrCodeEngineNotFound ||
			ue.Code == ErrCodeServiceNotFound ||
			ue.Code == ErrCodeAppNotFound ||
			ue.Code == ErrCodePipelineNotFound ||
			ue.Code == ErrCodeAlertRuleNotFound ||
			ue.Code == ErrCodeDeviceNotFound ||
			ue.Code == ErrCodeResourceSlotNotFound ||
			ue.Code == ErrCodeRecipeNotFound ||
			ue.Code == ErrCodeSkillNotFound ||
			ue.Code == ErrCodeConversationNotFound
	}
	return errors.Is(err, ErrNotFound)
}

// IsAlreadyExists 检查是否为资源已存在错误
func IsAlreadyExists(err error) bool {
	if ue, ok := AsUnitError(err); ok {
		return ue.Code == ErrCodeAlreadyExists ||
			ue.Code == ErrCodeModelAlreadyExists ||
			ue.Code == ErrCodeEngineAlreadyRunning ||
			ue.Code == ErrCodeRecipeAlreadyExists ||
			ue.Code == ErrCodeSkillAlreadyExists
	}
	return false
}

// IsTimeout 检查是否为超时错误
func IsTimeout(err error) bool {
	if ue, ok := AsUnitError(err); ok {
		return ue.Code == ErrCodeTimeout || ue.Code == ErrCodeInferenceTimeout
	}
	return errors.Is(err, ErrTimeout)
}

// IsRateLimited 检查是否为限流错误
func IsRateLimited(err error) bool {
	if ue, ok := AsUnitError(err); ok {
		return ue.Code == ErrCodeRateLimited || ue.Code == ErrCodeInferenceRateLimited
	}
	return errors.Is(err, ErrRateLimited)
}

// IsImmutable 检查是否为不可变资源错误
func IsImmutable(err error) bool {
	if ue, ok := AsUnitError(err); ok {
		return ue.Code == ErrCodeBuiltinSkillImmutable
	}
	return false
}

// Common errors for backward compatibility
var (
	// 通用错误
	ErrUnknown        = NewError(ErrCodeUnknown, "unknown error")
	ErrInvalidInput   = NewError(ErrCodeInvalidInput, "invalid input")
	ErrNotFound       = NewError(ErrCodeNotFound, "resource not found")
	ErrAlreadyExists  = NewError(ErrCodeAlreadyExists, "resource already exists")
	ErrTimeout        = NewError(ErrCodeTimeout, "operation timeout")
	ErrRateLimited    = NewError(ErrCodeRateLimited, "rate limited")
	ErrInternal       = NewError(ErrCodeInternalError, "internal error")
	ErrUnauthorized   = NewError(ErrCodeUnauthorized, "unauthorized")
	ErrValidation     = NewError(ErrCodeValidationFailed, "validation failed")
	ErrProviderNotSet = NewError(ErrCodeInternalError, "provider not set")
)
