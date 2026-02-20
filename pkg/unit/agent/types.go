// Package agent implements the Agent domain atomic units.
// It wraps the pkg/agent orchestration layer, exposing agent operations
// as gateway-compatible commands and queries.
package agent

import (
	"time"

	agentllm "github.com/jguan/ai-inference-managed-by-ai/pkg/agent/llm"
)

// AgentStatus represents the runtime status of the agent operator.
type AgentStatus struct {
	Enabled             bool   `json:"enabled"`
	Provider            string `json:"provider"`
	Model               string `json:"model"`
	ActiveConversations int    `json:"active_conversations"`
}

// ConversationSummary is a lightweight view of a conversation.
type ConversationSummary struct {
	ID           string    `json:"id"`
	MessageCount int       `json:"message_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// MessageView is a JSON-serializable view of a conversation message.
type MessageView struct {
	Role       string `json:"role"`
	Content    string `json:"content"`
	ToolCallID string `json:"tool_call_id,omitempty"`
}

// messageToView converts an LLM message to a view struct.
func messageToView(m agentllm.Message) MessageView {
	return MessageView{
		Role:       m.Role,
		Content:    m.Content,
		ToolCallID: m.ToolCallID,
	}
}
