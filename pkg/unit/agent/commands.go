package agent

import (
	"context"
	"fmt"

	coreagent "github.com/jguan/ai-inference-managed-by-ai/pkg/agent"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

// ChatCommand implements agent.chat.
// Input:  {message: string, conversation_id?: string}
// Output: {response: string, conversation_id: string, tool_calls_count: int}
type ChatCommand struct {
	agent  *coreagent.Agent
	events unit.EventPublisher
}

func NewChatCommand(agent *coreagent.Agent) *ChatCommand {
	return &ChatCommand{agent: agent}
}

func NewChatCommandWithEvents(agent *coreagent.Agent, events unit.EventPublisher) *ChatCommand {
	return &ChatCommand{agent: agent, events: events}
}

func (c *ChatCommand) Name() string        { return "agent.chat" }
func (c *ChatCommand) Domain() string      { return "agent" }
func (c *ChatCommand) Description() string { return "Send a message to the AI agent and get a response" }

func (c *ChatCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"message": {
				Name: "message",
				Schema: unit.Schema{
					Type:        "string",
					Description: "User message to send to the agent",
				},
			},
			"conversation_id": {
				Name: "conversation_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Conversation ID to continue; omit to start a new conversation",
				},
			},
		},
		Required: []string{"message"},
	}
}

func (c *ChatCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"response":         {Name: "response", Schema: unit.Schema{Type: "string"}},
			"conversation_id":  {Name: "conversation_id", Schema: unit.Schema{Type: "string"}},
			"tool_calls_count": {Name: "tool_calls_count", Schema: unit.Schema{Type: "integer"}},
		},
	}
}

func (c *ChatCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Description: "Start a new conversation",
			Input:       map[string]any{"message": "List all deployed models"},
			Output:      map[string]any{"response": "...", "conversation_id": "conv-abc123", "tool_calls_count": 1},
		},
	}
}

func (c *ChatCommand) Execute(ctx context.Context, input any) (any, error) {
	if c.agent == nil {
		return nil, errAgentNotEnabled()
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, &unit.UnitError{
			Code:    unit.ErrCodeInvalidInput,
			Message: fmt.Sprintf("invalid input type: expected map[string]any, got %T", input),
		}
	}

	message, ok := inputMap["message"].(string)
	if !ok || message == "" {
		return nil, &unit.UnitError{
			Code:    unit.ErrCodeInvalidInput,
			Message: "message is required",
		}
	}

	conversationID, _ := inputMap["conversation_id"].(string)

	// Detect new conversation creation (no existing ID provided).
	isNewConversation := conversationID == ""

	reply, convID, err := c.agent.Chat(ctx, conversationID, message)
	if err != nil {
		return nil, errAgentLLMError(err)
	}

	// Count tool calls in history to report back.
	toolCallsCount := 0
	conv := c.agent.GetConversation(convID)
	if conv != nil {
		for _, m := range conv.Messages {
			toolCallsCount += len(m.ToolCalls)
		}
	}

	result := map[string]any{
		"response":         reply,
		"conversation_id":  convID,
		"tool_calls_count": toolCallsCount,
	}

	if c.events != nil {
		if isNewConversation {
			_ = c.events.Publish(NewConversationCreatedEvent(convID))
		}
		// Emit tool_called events for tool calls in this conversation.
		// We scan conversation messages for tool call records.
		if conv != nil {
			for _, m := range conv.Messages {
				for _, tc := range m.ToolCalls {
					_ = c.events.Publish(NewToolCalledEvent(convID, tc.Name, true))
				}
			}
		}
		_ = c.events.Publish(NewMessageSentEvent(convID, message, reply))
	}

	return result, nil
}

// ResetCommand implements agent.reset.
// Input:  {conversation_id: string}
// Output: {success: bool, conversation_id: string}
type ResetCommand struct {
	agent  *coreagent.Agent
	events unit.EventPublisher
}

func NewResetCommand(agent *coreagent.Agent) *ResetCommand {
	return &ResetCommand{agent: agent}
}

func NewResetCommandWithEvents(agent *coreagent.Agent, events unit.EventPublisher) *ResetCommand {
	return &ResetCommand{agent: agent, events: events}
}

func (c *ResetCommand) Name() string        { return "agent.reset" }
func (c *ResetCommand) Domain() string      { return "agent" }
func (c *ResetCommand) Description() string { return "Reset (clear) a conversation" }

func (c *ResetCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"conversation_id": {
				Name: "conversation_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "ID of the conversation to reset",
				},
			},
		},
		Required: []string{"conversation_id"},
	}
}

func (c *ResetCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"success":         {Name: "success", Schema: unit.Schema{Type: "boolean"}},
			"conversation_id": {Name: "conversation_id", Schema: unit.Schema{Type: "string"}},
		},
	}
}

func (c *ResetCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Description: "Reset a conversation",
			Input:       map[string]any{"conversation_id": "conv-abc123"},
			Output:      map[string]any{"success": true, "conversation_id": "conv-abc123"},
		},
	}
}

func (c *ResetCommand) Execute(ctx context.Context, input any) (any, error) {
	if c.agent == nil {
		return nil, errAgentNotEnabled()
	}

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

	deleted := c.agent.ResetConversation(convID)
	if !deleted {
		return nil, errConversationNotFound(convID)
	}

	if c.events != nil {
		_ = c.events.Publish(NewConversationResetEvent(convID))
	}

	return map[string]any{
		"success":         true,
		"conversation_id": convID,
	}, nil
}
