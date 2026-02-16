package gateway

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type MCPToolDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	InputSchema map[string]any `json:"inputSchema"`
}

type MCPToolsListResult struct {
	Tools []MCPToolDefinition `json:"tools"`
}

type MCPToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

type MCPToolResult struct {
	Content []MCPContentBlock `json:"content"`
	IsError bool              `json:"isError,omitempty"`
}

type MCPContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

func (a *MCPAdapter) handleToolsList(ctx context.Context, req *MCPRequest) *MCPResponse {
	tools := a.GenerateToolDefinitions()
	result := &MCPToolsListResult{
		Tools: tools,
	}
	return a.successResponse(req.ID, result)
}

func (a *MCPAdapter) handleToolsCall(ctx context.Context, req *MCPRequest) *MCPResponse {
	var params MCPToolCallParams
	if len(req.Params) > 0 {
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return a.errorResponse(req.ID, MCPErrorCodeInvalidParams, "invalid tool call params: "+err.Error())
		}
	}

	if params.Name == "" {
		return a.errorResponse(req.ID, MCPErrorCodeInvalidParams, "tool name is required")
	}

	result, err := a.ExecuteTool(ctx, params.Name, params.Arguments)
	if err != nil {
		return a.errorResponseWithData(req.ID, MCPErrorCodeToolExecution, err.Error(), nil)
	}

	return a.successResponse(req.ID, result)
}

func (a *MCPAdapter) GenerateToolDefinitions() []MCPToolDefinition {
	var tools []MCPToolDefinition

	if a.gateway == nil || a.gateway.Registry() == nil {
		return tools
	}

	registry := a.gateway.Registry()

	for _, cmd := range registry.ListCommands() {
		tool := a.commandToToolDefinition(cmd)
		tools = append(tools, tool)
	}

	for _, q := range registry.ListQueries() {
		tool := a.queryToToolDefinition(q)
		tools = append(tools, tool)
	}

	return tools
}

func (a *MCPAdapter) commandToToolDefinition(cmd unit.Command) MCPToolDefinition {
	inputSchema := schemaToMCPInputSchema(cmd.InputSchema())

	return MCPToolDefinition{
		Name:        unitNameToToolName(cmd.Name()),
		Description: cmd.Description(),
		InputSchema: inputSchema,
	}
}

func (a *MCPAdapter) queryToToolDefinition(q unit.Query) MCPToolDefinition {
	inputSchema := schemaToMCPInputSchema(q.InputSchema())

	return MCPToolDefinition{
		Name:        unitNameToToolName(q.Name()),
		Description: q.Description(),
		InputSchema: inputSchema,
	}
}

func schemaToMCPInputSchema(schema unit.Schema) map[string]any {
	result := map[string]any{
		"type":       schema.Type,
		"properties": map[string]any{},
	}

	if len(schema.Properties) > 0 {
		props := make(map[string]any)
		for name, field := range schema.Properties {
			props[name] = fieldSchemaToMap(field.Schema)
		}
		result["properties"] = props
	}

	if len(schema.Required) > 0 {
		result["required"] = schema.Required
	}

	if schema.Description != "" {
		result["description"] = schema.Description
	}

	if len(schema.Enum) > 0 {
		result["enum"] = schema.Enum
	}

	return result
}

func fieldSchemaToMap(schema unit.Schema) map[string]any {
	result := map[string]any{
		"type": schema.Type,
	}

	if schema.Description != "" {
		result["description"] = schema.Description
	}

	if len(schema.Enum) > 0 {
		result["enum"] = schema.Enum
	}

	if schema.Min != nil {
		result["minimum"] = *schema.Min
	}

	if schema.Max != nil {
		result["maximum"] = *schema.Max
	}

	if schema.MinLength != nil {
		result["minLength"] = *schema.MinLength
	}

	if schema.MaxLength != nil {
		result["maxLength"] = *schema.MaxLength
	}

	if schema.Pattern != "" {
		result["pattern"] = schema.Pattern
	}

	if schema.Default != nil {
		result["default"] = schema.Default
	}

	if schema.Items != nil {
		result["items"] = fieldSchemaToMap(*schema.Items)
	}

	if len(schema.Properties) > 0 {
		props := make(map[string]any)
		for name, field := range schema.Properties {
			props[name] = fieldSchemaToMap(field.Schema)
		}
		result["properties"] = props
	}

	if len(schema.Required) > 0 {
		result["required"] = schema.Required
	}

	return result
}

func (a *MCPAdapter) ExecuteTool(ctx context.Context, name string, arguments json.RawMessage) (*MCPToolResult, error) {
	if a.gateway == nil {
		return nil, fmt.Errorf("gateway not initialized")
	}

	unitName := toolNameToUnitName(name)

	var input map[string]any
	if len(arguments) > 0 {
		if err := json.Unmarshal(arguments, &input); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
	}

	registry := a.gateway.Registry()
	if registry == nil {
		return nil, fmt.Errorf("registry not initialized")
	}

	var reqType string
	if registry.GetCommand(unitName) != nil {
		reqType = TypeCommand
	} else if registry.GetQuery(unitName) != nil {
		reqType = TypeQuery
	} else {
		return nil, fmt.Errorf("tool not found: %s", name)
	}

	req := &Request{
		Type:  reqType,
		Unit:  unitName,
		Input: input,
	}

	resp := a.gateway.Handle(ctx, req)

	result := &MCPToolResult{
		Content: []MCPContentBlock{},
	}

	if !resp.Success {
		result.IsError = true
		errMsg := "unknown error"
		if resp.Error != nil {
			errMsg = resp.Error.Message
			if resp.Error.Details != nil {
				errMsg = fmt.Sprintf("%s: %v", errMsg, resp.Error.Details)
			}
		}
		result.Content = append(result.Content, MCPContentBlock{
			Type: "text",
			Text: errMsg,
		})
		return result, nil
	}

	outputJSON, err := json.MarshalIndent(resp.Data, "", "  ")
	if err != nil {
		outputJSON, _ = json.Marshal(resp.Data)
	}

	result.Content = append(result.Content, MCPContentBlock{
		Type: "text",
		Text: string(outputJSON),
	})

	return result, nil
}
