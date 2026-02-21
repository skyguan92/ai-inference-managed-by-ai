package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"time"

	coreagent "github.com/jguan/ai-inference-managed-by-ai/pkg/agent"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/spf13/cobra"
)

// safeConvIDPattern only allows alphanumeric characters, hyphens, and underscores.
// This prevents path-traversal attacks when convID is used in filesystem paths.
var safeConvIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// agentChatTimeout is the per-request timeout for agent chat commands.
// Agent conversations involve multi-turn LLM calls + tool executions and can
// take several minutes, so we use a much longer timeout than the default 30s.
const agentChatTimeout = 10 * time.Minute

func NewAgentCommand(root *RootCommand) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "AI agent operator commands",
		Long: `Interact with the AIMA AI agent operator.

The agent can manage your inference infrastructure by understanding natural
language requests and using AIMA platform tools to fulfil them.`,
	}

	cmd.AddCommand(NewAgentChatCommand(root))
	cmd.AddCommand(NewAgentAskCommand(root))
	cmd.AddCommand(NewAgentStatusCommand(root))
	cmd.AddCommand(NewAgentHistoryCommand(root))
	cmd.AddCommand(NewAgentResetCommand(root))

	return cmd
}

// NewAgentChatCommand starts an interactive chat session with the agent.
func NewAgentChatCommand(root *RootCommand) *cobra.Command {
	var conversationID string

	cmd := &cobra.Command{
		Use:   "chat <message>",
		Short: "Send a message to the AI agent",
		Long: `Send a message to the AIMA AI agent and receive a response.

The agent has access to all AIMA platform tools and can manage models,
engines, services, and more based on your natural language requests.`,
		Example: `  # Start a new conversation
  aima agent chat "List all deployed models"

  # Continue an existing conversation
  aima agent chat --conversation conv-abc123 "Deploy the llama3 model"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAgentChat(cmd.Context(), root, args[0], conversationID)
		},
	}

	cmd.Flags().StringVarP(&conversationID, "conversation", "c", "", "Conversation ID to continue")

	return cmd
}

func runAgentChat(ctx context.Context, root *RootCommand, message, conversationID string) error {
	gw := root.Gateway()
	opts := root.OutputOptions()

	// Load persisted conversation into agent memory before dispatching.
	if conversationID != "" && root.Agent() != nil && root.DataDir() != "" {
		if conv, err := loadConversationFromFile(root.DataDir(), conversationID); err == nil && conv != nil {
			root.Agent().InjectConversation(conv)
		}
	}

	input := map[string]any{"message": message}
	if conversationID != "" {
		input["conversation_id"] = conversationID
	}

	req := &gateway.Request{
		Type:  gateway.TypeCommand,
		Unit:  "agent.chat",
		Input: input,
		Options: gateway.RequestOptions{
			// Override the default 30s gateway timeout: agent conversations
			// require multiple LLM+tool-execution turns and can take minutes.
			Timeout: agentChatTimeout,
		},
	}

	resp := gw.Handle(ctx, req)
	if !resp.Success {
		errMsg := fmt.Sprintf("%s: %s", resp.Error.Code, resp.Error.Message)
		if resp.Error.Details != nil {
			errMsg = fmt.Sprintf("%s\ndetails: %v", errMsg, resp.Error.Details)
		}
		PrintError(fmt.Errorf("%s", errMsg), opts)
		return fmt.Errorf("agent chat failed: %s", resp.Error.Message)
	}

	// Print the agent response as plain text for readability.
	if m, ok := resp.Data.(map[string]any); ok {
		if reply, ok := m["response"].(string); ok {
			fmt.Fprintln(opts.Writer, reply)
			var convID string
			if id, ok := m["conversation_id"].(string); ok {
				convID = id
				fmt.Fprintf(opts.Writer, "\n[conversation: %s]\n", convID)
			}

			// Persist the updated conversation to disk.
			if convID != "" && root.Agent() != nil && root.DataDir() != "" {
				if conv := root.Agent().GetConversation(convID); conv != nil {
					if err := saveConversationToFile(root.DataDir(), convID, conv); err != nil {
						slog.Warn("failed to persist conversation", "id", convID, "error", err)
					}
				}
			}
			return nil
		}
	}

	return PrintOutput(resp.Data, opts)
}

// loadConversationFromFile reads a persisted conversation from ~/.aima/conversations/<id>.json.
// Returns (nil, nil) when the file does not exist.
func loadConversationFromFile(dataDir, convID string) (*coreagent.Conversation, error) {
	if !safeConvIDPattern.MatchString(convID) {
		return nil, fmt.Errorf("invalid conversation ID %q: only alphanumeric, hyphens, and underscores are allowed", convID)
	}
	path := filepath.Join(dataDir, "conversations", convID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var conv coreagent.Conversation
	if err := json.Unmarshal(data, &conv); err != nil {
		return nil, err
	}
	return &conv, nil
}

// saveConversationToFile persists a conversation to ~/.aima/conversations/<id>.json.
// Uses an atomic write (temp file + rename) to avoid partial writes on crash.
func saveConversationToFile(dataDir, convID string, conv *coreagent.Conversation) error {
	if !safeConvIDPattern.MatchString(convID) {
		return fmt.Errorf("invalid conversation ID %q: only alphanumeric, hyphens, and underscores are allowed", convID)
	}
	convDir := filepath.Join(dataDir, "conversations")
	if err := os.MkdirAll(convDir, 0755); err != nil {
		return err
	}
	data, err := json.Marshal(conv)
	if err != nil {
		return err
	}
	dest := filepath.Join(convDir, convID+".json")
	tmp := dest + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, dest)
}

// NewAgentAskCommand is a one-shot version of chat (no conversation tracking).
func NewAgentAskCommand(root *RootCommand) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ask <message>",
		Short: "Ask the AI agent a one-shot question",
		Long: `Ask the AIMA AI agent a question without starting a persistent conversation.

Each ask creates a new conversation that is discarded after the response.`,
		Example: `  aima agent ask "What is the current GPU utilization?"`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAgentChat(cmd.Context(), root, args[0], "")
		},
	}

	return cmd
}

// NewAgentStatusCommand shows the agent operator status.
func NewAgentStatusCommand(root *RootCommand) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show the AI agent operator status",
		Long:  `Show whether the AI agent is enabled and which LLM it uses.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAgentStatus(cmd.Context(), root)
		},
	}

	return cmd
}

