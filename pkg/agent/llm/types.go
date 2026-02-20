// Package llm provides LLM client abstractions for the Agent domain.
package llm

import "context"

// LLMClient abstracts communication with different LLM providers.
type LLMClient interface {
	// Chat sends a conversation and optional tool definitions to the LLM.
	// It returns the model's response, which may include tool calls.
	Chat(ctx context.Context, messages []Message, tools []ToolDef, opts ChatOptions) (*ChatResponse, error)
	// Name returns the provider name (e.g. "anthropic", "openai", "ollama").
	Name() string
	// ModelName returns the specific model identifier being used.
	ModelName() string
}

// Message represents a single turn in a conversation.
type Message struct {
	Role       string     `json:"role"`                  // "system", "user", "assistant", "tool"
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"` // set when Role == "tool"
}

// ToolCall represents a tool invocation requested by the LLM.
type ToolCall struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// ToolDef describes a tool available to the LLM.
type ToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

// ChatOptions holds optional parameters for a Chat call.
type ChatOptions struct {
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
}

// Usage holds token consumption data from a Chat call.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ChatResponse holds the LLM's reply to a Chat call.
type ChatResponse struct {
	Message   Message    `json:"message"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Usage     Usage      `json:"usage"`
}
