package gateway

import (
	"context"
)

type MCPPrompt struct {
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	Arguments   []MCPPromptArgument `json:"arguments,omitempty"`
}

type MCPPromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type MCPPromptsListResult struct {
	Prompts []MCPPrompt `json:"prompts"`
}

func (a *MCPAdapter) handlePromptsList(ctx context.Context, req *MCPRequest) *MCPResponse {
	prompts := a.ListPrompts()
	result := &MCPPromptsListResult{
		Prompts: prompts,
	}
	return a.successResponse(req.ID, result)
}

func (a *MCPAdapter) ListPrompts() []MCPPrompt {
	return []MCPPrompt{}
}
