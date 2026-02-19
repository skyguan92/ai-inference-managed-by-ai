package app

import "github.com/jguan/ai-inference-managed-by-ai/pkg/unit"

// App domain errors
var (
	// Resource errors
	ErrAppNotFound       = unit.NewDomainError("app", unit.ErrCodeAppNotFound, "app not found")
	ErrAppAlreadyExists  = unit.NewError(unit.ErrCodeAlreadyExists, "app already exists")
	ErrAppNotRunning     = unit.NewError(unit.ErrCodeInternalError, "app not running")
	ErrAppAlreadyRunning = unit.NewError(unit.ErrCodeAlreadyExists, "app already running")

	// Operation errors
	ErrAppStartFailed = unit.NewDomainError("app", unit.ErrCodeAppStartFailed, "app start failed")
	ErrAppOOM         = unit.NewDomainError("app", unit.ErrCodeAppOOM, "app out of memory")

	// Input errors (backward compatibility)
	ErrInvalidInput     = unit.NewError(unit.ErrCodeInvalidInput, "invalid input")
	ErrInvalidAppID     = unit.NewError(unit.ErrCodeInvalidInput, "invalid app id")
	ErrProviderNotSet   = unit.NewError(unit.ErrCodeInternalError, "provider not set")
	ErrTemplateNotFound = unit.NewError(unit.ErrCodeNotFound, "template not found")
)
