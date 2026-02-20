package agent

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	agentllm "github.com/jguan/ai-inference-managed-by-ai/pkg/agent/llm"
)

// mockLLM is a controllable LLMClient for tests.
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
	// Default: echo a text response
	return &agentllm.ChatResponse{
		Message: agentllm.Message{Role: "assistant", Content: "default response"},
	}, nil
}

func newTextResponse(content string) *agentllm.ChatResponse {
	return &agentllm.ChatResponse{
		Message: agentllm.Message{Role: "assistant", Content: content},
	}
}

func newToolCallResponse(content string, calls ...agentllm.ToolCall) *agentllm.ChatResponse {
	return &agentllm.ChatResponse{
		Message:   agentllm.Message{Role: "assistant", Content: content, ToolCalls: calls},
		ToolCalls: calls,
	}
}

// --- Agent tests ---

func TestAgent_Chat_SimpleText(t *testing.T) {
	llm := &mockLLM{
		name:  "mock",
		model: "mock-model",
		responses: []*agentllm.ChatResponse{
			newTextResponse("Hello from AIMA!"),
		},
	}

	agent := NewAgent(llm, nil, nil, AgentOptions{})
	reply, convID, err := agent.Chat(context.Background(), "", "hello")
	require.NoError(t, err)
	assert.Equal(t, "Hello from AIMA!", reply)
	assert.NotEmpty(t, convID)
}

func TestAgent_Chat_PreservesConversationID(t *testing.T) {
	llm := &mockLLM{
		name:  "mock",
		model: "mock-model",
		responses: []*agentllm.ChatResponse{
			newTextResponse("First reply"),
			newTextResponse("Second reply"),
		},
	}

	agent := NewAgent(llm, nil, nil, AgentOptions{})

	_, convID, err := agent.Chat(context.Background(), "conv-fixed", "first")
	require.NoError(t, err)
	assert.Equal(t, "conv-fixed", convID)

	_, convID2, err := agent.Chat(context.Background(), "conv-fixed", "second")
	require.NoError(t, err)
	assert.Equal(t, "conv-fixed", convID2)

	// Conversation should have 4 messages: 2 user + 2 assistant.
	conv := agent.GetConversation("conv-fixed")
	require.NotNil(t, conv)
	assert.Len(t, conv.Messages, 4)
}

func TestAgent_Chat_ToolCallThenText(t *testing.T) {
	toolCall := agentllm.ToolCall{
		ID:        "tc_01",
		Name:      "model_list",
		Arguments: map[string]any{"limit": 10},
	}

	llm := &mockLLM{
		name:  "mock",
		model: "mock-model",
		responses: []*agentllm.ChatResponse{
			// First call: request a tool
			newToolCallResponse("Let me check.", toolCall),
			// Second call: final text after seeing tool result
			newTextResponse("There are 3 models installed."),
		},
	}

	agent := NewAgent(llm, nil, nil, AgentOptions{})
	reply, _, err := agent.Chat(context.Background(), "", "how many models?")
	require.NoError(t, err)
	assert.Equal(t, "There are 3 models installed.", reply)
	assert.Equal(t, 2, llm.callCount)
}

func TestAgent_Chat_EmptyMessage(t *testing.T) {
	llm := &mockLLM{name: "mock", model: "mock-model"}
	agent := NewAgent(llm, nil, nil, AgentOptions{})
	_, _, err := agent.Chat(context.Background(), "", "")
	assert.Error(t, err)
}

func TestAgent_Chat_NilLLM(t *testing.T) {
	agent := NewAgent(nil, nil, nil, AgentOptions{})
	_, _, err := agent.Chat(context.Background(), "", "hello")
	assert.Error(t, err)
}

func TestAgent_Chat_LLMError(t *testing.T) {
	llm := &mockLLM{
		name:  "mock",
		model: "mock-model",
		errors: []error{fmt.Errorf("LLM unavailable")},
	}
	agent := NewAgent(llm, nil, nil, AgentOptions{})
	_, _, err := agent.Chat(context.Background(), "", "hello")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LLM unavailable")
}

func TestAgent_Chat_MaxToolCallRounds(t *testing.T) {
	// LLM keeps requesting tool calls forever.
	toolCall := agentllm.ToolCall{ID: "tc", Name: "infinite_tool", Arguments: map[string]any{}}
	responses := make([]*agentllm.ChatResponse, maxToolCallRounds+2)
	for i := range responses {
		responses[i] = newToolCallResponse("still thinking...", toolCall)
	}

	llm := &mockLLM{name: "mock", model: "mock-model", responses: responses}
	agent := NewAgent(llm, nil, nil, AgentOptions{})
	_, _, err := agent.Chat(context.Background(), "", "loop me")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeded maximum tool call rounds")
}

func TestAgent_ResetConversation(t *testing.T) {
	llm := &mockLLM{
		name:  "mock",
		model: "mock-model",
		responses: []*agentllm.ChatResponse{
			newTextResponse("hi"),
		},
	}
	agent := NewAgent(llm, nil, nil, AgentOptions{})

	_, _, err := agent.Chat(context.Background(), "conv-del", "hello")
	require.NoError(t, err)

	ok := agent.ResetConversation("conv-del")
	assert.True(t, ok)

	conv := agent.GetConversation("conv-del")
	assert.Nil(t, conv)
}

func TestAgent_Metadata(t *testing.T) {
	llm := &mockLLM{name: "anthropic", model: "claude-haiku-4-5-20251001"}
	agent := NewAgent(llm, nil, nil, AgentOptions{})
	assert.Equal(t, "anthropic", agent.LLMName())
	assert.Equal(t, "claude-haiku-4-5-20251001", agent.LLMModelName())
}

func TestAgent_NilLLMMetadata(t *testing.T) {
	agent := NewAgent(nil, nil, nil, AgentOptions{})
	assert.Equal(t, "", agent.LLMName())
	assert.Equal(t, "", agent.LLMModelName())
}
