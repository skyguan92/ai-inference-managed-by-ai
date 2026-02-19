package model

import "github.com/jguan/ai-inference-managed-by-ai/pkg/unit"

// Model domain errors
var (
	// Resource errors
	ErrModelNotFound      = unit.NewDomainError("model", unit.ErrCodeModelNotFound, "model not found")
	ErrModelAlreadyExists = unit.NewDomainError("model", unit.ErrCodeModelAlreadyExists, "model already exists")

	// Operation errors
	ErrModelPullFailed   = unit.NewDomainError("model", unit.ErrCodeModelPullFailed, "model pull failed")
	ErrModelVerifyFailed = unit.NewDomainError("model", unit.ErrCodeModelVerifyFailed, "model verify failed")
	ErrModelImportFailed = unit.NewDomainError("model", unit.ErrCodeModelImportFailed, "model import failed")
	ErrModelDeleteFailed = unit.NewDomainError("model", unit.ErrCodeModelDeleteFailed, "model delete failed")

	// Input errors (backward compatibility)
	ErrInvalidModelID = unit.NewError(unit.ErrCodeInvalidInput, "invalid model id")
	ErrInvalidInput   = unit.NewError(unit.ErrCodeInvalidInput, "invalid input")
	ErrPullInProgress = unit.NewError(unit.ErrCodeAlreadyExists, "pull already in progress")
	ErrProviderNotSet = unit.NewError(unit.ErrCodeInternalError, "provider not set")
)
