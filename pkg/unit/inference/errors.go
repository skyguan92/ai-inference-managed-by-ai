package inference

import "github.com/jguan/ai-inference-managed-by-ai/pkg/unit"

// Inference domain errors
var (
	// Resource errors
	ErrInferenceModelNotLoaded = unit.NewDomainError("inference", unit.ErrCodeInferenceModelNotLoaded, "model not loaded")

	// Operation errors
	ErrInferenceEngineError = unit.NewDomainError("inference", unit.ErrCodeInferenceEngineError, "inference engine error")
	ErrInferenceTimeout     = unit.NewDomainError("inference", unit.ErrCodeInferenceTimeout, "inference timeout")
	ErrInferenceRateLimited = unit.NewDomainError("inference", unit.ErrCodeInferenceRateLimited, "inference rate limited")

	// Input errors (backward compatibility)
	ErrInvalidInput      = unit.NewError(unit.ErrCodeInvalidInput, "invalid input")
	ErrInvalidParams     = unit.NewError(unit.ErrCodeInferenceInvalidParams, "invalid inference parameters")
	ErrModelNotSpecified = unit.NewError(unit.ErrCodeInvalidInput, "model not specified")
	ErrInferenceFailed   = unit.NewError(unit.ErrCodeInternalError, "inference failed")
	ErrUnsupportedModel  = unit.NewError(unit.ErrCodeInvalidInput, "unsupported model type")
	ErrProviderNotSet    = unit.NewError(unit.ErrCodeInternalError, "provider not set")
)
