package engine

import "github.com/jguan/ai-inference-managed-by-ai/pkg/unit"

// Engine domain errors
var (
	// Resource errors
	ErrEngineNotFound       = unit.NewDomainError("engine", unit.ErrCodeEngineNotFound, "engine not found")
	ErrEngineAlreadyRunning = unit.NewDomainError("engine", unit.ErrCodeEngineAlreadyRunning, "engine already running")
	ErrEngineNotRunning     = unit.NewDomainError("engine", unit.ErrCodeEngineNotRunning, "engine not running")
	ErrEngineAlreadyExists  = unit.NewError(unit.ErrCodeAlreadyExists, "engine already exists")

	// Operation errors
	ErrEngineStartFailed   = unit.NewDomainError("engine", unit.ErrCodeEngineStartFailed, "engine start failed")
	ErrEngineStopFailed    = unit.NewDomainError("engine", unit.ErrCodeEngineStopFailed, "engine stop failed")
	ErrEngineInstallFailed = unit.NewDomainError("engine", unit.ErrCodeEngineInstallFailed, "engine install failed")

	// Input errors (backward compatibility)
	ErrInvalidInput      = unit.NewError(unit.ErrCodeInvalidInput, "invalid input")
	ErrInvalidEngineName = unit.NewError(unit.ErrCodeInvalidInput, "invalid engine name")
	ErrProviderNotSet    = unit.NewError(unit.ErrCodeInternalError, "provider not set")
)
