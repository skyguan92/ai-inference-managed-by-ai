package service

import "github.com/jguan/ai-inference-managed-by-ai/pkg/unit"

// Service domain errors
var (
	// Resource errors
	ErrServiceNotFound       = unit.NewDomainError("service", unit.ErrCodeServiceNotFound, "service not found")
	ErrServiceAlreadyExists  = unit.NewError(unit.ErrCodeAlreadyExists, "service already exists")
	ErrServiceNotRunning     = unit.NewError(unit.ErrCodeInternalError, "service not running")
	ErrServiceAlreadyRunning = unit.NewError(unit.ErrCodeAlreadyExists, "service already running")

	// Operation errors
	ErrServiceStartFailed = unit.NewDomainError("service", unit.ErrCodeServiceStartFailed, "service start failed")
	ErrServiceScaleFailed = unit.NewDomainError("service", unit.ErrCodeServiceScaleFailed, "service scale failed")

	// Input errors (backward compatibility)
	ErrInvalidInput     = unit.NewError(unit.ErrCodeInvalidInput, "invalid input")
	ErrProviderNotSet   = unit.NewError(unit.ErrCodeInternalError, "service provider not set")
	ErrInvalidServiceID = unit.NewError(unit.ErrCodeInvalidInput, "invalid service id")
)
