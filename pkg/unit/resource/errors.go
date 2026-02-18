package resource

import "github.com/jguan/ai-inference-managed-by-ai/pkg/unit"

// Resource domain errors
var (
	// Resource errors
	ErrResourceInsufficient     = unit.NewDomainError("resource", unit.ErrCodeResourceInsufficient, "insufficient resources")
	ErrResourceAllocationFailed = unit.NewDomainError("resource", unit.ErrCodeResourceAllocationFailed, "resource allocation failed")
	ErrResourceSlotNotFound     = unit.NewDomainError("resource", unit.ErrCodeResourceSlotNotFound, "resource slot not found")
	ErrResourcePressureHigh     = unit.NewDomainError("resource", unit.ErrCodeResourcePressureHigh, "resource pressure high")

	// Input errors (backward compatibility)
	ErrSlotNotFound       = unit.NewError(unit.ErrCodeResourceSlotNotFound, "slot not found")
	ErrInvalidSlotID      = unit.NewError(unit.ErrCodeInvalidInput, "invalid slot id")
	ErrInsufficientMemory = unit.NewError(unit.ErrCodeResourceInsufficient, "insufficient memory")
	ErrInvalidMemoryValue = unit.NewError(unit.ErrCodeInvalidInput, "invalid memory value")
	ErrProviderNotSet     = unit.NewError(unit.ErrCodeInternalError, "resource provider not set")
	ErrSlotAlreadyExists  = unit.NewError(unit.ErrCodeAlreadyExists, "slot already exists")
	ErrInvalidSlotType    = unit.NewError(unit.ErrCodeInvalidInput, "invalid slot type")
	ErrInvalidSlotStatus  = unit.NewError(unit.ErrCodeInvalidInput, "invalid slot status")
)
