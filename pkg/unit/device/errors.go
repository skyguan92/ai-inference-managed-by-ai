package device

import "github.com/jguan/ai-inference-managed-by-ai/pkg/unit"

// Device domain errors
var (
	// Resource errors
	ErrDeviceNotFound     = unit.NewDomainError("device", unit.ErrCodeDeviceNotFound, "device not found")
	ErrDeviceUnreachable  = unit.NewDomainError("device", unit.ErrCodeDeviceUnreachable, "device unreachable")
	ErrDeviceMetricsError = unit.NewDomainError("device", unit.ErrCodeDeviceMetricsError, "device metrics error")

	// Input errors (backward compatibility)
	ErrInvalidInput      = unit.NewError(unit.ErrCodeInvalidInput, "invalid input")
	ErrInvalidDeviceID   = unit.NewError(unit.ErrCodeInvalidInput, "invalid device id")
	ErrInvalidPowerLimit = unit.NewError(unit.ErrCodeInvalidInput, "invalid power limit")
	ErrProviderNotSet    = unit.NewError(unit.ErrCodeInternalError, "device provider not set")
)
