package remote

import "github.com/jguan/ai-inference-managed-by-ai/pkg/unit"

// Remote domain errors
var (
	// Resource errors
	ErrRemoteNotEnabled     = unit.NewDomainError("remote", unit.ErrCodeRemoteNotEnabled, "remote not enabled")
	ErrTunnelNotFound       = unit.NewError(unit.ErrCodeNotFound, "tunnel not found")
	ErrTunnelAlreadyExists  = unit.NewError(unit.ErrCodeAlreadyExists, "tunnel already exists")
	ErrTunnelNotConnected   = unit.NewError(unit.ErrCodeInternalError, "tunnel not connected")
	ErrTunnelAlreadyEnabled = unit.NewError(unit.ErrCodeAlreadyExists, "tunnel already enabled")

	// Operation errors
	ErrRemoteExecFailed = unit.NewDomainError("remote", unit.ErrCodeRemoteExecFailed, "remote execution failed")

	// Input errors (backward compatibility)
	ErrInvalidInput   = unit.NewError(unit.ErrCodeInvalidInput, "invalid input")
	ErrProviderNotSet = unit.NewError(unit.ErrCodeInternalError, "remote provider not set")
)
