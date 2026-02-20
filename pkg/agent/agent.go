// Package agent implements the AI Agent Operator domain.
// The agent drives a conversation loop with an LLM, using AIMA's MCP tools
// as its capability set, and loads relevant Skills into the system prompt.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	agentllm "github.com/jguan/ai-inference-managed-by-ai/pkg/agent/llm"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/skill"
)

const (
	// maxToolCallRounds prevents infinite loops in the tool-call cycle.
	maxToolCallRounds = 10

	baseSystemPrompt = `You are AIMA (AI Inference Managed by AI), an intelligent assistant that manages AI inference infrastructure. You help users deploy models, manage inference engines, monitor resources, and optimize their AI workloads.

You have access to a set of tools that correspond to AIMA platform operations. Use them to fulfil the user's requests. When uncertain, check the current state first before making changes. Always confirm destructive operations with the user.`
)

// Agent is the core AI agent operator.
type Agent struct {
	llm           agentllm.LLMClient
	mcpAdapter    *gateway.MCPAdapter
	skillRegistry *skill.SkillRegistry
	conversations *ConversationStore
	opts          AgentOptions
}

// AgentOptions holds configuration for the Agent.
type AgentOptions struct {
	// MaxTokens for each LLM call.
	MaxTokens int
	// Temperature for LLM sampling.
	Temperature float64
}

// NewAgent creates a new Agent.
func NewAgent(llm agentllm.LLMClient, mcpAdapter *gateway.MCPAdapter, skillRegistry *skill.SkillRegistry, opts AgentOptions) *Agent {
	if opts.MaxTokens <= 0 {
		opts.MaxTokens = 4096
	}
	return &Agent{
		llm:           llm,
		mcpAdapter:    mcpAdapter,
		skillRegistry: skillRegistry,
		conversations: NewConversationStore(),
		opts:          opts,
	}
}

// Chat handles a single user message within a conversation and returns the agent's reply.
// If conversationID is empty, a new conversation is created and its ID is returned in the error-free path.
func (a *Agent) Chat(ctx context.Context, conversationID, userMessage string) (string, string, error) {
	if a.llm == nil {
		return "", "", fmt.Errorf("LLM client is not configured")
	}
	if userMessage == "" {
		return "", "", fmt.Errorf("message is required")
	}

	// 1. Get or create conversation.
	conv := a.conversations.GetOrCreate(conversationID)
	conv.addMessage(agentllm.Message{Role: "user", Content: userMessage})

	// 2. Build system prompt (base + always-on skills + matched skills).
	systemPrompt := a.buildSystemPrompt(ctx, userMessage)

	// 3. Get available tools from MCP adapter.
	toolDefs := a.mcpToolsToLLMTools()

	chatOpts := agentllm.ChatOptions{
		MaxTokens:   a.opts.MaxTokens,
		Temperature: a.opts.Temperature,
	}

	// 4. Conversation loop (handle multiple rounds of tool calls).
	for round := 0; round < maxToolCallRounds; round++ {
		// Prepend system message for each call.
		msgs := make([]agentllm.Message, 0, len(conv.Messages)+1)
		msgs = append(msgs, agentllm.Message{Role: "system", Content: systemPrompt})
		msgs = append(msgs, conv.Messages...)

		response, err := a.llm.Chat(ctx, msgs, toolDefs, chatOpts)
		if err != nil {
			return "", conv.ID, fmt.Errorf("LLM error: %w", err)
		}

		// Add assistant message to history.
		conv.addMessage(response.Message)

		// 5. If no tool calls, return the text response.
		if len(response.ToolCalls) == 0 {
			slog.Debug("agent chat complete", "conversation_id", conv.ID, "rounds", round+1)
			return response.Message.Content, conv.ID, nil
		}

		// 6. Execute each tool call via the MCP adapter.
		slog.Debug("agent executing tool calls",
			"conversation_id", conv.ID,
			"tool_count", len(response.ToolCalls),
			"round", round+1,
		)

		for _, tc := range response.ToolCalls {
			result := a.executeTool(ctx, tc)
			conv.addMessage(agentllm.Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
			})
		}
		// Continue loop: LLM sees tool results and may reply or call more tools.
	}

	return "", conv.ID, fmt.Errorf("exceeded maximum tool call rounds (%d)", maxToolCallRounds)
}

