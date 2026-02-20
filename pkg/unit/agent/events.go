package agent

import (
	"time"

	"github.com/google/uuid"
)

const (
	EventTypeMessageSent         = "agent.message_sent"
	EventTypeToolCalled          = "agent.tool_called"
	EventTypeConversationCreated = "agent.conversation_created"
	EventTypeConversationReset   = "agent.conversation_reset"
)

// MessageSentEvent is published after the agent successfully replies.
type MessageSentEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewMessageSentEvent(conversationID, userMessage, agentReply string) *MessageSentEvent {
	return &MessageSentEvent{
		eventType: EventTypeMessageSent,
		domain:    "agent",
		payload: map[string]any{
			"conversation_id": conversationID,
			"user_message":    userMessage,
			"agent_reply":     agentReply,
		},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *MessageSentEvent) Type() string          { return e.eventType }
func (e *MessageSentEvent) Domain() string        { return e.domain }
func (e *MessageSentEvent) Payload() any          { return e.payload }
func (e *MessageSentEvent) Timestamp() time.Time  { return e.timestamp }
func (e *MessageSentEvent) CorrelationID() string { return e.correlationID }

// ConversationResetEvent is published when a conversation is cleared.
type ConversationResetEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewConversationResetEvent(conversationID string) *ConversationResetEvent {
	return &ConversationResetEvent{
		eventType:     EventTypeConversationReset,
		domain:        "agent",
		payload:       map[string]any{"conversation_id": conversationID},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *ConversationResetEvent) Type() string          { return e.eventType }
func (e *ConversationResetEvent) Domain() string        { return e.domain }
func (e *ConversationResetEvent) Payload() any          { return e.payload }
func (e *ConversationResetEvent) Timestamp() time.Time  { return e.timestamp }
func (e *ConversationResetEvent) CorrelationID() string { return e.correlationID }

// ToolCalledEvent is published when the agent invokes a tool via MCP.
type ToolCalledEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewToolCalledEvent(conversationID, toolName string, success bool) *ToolCalledEvent {
	return &ToolCalledEvent{
		eventType: EventTypeToolCalled,
		domain:    "agent",
		payload: map[string]any{
			"conversation_id": conversationID,
			"tool_name":       toolName,
			"success":         success,
		},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *ToolCalledEvent) Type() string          { return e.eventType }
func (e *ToolCalledEvent) Domain() string        { return e.domain }
func (e *ToolCalledEvent) Payload() any          { return e.payload }
func (e *ToolCalledEvent) Timestamp() time.Time  { return e.timestamp }
func (e *ToolCalledEvent) CorrelationID() string { return e.correlationID }

// ConversationCreatedEvent is published when a new conversation is started.
type ConversationCreatedEvent struct {
	eventType     string
	domain        string
	payload       any
	timestamp     time.Time
	correlationID string
}

func NewConversationCreatedEvent(conversationID string) *ConversationCreatedEvent {
	return &ConversationCreatedEvent{
		eventType:     EventTypeConversationCreated,
		domain:        "agent",
		payload:       map[string]any{"conversation_id": conversationID},
		timestamp:     time.Now(),
		correlationID: uuid.New().String(),
	}
}

func (e *ConversationCreatedEvent) Type() string          { return e.eventType }
func (e *ConversationCreatedEvent) Domain() string        { return e.domain }
func (e *ConversationCreatedEvent) Payload() any          { return e.payload }
func (e *ConversationCreatedEvent) Timestamp() time.Time  { return e.timestamp }
func (e *ConversationCreatedEvent) CorrelationID() string { return e.correlationID }
