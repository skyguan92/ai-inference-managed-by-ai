package cli

import (
	"context"
	"fmt"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/spf13/cobra"
)

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

	input := map[string]any{"message": message}
	if conversationID != "" {
		input["conversation_id"] = conversationID
	}

	req := &gateway.Request{
		Type:  gateway.TypeCommand,
		Unit:  "agent.chat",
		Input: input,
	}

	resp := gw.Handle(ctx, req)
	if !resp.Success {
		PrintError(fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message), opts)
		return fmt.Errorf("agent chat failed: %s", resp.Error.Message)
	}

	// Print the agent response as plain text for readability.
	if m, ok := resp.Data.(map[string]any); ok {
		if reply, ok := m["response"].(string); ok {
			fmt.Fprintln(opts.Writer, reply)
			if convID, ok := m["conversation_id"].(string); ok {
				fmt.Fprintf(opts.Writer, "\n[conversation: %s]\n", convID)
			}
			return nil
		}
	}

	return PrintOutput(resp.Data, opts)
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

	PrintSuccess(fmt.Sprintf("Conversation %s reset", conversationID), opts)
	return nil
}
