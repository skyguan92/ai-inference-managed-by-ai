package gateway

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestMCPAdapter_GenerateToolDefinitions(t *testing.T) {
	t.Run("with commands and queries", func(t *testing.T) {
		reg := unit.NewRegistry()

		cmd := &mockCommand{
			name:    "model.pull",
			domain:  "model",
			execute: nil,
		}
		_ = reg.RegisterCommand(cmd)

		query := &mockQuery{
			name:    "model.list",
			domain:  "model",
			execute: nil,
		}
		_ = reg.RegisterQuery(query)

		g := NewGateway(reg)
		adapter := NewMCPAdapter(g)

		tools := adapter.GenerateToolDefinitions()

		if len(tools) != 2 {
			t.Errorf("expected 2 tools, got %d", len(tools))
		}

		toolNames := make(map[string]bool)
		for _, tool := range tools {
			toolNames[tool.Name] = true
		}

		if !toolNames["model_pull"] {
			t.Error("expected tool model_pull")
		}
		if !toolNames["model_list"] {
			t.Error("expected tool model_list")
		}
	})

	t.Run("with nil gateway", func(t *testing.T) {
		adapter := NewMCPAdapter(nil)
		tools := adapter.GenerateToolDefinitions()

		if len(tools) != 0 {
			t.Errorf("expected 0 tools, got %d", len(tools))
		}
	})

	t.Run("with empty registry", func(t *testing.T) {
		g := NewGateway(nil)
		adapter := NewMCPAdapter(g)
		tools := adapter.GenerateToolDefinitions()

		if len(tools) != 0 {
			t.Errorf("expected 0 tools, got %d", len(tools))
		}
	})
}

func TestMCPAdapter_commandToToolDefinition(t *testing.T) {
	reg := unit.NewRegistry()
	g := NewGateway(reg)
	adapter := NewMCPAdapter(g)

	cmd := &mockCommandWithSchema{
		name:   "test.command",
		domain: "test",
		desc:   "test command description",
		inputSchema: unit.Schema{
			Type:        "object",
			Description: "input schema",
			Properties: map[string]unit.Field{
				"source": {
					Name: "source",
					Schema: unit.Schema{
						Type:        "string",
						Description: "source name",
						Enum:        []any{"ollama", "huggingface"},
					},
				},
				"repo": {
					Name: "repo",
					Schema: unit.Schema{
						Type:      "string",
						MinLength: ptrInt(1),
					},
				},
			},
			Required: []string{"source", "repo"},
		},
		outputSchema: unit.Schema{
			Type: "object",
		},
	}

	tool := adapter.commandToToolDefinition(cmd)

	if tool.Name != "test_command" {
		t.Errorf("expected name test_command, got %s", tool.Name)
	}

	if tool.Description != "test command description" {
		t.Errorf("expected description 'test command description', got %s", tool.Description)
	}

	if tool.InputSchema["type"] != "object" {
		t.Errorf("expected type object, got %v", tool.InputSchema["type"])
	}

	props, ok := tool.InputSchema["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties map")
	}

	if len(props) != 2 {
		t.Errorf("expected 2 properties, got %d", len(props))
	}

	required, ok := tool.InputSchema["required"].([]string)
	if !ok {
		t.Fatal("expected required array")
	}

	if len(required) != 2 {
		t.Errorf("expected 2 required fields, got %d", len(required))
	}
}

func TestMCPAdapter_queryToToolDefinition(t *testing.T) {
	reg := unit.NewRegistry()
	g := NewGateway(reg)
	adapter := NewMCPAdapter(g)

	query := &mockQueryWithSchema{
		name:   "test.query",
		domain: "test",
		desc:   "test query description",
		inputSchema: unit.Schema{
			Type: "object",
			Properties: map[string]unit.Field{
				"id": {
					Name: "id",
					Schema: unit.Schema{
						Type: "string",
					},
				},
			},
			Required: []string{"id"},
		},
		outputSchema: unit.Schema{
			Type: "object",
		},
	}

	tool := adapter.queryToToolDefinition(query)

	if tool.Name != "test_query" {
		t.Errorf("expected name test_query, got %s", tool.Name)
	}

	if tool.Description != "test query description" {
		t.Errorf("expected description 'test query description', got %s", tool.Description)
	}
}

