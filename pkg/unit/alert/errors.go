package alert

import "github.com/jguan/ai-inference-managed-by-ai/pkg/unit"

// Alert domain errors
var (
	// Resource errors
	ErrAlertRuleNotFound = unit.NewDomainError("alert", unit.ErrCodeAlertRuleNotFound, "alert rule not found")
	ErrRuleNotFound      = unit.NewError(unit.ErrCodeNotFound, "rule not found")
	ErrAlertNotFound     = unit.NewError(unit.ErrCodeNotFound, "alert not found")
	ErrRuleExists        = unit.NewError(unit.ErrCodeAlreadyExists, "rule already exists")

	// Input errors (backward compatibility)
	ErrInvalidInput    = unit.NewError(unit.ErrCodeInvalidInput, "invalid input")
	ErrInvalidSeverity = unit.NewError(unit.ErrCodeInvalidInput, "invalid severity")
	ErrInvalidRuleID   = unit.NewError(unit.ErrCodeInvalidInput, "invalid rule id")
	ErrInvalidAlertID  = unit.NewError(unit.ErrCodeInvalidInput, "invalid alert id")
)
