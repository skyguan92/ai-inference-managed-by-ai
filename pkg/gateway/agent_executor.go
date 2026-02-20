package gateway

import (
	"context"
	"encoding/json"

	coreagent "github.com/jguan/ai-inference-managed-by-ai/pkg/agent"
)

// AgentExecutorAdapter bridges MCPAdapter to the coreagent.ToolExecutor interface.
// MCPAdapter and coreagent.ToolExecutor use structurally identical but nominally
// distinct types; this adapter performs the explicit field-by-field conversion.
type AgentExecutorAdapter struct {
	adapter *MCPAdapter
}

// NewAgentExecutorAdapter wraps an MCPAdapter so it can be used as an
// coreagent.ToolExecutor by the AI Agent Operator.
func NewAgentExecutorAdapter(adapter *MCPAdapter) *AgentExecutorAdapter {
	return &AgentExecutorAdapter{adapter: adapter}
}

// GenerateToolDefinitions converts MCPToolDefinition → coreagent.ToolDefinition.
func (e *AgentExecutorAdapter) GenerateToolDefinitions() []coreagent.ToolDefinition {
	mcpDefs := e.adapter.GenerateToolDefinitions()
	defs := make([]coreagent.ToolDefinition, 0, len(mcpDefs))
	for _, d := range mcpDefs {
		defs = append(defs, coreagent.ToolDefinition{
			Name:        d.Name,
			Description: d.Description,
			InputSchema: d.InputSchema,
		})
	}
	return defs
}

// ExecuteTool delegates to MCPAdapter and converts the result type.
func (e *AgentExecutorAdapter) ExecuteTool(ctx context.Context, name string, arguments json.RawMessage) (*coreagent.ToolResult, error) {
	result, err := e.adapter.ExecuteTool(ctx, name, arguments)
	if err != nil {
		return nil, err
	}

	content := make([]coreagent.ContentBlock, 0, len(result.Content))
	for _, c := range result.Content {
		content = append(content, coreagent.ContentBlock{
			Type: c.Type,
			Text: c.Text,
		})
	}

	return &coreagent.ToolResult{
		Content: content,
		IsError: result.IsError,
	}, nil
}
