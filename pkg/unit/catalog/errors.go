package catalog

import "github.com/jguan/ai-inference-managed-by-ai/pkg/unit"

// Catalog domain errors.
var (
	ErrRecipeNotFound      = unit.NewDomainError("catalog", unit.ErrCodeRecipeNotFound, "recipe not found")
	ErrRecipeAlreadyExists = unit.NewDomainError("catalog", unit.ErrCodeRecipeAlreadyExists, "recipe already exists")
	ErrRecipeInvalid       = unit.NewDomainError("catalog", unit.ErrCodeRecipeInvalid, "recipe is invalid")
	ErrRecipeApplyFailed   = unit.NewDomainError("catalog", unit.ErrCodeRecipeApplyFailed, "recipe apply failed")

	ErrInvalidInput   = unit.NewError(unit.ErrCodeInvalidInput, "invalid input")
	ErrProviderNotSet = unit.NewError(unit.ErrCodeInternalError, "provider not set")
)
