package agent

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	coreagent "github.com/jguan/ai-inference-managed-by-ai/pkg/agent"
	agentllm "github.com/jguan/ai-inference-managed-by-ai/pkg/agent/llm"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

// mockLLM satisfies agentllm.LLMClient for testing.
type mockLLM struct {
	name      string
	model     string
	responses []*agentllm.ChatResponse
	errors    []error
	callCount int
}

func (m *mockLLM) Name() string      { return m.name }
func (m *mockLLM) ModelName() string { return m.model }

func (m *mockLLM) Chat(_ context.Context, _ []agentllm.Message, _ []agentllm.ToolDef, _ agentllm.ChatOptions) (*agentllm.ChatResponse, error) {
	i := m.callCount
	m.callCount++
	if i < len(m.errors) && m.errors[i] != nil {
		return nil, m.errors[i]
	}
	if i < len(m.responses) {
		return m.responses[i], nil
	}
	return &agentllm.ChatResponse{
		Message: agentllm.Message{Role: "assistant", Content: "ok"},
	}, nil
}

func newAgent(llm agentllm.LLMClient) *coreagent.Agent {
	return coreagent.NewAgent(llm, nil, nil, coreagent.AgentOptions{})
}

func TestChatCommand_Execute_Success(t *testing.T) {
	llm := &mockLLM{
		name:  "mock",
		model: "mock-model",
		responses: []*agentllm.ChatResponse{
			{Message: agentllm.Message{Role: "assistant", Content: "Hello!"}},
		},
	}
	cmd := NewChatCommand(newAgent(llm))

	result, err := cmd.Execute(context.Background(), map[string]any{"message": "hi"})
	require.NoError(t, err)

	m, ok := result.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "Hello!", m["response"])
	assert.NotEmpty(t, m["conversation_id"])
	assert.Equal(t, 0, m["tool_calls_count"])
}

func TestChatCommand_Execute_ContinuesConversation(t *testing.T) {
	llm := &mockLLM{
		name:  "mock",
		model: "mock-model",
		responses: []*agentllm.ChatResponse{
			{Message: agentllm.Message{Role: "assistant", Content: "First reply"}},
			{Message: agentllm.Message{Role: "assistant", Content: "Second reply"}},
		},
	}
	cmd := NewChatCommand(newAgent(llm))

	r1, err := cmd.Execute(context.Background(), map[string]any{
		"message":         "first",
		"conversation_id": "test-conv",
	})
	require.NoError(t, err)
	assert.Equal(t, "test-conv", r1.(map[string]any)["conversation_id"])

	r2, err := cmd.Execute(context.Background(), map[string]any{
		"message":         "second",
		"conversation_id": "test-conv",
	})
	require.NoError(t, err)
	assert.Equal(t, "test-conv", r2.(map[string]any)["conversation_id"])
	assert.Equal(t, "Second reply", r2.(map[string]any)["response"])
}

func TestChatCommand_Execute_NilAgent(t *testing.T) {
	cmd := NewChatCommand(nil)
	_, err := cmd.Execute(context.Background(), map[string]any{"message": "hi"})
	assert.Error(t, err)
	ue, ok := unit.AsUnitError(err)
	require.True(t, ok)
	assert.Equal(t, unit.ErrCodeAgentNotEnabled, ue.Code)
}

func TestChatCommand_Execute_EmptyMessage(t *testing.T) {
	llm := &mockLLM{name: "mock", model: "mock-model"}
	cmd := NewChatCommand(newAgent(llm))
	_, err := cmd.Execute(context.Background(), map[string]any{"message": ""})
	assert.Error(t, err)
	ue, ok := unit.AsUnitError(err)
	require.True(t, ok)
	assert.Equal(t, unit.ErrCodeInvalidInput, ue.Code)
}

func TestChatCommand_Execute_LLMError(t *testing.T) {
	llm := &mockLLM{
		name:   "mock",
		model:  "mock-model",
		errors: []error{fmt.Errorf("LLM offline")},
	}
	cmd := NewChatCommand(newAgent(llm))
	_, err := cmd.Execute(context.Background(), map[string]any{"message": "hi"})
	assert.Error(t, err)
	ue, ok := unit.AsUnitError(err)
	require.True(t, ok)
	assert.Equal(t, unit.ErrCodeAgentLLMError, ue.Code)
}

func TestChatCommand_Metadata(t *testing.T) {
	cmd := NewChatCommand(nil)
	assert.Equal(t, "agent.chat", cmd.Name())
	assert.Equal(t, "agent", cmd.Domain())
	assert.NotEmpty(t, cmd.Description())
	assert.NotEmpty(t, cmd.InputSchema().Properties)
	assert.NotEmpty(t, cmd.Examples())
}

func TestResetCommand_Execute_Success(t *testing.T) {
	llm := &mockLLM{
		name:  "mock",
		model: "mock-model",
		responses: []*agentllm.ChatResponse{
			{Message: agentllm.Message{Role: "assistant", Content: "ok"}},
		},
	}
	a := newAgent(llm)
	chatCmd := NewChatCommand(a)
	_, err := chatCmd.Execute(context.Background(), map[string]any{
		"message":         "hello",
		"conversation_id": "reset-test",
	})
	require.NoError(t, err)

	resetCmd := NewResetCommand(a)
	result, err := resetCmd.Execute(context.Background(), map[string]any{"conversation_id": "reset-test"})
	require.NoError(t, err)
	m := result.(map[string]any)
	assert.True(t, m["success"].(bool))
	assert.Equal(t, "reset-test", m["conversation_id"])
}

func TestResetCommand_Execute_NotFound(t *testing.T) {
	a := newAgent(&mockLLM{name: "mock", model: "m"})
	cmd := NewResetCommand(a)
	_, err := cmd.Execute(context.Background(), map[string]any{"conversation_id": "nonexistent"})
	assert.Error(t, err)
	ue, ok := unit.AsUnitError(err)
	require.True(t, ok)
	assert.Equal(t, unit.ErrCodeConversationNotFound, ue.Code)
}

func TestResetCommand_Execute_NilAgent(t *testing.T) {
	cmd := NewResetCommand(nil)
	_, err := cmd.Execute(context.Background(), map[string]any{"conversation_id": "x"})
	assert.Error(t, err)
}

func TestResetCommand_Execute_MissingID(t *testing.T) {
	a := newAgent(&mockLLM{name: "mock", model: "m"})
	cmd := NewResetCommand(a)
	_, err := cmd.Execute(context.Background(), map[string]any{})
	assert.Error(t, err)
	ue, ok := unit.AsUnitError(err)
	require.True(t, ok)
	assert.Equal(t, unit.ErrCodeInvalidInput, ue.Code)
}

func TestResetCommand_Metadata(t *testing.T) {
	cmd := NewResetCommand(nil)
	assert.Equal(t, "agent.reset", cmd.Name())
	assert.Equal(t, "agent", cmd.Domain())
}
