package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	coreagent "github.com/jguan/ai-inference-managed-by-ai/pkg/agent"
	agentllm "github.com/jguan/ai-inference-managed-by-ai/pkg/agent/llm"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

// newAgentTestRoot builds a minimal RootCommand with an empty registry suitable
// for CLI structure and execution tests (all gateway calls will return "not found").
func newAgentTestRoot(t *testing.T) *RootCommand {
	t.Helper()
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)
	return &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     NewOutputOptions(),
	}
}

func newAgentTestRootWithBuf(t *testing.T) (*RootCommand, *bytes.Buffer) {
	t.Helper()
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)
	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputJSON, Writer: buf},
	}
	return root, buf
}

// ---------------------------------------------------------------------------
// NewAgentCommand — structure
// ---------------------------------------------------------------------------

func TestNewAgentCommand(t *testing.T) {
	root := newAgentTestRoot(t)
	cmd := NewAgentCommand(root)

	assert.NotNil(t, cmd)
	assert.Equal(t, "agent", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
}

func TestNewAgentCommand_Subcommands(t *testing.T) {
	root := newAgentTestRoot(t)
	cmd := NewAgentCommand(root)

	subCmds := cmd.Commands()
	assert.Len(t, subCmds, 5)

	names := make([]string, len(subCmds))
	for i, c := range subCmds {
		names[i] = c.Name()
	}
	assert.Contains(t, names, "chat")
	assert.Contains(t, names, "ask")
	assert.Contains(t, names, "status")
	assert.Contains(t, names, "history")
	assert.Contains(t, names, "reset")
}

// ---------------------------------------------------------------------------
// NewAgentChatCommand
// ---------------------------------------------------------------------------

func TestNewAgentChatCommand_Structure(t *testing.T) {
	root := newAgentTestRoot(t)
	cmd := NewAgentChatCommand(root)

	assert.NotNil(t, cmd)
	assert.Equal(t, "chat <message>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotEmpty(t, cmd.Example)
	assert.NotNil(t, cmd.Args) // cobra.ExactArgs(1)
}

func TestNewAgentChatCommand_ConversationFlag(t *testing.T) {
	root := newAgentTestRoot(t)
	cmd := NewAgentChatCommand(root)

	flag := cmd.Flags().Lookup("conversation")
	require.NotNil(t, flag)
	assert.Equal(t, "c", flag.Shorthand)
	assert.Equal(t, "", flag.DefValue)
}

func TestRunAgentChat_GatewayError(t *testing.T) {
	root, _ := newAgentTestRootWithBuf(t)

	err := runAgentChat(context.Background(), root, "hello", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent chat failed")
}

func TestRunAgentChat_WithConversationID(t *testing.T) {
	root, _ := newAgentTestRootWithBuf(t)

	err := runAgentChat(context.Background(), root, "how many models?", "conv-abc123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent chat failed")
}

func TestRunAgentChat_EmptyConversationID(t *testing.T) {
	root, _ := newAgentTestRootWithBuf(t)

	// Empty conversation ID means start new — still fails at gateway level.
	err := runAgentChat(context.Background(), root, "hello", "")
	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// NewAgentAskCommand
// ---------------------------------------------------------------------------

func TestNewAgentAskCommand_Structure(t *testing.T) {
	root := newAgentTestRoot(t)
	cmd := NewAgentAskCommand(root)

	assert.NotNil(t, cmd)
	assert.Equal(t, "ask <message>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotEmpty(t, cmd.Example)
	assert.NotNil(t, cmd.Args) // cobra.ExactArgs(1)
}

func TestNewAgentAskCommand_NoFlags(t *testing.T) {
	root := newAgentTestRoot(t)
	cmd := NewAgentAskCommand(root)

	// ask is a one-shot command with no extra flags.
	assert.Empty(t, cmd.Flags().FlagUsages())
}

func TestRunAgentAsk_GatewayError(t *testing.T) {
	root, _ := newAgentTestRootWithBuf(t)

	// ask delegates to runAgentChat with empty conversationID.
	err := runAgentChat(context.Background(), root, "what is GPU utilisation?", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent chat failed")
}

// ---------------------------------------------------------------------------
// NewAgentStatusCommand
// ---------------------------------------------------------------------------

func TestNewAgentStatusCommand_Structure(t *testing.T) {
	root := newAgentTestRoot(t)
	cmd := NewAgentStatusCommand(root)

	assert.NotNil(t, cmd)
	assert.Equal(t, "status", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
}

func TestRunAgentStatus_GatewayError(t *testing.T) {
	root, _ := newAgentTestRootWithBuf(t)

	err := runAgentStatus(context.Background(), root)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent status failed")
}

// ---------------------------------------------------------------------------
// NewAgentHistoryCommand
// ---------------------------------------------------------------------------

func TestNewAgentHistoryCommand_Structure(t *testing.T) {
	root := newAgentTestRoot(t)
	cmd := NewAgentHistoryCommand(root)

	assert.NotNil(t, cmd)
	assert.Equal(t, "history <conversation-id>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Example)
	assert.NotNil(t, cmd.Args) // cobra.ExactArgs(1)
}

func TestNewAgentHistoryCommand_LimitFlag(t *testing.T) {
	root := newAgentTestRoot(t)
	cmd := NewAgentHistoryCommand(root)

	flag := cmd.Flags().Lookup("limit")
	require.NotNil(t, flag)
	assert.Equal(t, "0", flag.DefValue)
}

func TestRunAgentHistory_GatewayError(t *testing.T) {
	root, _ := newAgentTestRootWithBuf(t)

	err := runAgentHistory(context.Background(), root, "conv-abc123", 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent history failed")
}

func TestRunAgentHistory_WithLimit(t *testing.T) {
	root, _ := newAgentTestRootWithBuf(t)

	err := runAgentHistory(context.Background(), root, "conv-abc123", 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent history failed")
}

// ---------------------------------------------------------------------------
// NewAgentResetCommand
// ---------------------------------------------------------------------------

func TestNewAgentResetCommand_Structure(t *testing.T) {
	root := newAgentTestRoot(t)
	cmd := NewAgentResetCommand(root)

	assert.NotNil(t, cmd)
	assert.Equal(t, "reset <conversation-id>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Example)
	assert.NotNil(t, cmd.Args) // cobra.ExactArgs(1)
}

func TestRunAgentReset_GatewayError(t *testing.T) {
	root, _ := newAgentTestRootWithBuf(t)

	err := runAgentReset(context.Background(), root, "conv-abc123")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent reset failed")
}

func TestRunAgentReset_DifferentConversationIDs(t *testing.T) {
	tests := []struct {
		name           string
		conversationID string
	}{
		{"short id", "conv-1"},
		{"long uuid", "conv-abc123def456"},
		{"empty-like id", "conv-"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root, _ := newAgentTestRootWithBuf(t)
			err := runAgentReset(context.Background(), root, tc.conversationID)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "agent reset failed")
		})
	}
}

// ---------------------------------------------------------------------------
// Conversation persistence helpers
// ---------------------------------------------------------------------------

func makeTestConversation(id string) *coreagent.Conversation {
	return &coreagent.Conversation{
		ID: id,
		Messages: []agentllm.Message{
			{Role: "user", Content: "hello"},
			{Role: "assistant", Content: "hi there"},
		},
		CreatedAt: time.Now().Truncate(time.Second),
		UpdatedAt: time.Now().Truncate(time.Second),
	}
}

func TestConversationPersistence_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	convID := "conv-test123"

	orig := makeTestConversation(convID)
	err := saveConversationToFile(dir, convID, orig)
	require.NoError(t, err)

	loaded, err := loadConversationFromFile(dir, convID)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, orig.ID, loaded.ID)
	assert.Len(t, loaded.Messages, 2)
	assert.Equal(t, orig.Messages[0].Role, loaded.Messages[0].Role)
	assert.Equal(t, orig.Messages[0].Content, loaded.Messages[0].Content)
	assert.Equal(t, orig.Messages[1].Role, loaded.Messages[1].Role)
	assert.Equal(t, orig.Messages[1].Content, loaded.Messages[1].Content)
}

func TestConversationPersistence_FileNotFound(t *testing.T) {
	dir := t.TempDir()

	conv, err := loadConversationFromFile(dir, "nonexistent-conv")
	require.NoError(t, err)
	assert.Nil(t, conv)
}

func TestConversationPersistence_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	convID := "conv-bad"

	err := saveConversationToFile(dir, convID, makeTestConversation(convID))
	require.NoError(t, err)

	// Overwrite with invalid JSON.
	convPath := filepath.Join(dir, "conversations", convID+".json")
	require.NoError(t, os.WriteFile(convPath, []byte("not json {"), 0644))

	_, err = loadConversationFromFile(dir, convID)
	require.Error(t, err)
}

func TestConversationPersistence_MultipleConversations(t *testing.T) {
	dir := t.TempDir()

	for _, id := range []string{"conv-a", "conv-b", "conv-c"} {
		require.NoError(t, saveConversationToFile(dir, id, makeTestConversation(id)))
	}

	for _, id := range []string{"conv-a", "conv-b", "conv-c"} {
		conv, err := loadConversationFromFile(dir, id)
		require.NoError(t, err)
		require.NotNil(t, conv)
		assert.Equal(t, id, conv.ID)
	}
}

