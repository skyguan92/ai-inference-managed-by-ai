package agent

import (
	"context"
	"fmt"

	coreagent "github.com/jguan/ai-inference-managed-by-ai/pkg/agent"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

// StatusQuery implements agent.status.
// Output: {enabled, provider, model, active_conversations}
type StatusQuery struct {
	agent *coreagent.Agent
}

func NewStatusQuery(agent *coreagent.Agent) *StatusQuery {
	return &StatusQuery{agent: agent}
}

func (q *StatusQuery) Name() string        { return "agent.status" }
func (q *StatusQuery) Domain() string      { return "agent" }
func (q *StatusQuery) Description() string { return "Get the current status of the AI agent operator" }

func (q *StatusQuery) InputSchema() unit.Schema {
	return unit.Schema{Type: "object", Properties: map[string]unit.Field{}}
}

func (q *StatusQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"enabled":              {Name: "enabled", Schema: unit.Schema{Type: "boolean"}},
			"provider":             {Name: "provider", Schema: unit.Schema{Type: "string"}},
			"model":                {Name: "model", Schema: unit.Schema{Type: "string"}},
			"active_conversations": {Name: "active_conversations", Schema: unit.Schema{Type: "integer"}},
		},
	}
}

func (q *StatusQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Description: "Get agent status",
			Input:       map[string]any{},
			Output: map[string]any{
				"enabled":              true,
				"provider":             "anthropic",
				"model":                "claude-haiku-4-5-20251001",
				"active_conversations": 2,
			},
		},
	}
}

func (q *StatusQuery) Execute(_ context.Context, _ any) (any, error) {
	if q.agent == nil {
		return AgentStatus{
			Enabled:             false,
			Provider:            "",
			Model:               "",
			ActiveConversations: 0,
		}, nil
	}

	return AgentStatus{
		Enabled:             true,
		Provider:            q.agent.LLMName(),
		Model:               q.agent.LLMModelName(),
		ActiveConversations: q.agent.ActiveConversationCount(),
	}, nil
}

// HistoryQuery implements agent.history.
// Input:  {conversation_id: string, limit?: int}
// Output: {conversation_id: string, messages: []MessageView}
type HistoryQuery struct {
	agent *coreagent.Agent
}

func NewHistoryQuery(agent *coreagent.Agent) *HistoryQuery {
	return &HistoryQuery{agent: agent}
}

func (q *HistoryQuery) Name() string        { return "agent.history" }
func (q *HistoryQuery) Domain() string      { return "agent" }
func (q *HistoryQuery) Description() string { return "Retrieve conversation message history" }

func (q *HistoryQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"conversation_id": {
				Name: "conversation_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "ID of the conversation",
				},
			},
			"limit": {
				Name: "limit",
				Schema: unit.Schema{
					Type:        "integer",
					Description: "Maximum number of messages to return (newest first); 0 means all",
					Default:     0,
				},
			},
		},
		Required: []string{"conversation_id"},
	}
}

func (q *HistoryQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"conversation_id": {Name: "conversation_id", Schema: unit.Schema{Type: "string"}},
			"messages":        {Name: "messages", Schema: unit.Schema{Type: "array"}},
		},
	}
}

func (q *HistoryQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Description: "Get conversation history",
			Input:       map[string]any{"conversation_id": "conv-abc123"},
			Output: map[string]any{
				"conversation_id": "conv-abc123",
				"messages":        []any{},
			},
		},
	}
}

func (q *HistoryQuery) Execute(_ context.Context, input any) (any, error) {
	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, &unit.UnitError{
			Code:    unit.ErrCodeInvalidInput,
			Message: fmt.Sprintf("invalid input type: expected map[string]any, got %T", input),
		}
	}

	convID, ok := inputMap["conversation_id"].(string)
	if !ok || convID == "" {
		return nil, &unit.UnitError{
			Code:    unit.ErrCodeInvalidInput,
			Message: "conversation_id is required",
		}
	}

	if q.agent == nil {
		return nil, errAgentNotEnabled()
	}

	conv := q.agent.GetConversation(convID)
	if conv == nil {
		return nil, errConversationNotFound(convID)
	}

	limit := 0
	if v, ok := inputMap["limit"]; ok {
		switch n := v.(type) {
		case int:
			limit = n
		case float64:
			limit = int(n)
		}
	}

	msgs := conv.Messages
	if limit > 0 && limit < len(msgs) {
		msgs = msgs[len(msgs)-limit:]
	}

	views := make([]MessageView, 0, len(msgs))
	for _, m := range msgs {
		views = append(views, messageToView(m))
	}

	return map[string]any{
		"conversation_id": convID,
		"messages":        views,
	}, nil
}