func TestMCPAdapter_handleToolsList(t *testing.T) {
	reg := unit.NewRegistry()
	cmd := &mockCommand{name: "test.cmd", domain: "test"}
	_ = reg.RegisterCommand(cmd)

	g := NewGateway(reg)
	adapter := NewMCPAdapter(g)

	req := &MCPRequest{
		JSONRPC: JSONRPC,
		ID:      1,
		Method:  "tools/list",
	}

	resp := adapter.HandleRequest(context.Background(), req)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(*MCPToolsListResult)
	if !ok {
		t.Fatalf("expected MCPToolsListResult, got %T", resp.Result)
	}

	if len(result.Tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(result.Tools))
	}

	if result.Tools[0].Name != "test_cmd" {
		t.Errorf("expected tool name test_cmd, got %s", result.Tools[0].Name)
	}
}

func TestMCPAdapter_handleToolsCall(t *testing.T) {
	t.Run("successful call", func(t *testing.T) {
		reg := unit.NewRegistry()
		cmd := &mockCommand{
			name:   "test.echo",
			domain: "test",
			execute: func(ctx context.Context, input any) (any, error) {
				return map[string]any{"echo": input}, nil
			},
		}
		_ = reg.RegisterCommand(cmd)

		g := NewGateway(reg)
		adapter := NewMCPAdapter(g)

		req := &MCPRequest{
			JSONRPC: JSONRPC,
			ID:      1,
			Method:  "tools/call",
			Params:  json.RawMessage(`{"name":"test_echo","arguments":{"msg":"hello"}}`),
		}

		resp := adapter.HandleRequest(context.Background(), req)
		if resp == nil {
			t.Fatal("expected response, got nil")
		}

		if resp.Error != nil {
			t.Fatalf("unexpected error: %v", resp.Error)
		}

		result, ok := resp.Result.(*MCPToolResult)
		if !ok {
			t.Fatalf("expected MCPToolResult, got %T", resp.Result)
		}

		if len(result.Content) != 1 {
			t.Fatalf("expected 1 content block, got %d", len(result.Content))
		}

		if result.Content[0].Type != "text" {
			t.Errorf("expected content type text, got %s", result.Content[0].Type)
		}

		if result.IsError {
			t.Error("expected isError to be false")
		}
	})

	t.Run("invalid params", func(t *testing.T) {
		g := NewGateway(nil)
		adapter := NewMCPAdapter(g)

		req := &MCPRequest{
			JSONRPC: JSONRPC,
			ID:      1,
			Method:  "tools/call",
			Params:  json.RawMessage(`invalid json`),
		}

		resp := adapter.HandleRequest(context.Background(), req)
		if resp == nil {
			t.Fatal("expected response, got nil")
		}

		if resp.Error == nil {
			t.Fatal("expected error")
		}

		if resp.Error.Code != MCPErrorCodeInvalidParams {
			t.Errorf("expected error code %d, got %d", MCPErrorCodeInvalidParams, resp.Error.Code)
		}
	})

	t.Run("empty tool name", func(t *testing.T) {
		g := NewGateway(nil)
		adapter := NewMCPAdapter(g)

		req := &MCPRequest{
			JSONRPC: JSONRPC,
			ID:      1,
			Method:  "tools/call",
			Params:  json.RawMessage(`{"name":""}`),
		}

		resp := adapter.HandleRequest(context.Background(), req)
		if resp == nil {
			t.Fatal("expected response, got nil")
		}

		if resp.Error == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("tool not found", func(t *testing.T) {
		g := NewGateway(nil)
		adapter := NewMCPAdapter(g)

		req := &MCPRequest{
			JSONRPC: JSONRPC,
			ID:      1,
			Method:  "tools/call",
			Params:  json.RawMessage(`{"name":"unknown_tool"}`),
		}

		resp := adapter.HandleRequest(context.Background(), req)
		if resp == nil {
			t.Fatal("expected response, got nil")
		}

		if resp.Error == nil {
			t.Fatal("expected error")
		}

		if resp.Error.Code != MCPErrorCodeToolExecution {
			t.Errorf("expected error code %d, got %d", MCPErrorCodeToolExecution, resp.Error.Code)
		}
	})

	t.Run("execution error", func(t *testing.T) {
		reg := unit.NewRegistry()
		cmd := &mockCommand{
			name:   "test.error",
			domain: "test",
			execute: func(ctx context.Context, input any) (any, error) {
				return nil, &ErrorInfo{Code: "TEST_ERROR", Message: "test error"}
			},
		}
		_ = reg.RegisterCommand(cmd)

		g := NewGateway(reg)
		adapter := NewMCPAdapter(g)

		req := &MCPRequest{
			JSONRPC: JSONRPC,
			ID:      1,
			Method:  "tools/call",
			Params:  json.RawMessage(`{"name":"test_error"}`),
		}

		resp := adapter.HandleRequest(context.Background(), req)
		if resp == nil {
			t.Fatal("expected response, got nil")
		}

		result, ok := resp.Result.(*MCPToolResult)
		if !ok {
			t.Fatalf("expected MCPToolResult, got %T", resp.Result)
		}

		if !result.IsError {
			t.Error("expected isError to be true")
		}

		if len(result.Content) != 1 {
			t.Fatalf("expected 1 content block, got %d", len(result.Content))
		}

		if result.Content[0].Text == "" {
			t.Error("expected error message in content")
		}
	})
}

func TestMCPAdapter_ExecuteTool(t *testing.T) {
	t.Run("with nil gateway", func(t *testing.T) {
		adapter := NewMCPAdapter(nil)
		_, err := adapter.ExecuteTool(context.Background(), "test_tool", nil)
		if err == nil {
			t.Error("expected error")
		}
	})

	t.Run("with query", func(t *testing.T) {
		reg := unit.NewRegistry()
		query := &mockQuery{
			name:   "test.get",
			domain: "test",
			execute: func(ctx context.Context, input any) (any, error) {
				return map[string]any{"result": "ok"}, nil
			},
		}
		_ = reg.RegisterQuery(query)

		g := NewGateway(reg)
		adapter := NewMCPAdapter(g)

		result, err := adapter.ExecuteTool(context.Background(), "test_get", json.RawMessage(`{}`))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.IsError {
			t.Error("expected isError to be false")
		}
	})

	t.Run("with invalid arguments", func(t *testing.T) {
		reg := unit.NewRegistry()
		cmd := &mockCommand{name: "test.cmd", domain: "test"}
		_ = reg.RegisterCommand(cmd)

		g := NewGateway(reg)
		adapter := NewMCPAdapter(g)

		_, err := adapter.ExecuteTool(context.Background(), "test_cmd", json.RawMessage(`invalid`))
		if err == nil {
			t.Error("expected error for invalid arguments")
		}
	})
}

func TestSchemaToMCPInputSchema(t *testing.T) {
	t.Run("simple schema", func(t *testing.T) {
		schema := unit.Schema{
			Type:        "object",
			Description: "test schema",
			Properties: map[string]unit.Field{
				"name": {
					Name: "name",
					Schema: unit.Schema{
						Type: "string",
					},
				},
			},
			Required: []string{"name"},
		}

		result := schemaToMCPInputSchema(schema)

		if result["type"] != "object" {
			t.Errorf("expected type object, got %v", result["type"])
		}

		if result["description"] != "test schema" {
			t.Errorf("expected description 'test schema', got %v", result["description"])
		}

		props, ok := result["properties"].(map[string]any)
		if !ok {
			t.Fatal("expected properties map")
		}

		if _, ok := props["name"]; !ok {
			t.Error("expected name property")
		}
	})

	t.Run("schema with enum", func(t *testing.T) {
		schema := unit.Schema{
			Type: "string",
			Enum: []any{"a", "b", "c"},
		}

		result := schemaToMCPInputSchema(schema)

		enum, ok := result["enum"].([]any)
		if !ok {
			t.Fatal("expected enum array")
		}

		if len(enum) != 3 {
			t.Errorf("expected 3 enum values, got %d", len(enum))
		}
	})

	t.Run("schema with min/max in field", func(t *testing.T) {
		min := 0.0
		max := 100.0
		schema := unit.Schema{
			Type: "object",
			Properties: map[string]unit.Field{
				"value": {
					Name: "value",
					Schema: unit.Schema{
						Type: "number",
						Min:  &min,
						Max:  &max,
					},
				},
			},
		}

		result := schemaToMCPInputSchema(schema)

		props, ok := result["properties"].(map[string]any)
		if !ok {
			t.Fatal("expected properties map")
		}

		valueField, ok := props["value"].(map[string]any)
		if !ok {
			t.Fatal("expected value field map")
		}

		if valueField["minimum"] != 0.0 {
			t.Errorf("expected minimum 0.0, got %v", valueField["minimum"])
		}

		if valueField["maximum"] != 100.0 {
			t.Errorf("expected maximum 100.0, got %v", valueField["maximum"])
		}
	})

	t.Run("empty properties", func(t *testing.T) {
		schema := unit.Schema{
			Type: "object",
		}

		result := schemaToMCPInputSchema(schema)

		props, ok := result["properties"].(map[string]any)
		if !ok {
			t.Fatal("expected properties map")
		}

		if len(props) != 0 {
			t.Errorf("expected empty properties, got %d", len(props))
		}
	})
}

type mockCommandWithSchema struct {
	name         string
	domain       string
	desc         string
	inputSchema  unit.Schema
	outputSchema unit.Schema
	execute      func(ctx context.Context, input any) (any, error)
}

func (m *mockCommandWithSchema) Name() string              { return m.name }
func (m *mockCommandWithSchema) Domain() string            { return m.domain }
func (m *mockCommandWithSchema) InputSchema() unit.Schema  { return m.inputSchema }
func (m *mockCommandWithSchema) OutputSchema() unit.Schema { return m.outputSchema }
func (m *mockCommandWithSchema) Execute(ctx context.Context, input any) (any, error) {
	if m.execute != nil {
		return m.execute(ctx, input)
	}
	return nil, nil
}
func (m *mockCommandWithSchema) Description() string      { return m.desc }
func (m *mockCommandWithSchema) Examples() []unit.Example { return nil }

type mockQueryWithSchema struct {
	name         string
	domain       string
	desc         string
	inputSchema  unit.Schema
	outputSchema unit.Schema
	execute      func(ctx context.Context, input any) (any, error)
}

func (m *mockQueryWithSchema) Name() string              { return m.name }
func (m *mockQueryWithSchema) Domain() string            { return m.domain }
func (m *mockQueryWithSchema) InputSchema() unit.Schema  { return m.inputSchema }
func (m *mockQueryWithSchema) OutputSchema() unit.Schema { return m.outputSchema }
func (m *mockQueryWithSchema) Execute(ctx context.Context, input any) (any, error) {
	if m.execute != nil {
		return m.execute(ctx, input)
	}
	return nil, nil
}
func (m *mockQueryWithSchema) Description() string      { return m.desc }
func (m *mockQueryWithSchema) Examples() []unit.Example { return nil }

func ptrInt(v int) *int {
	return &v
}