func runAgentStatus(ctx context.Context, root *RootCommand) error {
	gw := root.Gateway()
	opts := root.OutputOptions()

	req := &gateway.Request{
		Type:  gateway.TypeQuery,
		Unit:  "agent.status",
		Input: map[string]any{},
	}

	resp := gw.Handle(ctx, req)
	if !resp.Success {
		PrintError(fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message), opts)
		return fmt.Errorf("agent status failed: %s", resp.Error.Message)
	}

	return PrintOutput(resp.Data, opts)
}

// NewAgentHistoryCommand shows conversation message history.
func NewAgentHistoryCommand(root *RootCommand) *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "history <conversation-id>",
		Short: "Show conversation history",
		Long:  `Retrieve the message history for a specific conversation.`,
		Example: `  # Show full history
  aima agent history conv-abc123

  # Show last 5 messages
  aima agent history conv-abc123 --limit 5`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAgentHistory(cmd.Context(), root, args[0], limit)
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 0, "Maximum number of messages to show (0 = all)")

	return cmd
}

func runAgentHistory(ctx context.Context, root *RootCommand, conversationID string, limit int) error {
	gw := root.Gateway()
	opts := root.OutputOptions()

	input := map[string]any{"conversation_id": conversationID}
	if limit > 0 {
		input["limit"] = limit
	}

	req := &gateway.Request{
		Type:  gateway.TypeQuery,
		Unit:  "agent.history",
		Input: input,
	}

	resp := gw.Handle(ctx, req)
	if !resp.Success {
		PrintError(fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message), opts)
		return fmt.Errorf("agent history failed: %s", resp.Error.Message)
	}

	return PrintOutput(resp.Data, opts)
}

// NewAgentResetCommand clears a conversation.
func NewAgentResetCommand(root *RootCommand) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset <conversation-id>",
		Short: "Reset (clear) a conversation",
		Long:  `Delete all messages in a conversation and remove it from memory.`,
		Example: `  aima agent reset conv-abc123`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAgentReset(cmd.Context(), root, args[0])
		},
	}

	return cmd
}

func runAgentReset(ctx context.Context, root *RootCommand, conversationID string) error {
	gw := root.Gateway()
	opts := root.OutputOptions()

	req := &gateway.Request{
		Type: gateway.TypeCommand,
		Unit: "agent.reset",
		Input: map[string]any{
			"conversation_id": conversationID,
		},
	}

	resp := gw.Handle(ctx, req)
	if !resp.Success {
		PrintError(fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message), opts)
		return fmt.Errorf("agent reset failed: %s", resp.Error.Message)
	}

	// Also remove the persisted conversation file if it exists.
	if root.DataDir() != "" {
		convFile := filepath.Join(root.DataDir(), "conversations", conversationID+".json")
		if err := os.Remove(convFile); err != nil && !os.IsNotExist(err) {
			slog.Warn("failed to remove conversation file", "id", conversationID, "error", err)
		}
	}

	PrintSuccess(fmt.Sprintf("Conversation %s reset", conversationID), opts)
	return nil
}
