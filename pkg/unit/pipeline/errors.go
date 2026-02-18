package pipeline

import "github.com/jguan/ai-inference-managed-by-ai/pkg/unit"

// Pipeline domain errors
var (
	// Resource errors
	ErrPipelineNotFound      = unit.NewDomainError("pipeline", unit.ErrCodePipelineNotFound, "pipeline not found")
	ErrPipelineAlreadyExists = unit.NewError(unit.ErrCodeAlreadyExists, "pipeline already exists")
	ErrRunNotFound           = unit.NewError(unit.ErrCodeNotFound, "run not found")

	// Operation errors
	ErrPipelineValidationFailed = unit.NewDomainError("pipeline", unit.ErrCodePipelineValidationFailed, "pipeline validation failed")
	ErrPipelineStepFailed       = unit.NewDomainError("pipeline", unit.ErrCodePipelineStepFailed, "pipeline step failed")
	ErrPipelineRunning          = unit.NewError(unit.ErrCodeInternalError, "pipeline is running")
	ErrRunNotCancellable        = unit.NewError(unit.ErrCodeInternalError, "run not cancellable")

	// Input errors (backward compatibility)
	ErrInvalidInput          = unit.NewError(unit.ErrCodeInvalidInput, "invalid input")
	ErrStoreNotSet           = unit.NewError(unit.ErrCodeInternalError, "store not set")
	ErrExecutorNotSet        = unit.NewError(unit.ErrCodeInternalError, "executor not set")
	ErrInvalidStepDependency = unit.NewError(unit.ErrCodeInvalidInput, "invalid step dependency")
)
