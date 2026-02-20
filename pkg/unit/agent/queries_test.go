package agent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	coreagent "github.com/jguan/ai-inference-managed-by-ai/pkg/agent"
	agentllm "github.com/jguan/ai-inference-managed-by-ai/pkg/agent/llm"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestStatusQuery_NilAgent(t *testing.T) {
	q := NewStatusQuery(nil)
	result, err := q.Execute(context.Background(), nil)
	require.NoError(t, err)
	s, ok := result.(AgentStatus)
	require.True(t, ok)
	assert.False(t, s.Enabled)
}

func TestStatusQuery_WithAgent(t *testing.T) {
	llm := &mockLLM{name: "anthropic", model: "claude-haiku-4-5-20251001"}
	a := coreagent.NewAgent(llm, nil, nil, coreagent.AgentOptions{})
	q := NewStatusQuery(a)

	result, err := q.Execute(context.Background(), nil)
	require.NoError(t, err)
	s, ok := result.(AgentStatus)
	require.True(t, ok)
	assert.True(t, s.Enabled)
	assert.Equal(t, "anthropic", s.Provider)
	assert.Equal(t, "claude-haiku-4-5-20251001", s.Model)
	assert.Equal(t, 0, s.ActiveConversations)
}

func TestStatusQuery_Metadata(t *testing.T) {
	q := NewStatusQuery(nil)
	assert.Equal(t, "agent.status", q.Name())
	assert.Equal(t, "agent", q.Domain())
	assert.NotEmpty(t, q.Description())
}

func TestHistoryQuery_MissingConversationID(t *testing.T) {
	a := coreagent.NewAgent(&mockLLM{name: "m", model: "m"}, nil, nil, coreagent.AgentOptions{})
	q := NewHistoryQuery(a)
	_, err := q.Execute(context.Background(), map[string]any{})
	require.Error(t, err)
	ue, ok := unit.AsUnitError(err)
	require.True(t, ok)
	assert.Equal(t, unit.ErrCodeInvalidInput, ue.Code)
}

func TestHistoryQuery_ConversationNotFound(t *testing.T) {
	a := coreagent.NewAgent(&mockLLM{name: "m", model: "m"}, nil, nil, coreagent.AgentOptions{})
	q := NewHistoryQuery(a)
	_, err := q.Execute(context.Background(), map[string]any{"conversation_id": "missing"})
	require.Error(t, err)
	ue, ok := unit.AsUnitError(err)
	require.True(t, ok)
	assert.Equal(t, unit.ErrCodeConversationNotFound, ue.Code)
}

func TestHistoryQuery_Success(t *testing.T) {
	llm := &mockLLM{
		name:  "mock",
		model: "mock-model",
		responses: []*agentllm.ChatResponse{
			{Message: agentllm.Message{Role: "assistant", Content: "hi"}},
		},
	}
	a := coreagent.NewAgent(llm, nil, nil, coreagent.AgentOptions{})

	// Seed a conversation via Chat.
	_, convID, err := a.Chat(context.Background(), "hist-test", "hello")
	require.NoError(t, err)

	q := NewHistoryQuery(a)
	result, err := q.Execute(context.Background(), map[string]any{"conversation_id": convID})
	require.NoError(t, err)

	m := result.(map[string]any)
	assert.Equal(t, convID, m["conversation_id"])

	views, ok := m["messages"].([]MessageView)
	require.True(t, ok)
	assert.Len(t, views, 2) // 1 user + 1 assistant
}

func TestHistoryQuery_WithLimit(t *testing.T) {
	llm := &mockLLM{
		name:  "mock",
		model: "mock-model",
		responses: []*agentllm.ChatResponse{
			{Message: agentllm.Message{Role: "assistant", Content: "r1"}},
			{Message: agentllm.Message{Role: "assistant", Content: "r2"}},
			{Message: agentllm.Message{Role: "assistant", Content: "r3"}},
		},
	}
	a := coreagent.NewAgent(llm, nil, nil, coreagent.AgentOptions{})

	ctx := context.Background()
	for _, msg := range []string{"m1", "m2", "m3"} {
		_, _, err := a.Chat(ctx, "limit-test", msg)
		require.NoError(t, err)
	}

	q := NewHistoryQuery(a)
	result, err := q.Execute(ctx, map[string]any{
		"conversation_id": "limit-test",
		"limit":           float64(2),
	})
	require.NoError(t, err)

	m := result.(map[string]any)
	views := m["messages"].([]MessageView)
	assert.Len(t, views, 2)
}

func TestHistoryQuery_NilAgent(t *testing.T) {
	q := NewHistoryQuery(nil)
	_, err := q.Execute(context.Background(), map[string]any{"conversation_id": "x"})
	assert.Error(t, err)
}

func TestHistoryQuery_Metadata(t *testing.T) {
	q := NewHistoryQuery(nil)
	assert.Equal(t, "agent.history", q.Name())
	assert.Equal(t, "agent", q.Domain())
}