// ResetConversation clears all messages in a conversation.
func (a *Agent) ResetConversation(conversationID string) bool {
	return a.conversations.Delete(conversationID)
}

// GetConversation returns a conversation by ID.
func (a *Agent) GetConversation(id string) *Conversation {
	return a.conversations.Get(id)
}

// ActiveConversationCount returns the number of live conversations.
func (a *Agent) ActiveConversationCount() int {
	return a.conversations.Count()
}

// ListConversations returns a snapshot of all active conversations.
func (a *Agent) ListConversations() []*Conversation {
	return a.conversations.List()
}

// LLMName returns the provider name of the configured LLM.
func (a *Agent) LLMName() string {
	if a.llm == nil {
		return ""
	}
	return a.llm.Name()
}

// LLMModelName returns the model identifier of the configured LLM.
func (a *Agent) LLMModelName() string {
	if a.llm == nil {
		return ""
	}
	return a.llm.ModelName()
}

// buildSystemPrompt assembles the full system prompt for a conversation turn.
func (a *Agent) buildSystemPrompt(ctx context.Context, userMessage string) string {
	var sb strings.Builder
	sb.WriteString(baseSystemPrompt)
	sb.WriteString("\n\n")

	if a.skillRegistry == nil {
		return sb.String()
	}

	// Load always-on skills.
	alwaysOn, err := a.skillRegistry.GetAlwaysOnSkills(ctx)
	if err != nil {
		slog.Warn("failed to load always-on skills", "error", err)
	}

	// Load message-triggered skills.
	matched, err := a.skillRegistry.MatchSkills(ctx, userMessage, nil)
	if err != nil {
		slog.Warn("failed to match skills", "error", err)
	}

	// Combine and deduplicate by ID.
	seen := make(map[string]struct{})
	var skills []skill.Skill
	for _, s := range alwaysOn {
		if _, ok := seen[s.ID]; !ok {
			seen[s.ID] = struct{}{}
			skills = append(skills, s)
		}
	}
	for _, s := range matched {
		if _, ok := seen[s.ID]; !ok {
			seen[s.ID] = struct{}{}
			skills = append(skills, s)
		}
	}

	if len(skills) > 0 {
		sb.WriteString(a.skillRegistry.FormatForSystemPrompt(skills))
	}

	return sb.String()
}

// mcpToolsToLLMTools converts MCP tool definitions to the LLM ToolDef format.
func (a *Agent) mcpToolsToLLMTools() []agentllm.ToolDef {
	if a.mcpAdapter == nil {
		return nil
	}
	mcpTools := a.mcpAdapter.GenerateToolDefinitions()
	tools := make([]agentllm.ToolDef, 0, len(mcpTools))
	for _, t := range mcpTools {
		tools = append(tools, agentllm.ToolDef{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		})
	}
	return tools
}

// executeTool runs a single tool call via the MCP adapter and returns the result as a string.
func (a *Agent) executeTool(ctx context.Context, tc agentllm.ToolCall) string {
	if a.mcpAdapter == nil {
		return `{"error": "MCP adapter not available"}`
	}

	argsJSON, err := json.Marshal(tc.Arguments)
	if err != nil {
		return fmt.Sprintf(`{"error": "failed to marshal arguments: %s"}`, err)
	}

	result, err := a.mcpAdapter.ExecuteTool(ctx, tc.Name, argsJSON)
	if err != nil {
		slog.Warn("agent tool call failed",
			"tool", tc.Name,
			"error", err,
		)
		return fmt.Sprintf(`{"error": "%s"}`, err)
	}

	if result == nil || len(result.Content) == 0 {
		return "{}"
	}

	// Concatenate all text content blocks.
	var parts []string
	for _, block := range result.Content {
		if block.Text != "" {
			parts = append(parts, block.Text)
		}
	}
	return strings.Join(parts, "\n")
}

