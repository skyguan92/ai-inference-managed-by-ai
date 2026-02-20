package agent

import (
	"fmt"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

// Domain-level error constructors using the error codes defined in pkg/unit/errors.go.

func errAgentNotEnabled() error {
	return &unit.UnitError{
		Code:    unit.ErrCodeAgentNotEnabled,
		Message: "agent operator is not enabled (LLM client not configured)",
	}
}

func errAgentLLMError(cause error) error {
	return &unit.UnitError{
		Code:    unit.ErrCodeAgentLLMError,
		Message: fmt.Sprintf("LLM error: %s", cause),
	}
}

func errConversationNotFound(id string) error {
	return &unit.UnitError{
		Code:    unit.ErrCodeConversationNotFound,
		Message: fmt.Sprintf("conversation %q not found", id),
	}
}
