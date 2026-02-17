package integration

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	registrypkg "github.com/jguan/ai-inference-managed-by-ai/pkg/registry"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestMCPAdapter(t *testing.T) *gateway.MCPAdapter {
	registry := unit.NewRegistry()
	err := registrypkg.RegisterAll(registry)
	require.NoError(t, err)
	gw := gateway.NewGateway(registry)
	return gateway.NewMCPAdapter(gw)
}

func TestMCPServerIntegrationInitialize(t *testing.T) {
	adapter := createTestMCPAdapter(t)

	resp := adapter.HandleRequest(context.Background(), &gateway.MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  json.RawMessage(`{}`),
	})

	require.NotNil(t, resp)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Result)

	result, ok := resp.Result.(*gateway.MCPInitializeResult)
	require.True(t, ok)
	assert.Equal(t, gateway.MCPVersion, result.ProtocolVersion)
	assert.Equal(t, gateway.MCPServerName, result.ServerInfo.Name)
	assert.NotEmpty(t, result.ServerInfo.Version)
	assert.NotNil(t, result.Capabilities.Tools)
	assert.NotNil(t, result.Capabilities.Resources)
	assert.NotNil(t, result.Capabilities.Prompts)
}

func TestMCPServerIntegrationInitializeWithClientInfo(t *testing.T) {
	adapter := createTestMCPAdapter(t)

	params := gateway.MCPInitializeParams{
		ProtocolVersion: "2024-11-05",
		ClientInfo: gateway.MCPImplementationInfo{
			Name:    "test-client",
			Version: "1.0.0",
		},
	}
	paramsBytes, _ := json.Marshal(params)

	resp := adapter.HandleRequest(context.Background(), &gateway.MCPRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "initialize",
		Params:  paramsBytes,
	})

	require.NotNil(t, resp)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Result)
}

func TestMCPServerIntegrationPing(t *testing.T) {
	adapter := createTestMCPAdapter(t)

	resp := adapter.HandleRequest(context.Background(), &gateway.MCPRequest{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "ping",
		Params:  json.RawMessage(`{}`),
	})

	require.NotNil(t, resp)
	assert.Nil(t, resp.Error)
	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.Equal(t, 3, resp.ID)
}

func TestMCPServerIntegrationToolsList(t *testing.T) {
	adapter := createTestMCPAdapter(t)

	adapter.HandleRequest(context.Background(), &gateway.MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  json.RawMessage(`{}`),
	})

	resp := adapter.HandleRequest(context.Background(), &gateway.MCPRequest{
		JSONRPC: "2.0",
		ID:      4,
		Method:  "tools/list",
		Params:  json.RawMessage(`{}`),
	})

	require.NotNil(t, resp)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Result)

	result, ok := resp.Result.(*gateway.MCPToolsListResult)
	require.True(t, ok)
	assert.Greater(t, len(result.Tools), 0)

	toolNames := make(map[string]bool)
	for _, tool := range result.Tools {
		toolNames[tool.Name] = true
		assert.NotEmpty(t, tool.Name)
		assert.NotEmpty(t, tool.InputSchema)
	}

	expectedTools := []string{
		"model_list",
		"model_get",
		"model_create",
		"device_detect",
		"engine_list",
		"inference_chat",
	}

	for _, expected := range expectedTools {
		assert.True(t, toolNames[expected], "tool %s should be in list", expected)
	}
}

func TestMCPServerIntegrationToolsCall(t *testing.T) {
	adapter := createTestMCPAdapter(t)

	adapter.HandleRequest(context.Background(), &gateway.MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  json.RawMessage(`{}`),
	})

	t.Run("call model_list", func(t *testing.T) {
		resp := adapter.HandleRequest(context.Background(), &gateway.MCPRequest{
			JSONRPC: "2.0",
			ID:      5,
			Method:  "tools/call",
			Params:  json.RawMessage(`{"name": "model_list", "arguments": {}}`),
		})

		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)
		assert.NotNil(t, resp.Result)

		result, ok := resp.Result.(*gateway.MCPToolResult)
		require.True(t, ok)
		assert.False(t, result.IsError)
		assert.Greater(t, len(result.Content), 0)
		assert.Equal(t, "text", result.Content[0].Type)
	})

	t.Run("call engine_list", func(t *testing.T) {
		resp := adapter.HandleRequest(context.Background(), &gateway.MCPRequest{
			JSONRPC: "2.0",
			ID:      6,
			Method:  "tools/call",
			Params:  json.RawMessage(`{"name": "engine_list", "arguments": {}}`),
		})

		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)

		result, ok := resp.Result.(*gateway.MCPToolResult)
		require.True(t, ok)
		assert.False(t, result.IsError)
	})

	t.Run("call service_list", func(t *testing.T) {
		resp := adapter.HandleRequest(context.Background(), &gateway.MCPRequest{
			JSONRPC: "2.0",
			ID:      7,
			Method:  "tools/call",
			Params:  json.RawMessage(`{"name": "service_list", "arguments": {}}`),
		})

		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)

		result, ok := resp.Result.(*gateway.MCPToolResult)
		require.True(t, ok)
		assert.False(t, result.IsError)
	})

	t.Run("call app_list", func(t *testing.T) {
		resp := adapter.HandleRequest(context.Background(), &gateway.MCPRequest{
			JSONRPC: "2.0",
			ID:      8,
			Method:  "tools/call",
			Params:  json.RawMessage(`{"name": "app_list", "arguments": {}}`),
		})

		require.NotNil(t, resp)
		assert.Nil(t, resp.Error)

		result, ok := resp.Result.(*gateway.MCPToolResult)
		require.True(t, ok)
		assert.False(t, result.IsError)
	})
}

func TestMCPServerIntegrationResourcesList(t *testing.T) {
	adapter := createTestMCPAdapter(t)

	adapter.HandleRequest(context.Background(), &gateway.MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  json.RawMessage(`{}`),
	})

	resp := adapter.HandleRequest(context.Background(), &gateway.MCPRequest{
		JSONRPC: "2.0",
		ID:      9,
		Method:  "resources/list",
		Params:  json.RawMessage(`{}`),
	})

	require.NotNil(t, resp)
}

func TestMCPServerIntegrationPromptsList(t *testing.T) {
	adapter := createTestMCPAdapter(t)

	adapter.HandleRequest(context.Background(), &gateway.MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  json.RawMessage(`{}`),
	})

	resp := adapter.HandleRequest(context.Background(), &gateway.MCPRequest{
		JSONRPC: "2.0",
		ID:      10,
		Method:  "prompts/list",
		Params:  json.RawMessage(`{}`),
	})

	require.NotNil(t, resp)
}

func TestMCPServerIntegrationErrors(t *testing.T) {
	adapter := createTestMCPAdapter(t)

	adapter.HandleRequest(context.Background(), &gateway.MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  json.RawMessage(`{}`),
	})

	t.Run("invalid JSON-RPC version", func(t *testing.T) {
		resp := adapter.HandleRequest(context.Background(), &gateway.MCPRequest{
			JSONRPC: "1.0",
			ID:      100,
			Method:  "initialize",
			Params:  json.RawMessage(`{}`),
		})

		require.NotNil(t, resp)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, gateway.MCPErrorCodeInvalidRequest, resp.Error.Code)
	})

	t.Run("empty method", func(t *testing.T) {
		resp := adapter.HandleRequest(context.Background(), &gateway.MCPRequest{
			JSONRPC: "2.0",
			ID:      101,
			Method:  "",
			Params:  json.RawMessage(`{}`),
		})

		require.NotNil(t, resp)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, gateway.MCPErrorCodeInvalidRequest, resp.Error.Code)
	})

	t.Run("method not found", func(t *testing.T) {
		resp := adapter.HandleRequest(context.Background(), &gateway.MCPRequest{
			JSONRPC: "2.0",
			ID:      102,
			Method:  "unknown/method",
			Params:  json.RawMessage(`{}`),
		})

		require.NotNil(t, resp)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, gateway.MCPErrorCodeMethodNotFound, resp.Error.Code)
	})

	t.Run("tool not found", func(t *testing.T) {
		resp := adapter.HandleRequest(context.Background(), &gateway.MCPRequest{
			JSONRPC: "2.0",
			ID:      103,
			Method:  "tools/call",
			Params:  json.RawMessage(`{"name": "nonexistent_tool", "arguments": {}}`),
		})

		require.NotNil(t, resp)
		assert.NotNil(t, resp.Error)
	})

	t.Run("invalid tool call params", func(t *testing.T) {
		resp := adapter.HandleRequest(context.Background(), &gateway.MCPRequest{
			JSONRPC: "2.0",
			ID:      104,
			Method:  "tools/call",
			Params:  json.RawMessage(`invalid json`),
		})

		require.NotNil(t, resp)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, gateway.MCPErrorCodeInvalidParams, resp.Error.Code)
	})

	t.Run("missing tool name", func(t *testing.T) {
		resp := adapter.HandleRequest(context.Background(), &gateway.MCPRequest{
			JSONRPC: "2.0",
			ID:      105,
			Method:  "tools/call",
			Params:  json.RawMessage(`{"arguments": {}}`),
		})

		require.NotNil(t, resp)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, gateway.MCPErrorCodeInvalidParams, resp.Error.Code)
	})
}

func TestMCPServerIntegrationShutdown(t *testing.T) {
	adapter := createTestMCPAdapter(t)

	adapter.HandleRequest(context.Background(), &gateway.MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  json.RawMessage(`{}`),
	})

	resp := adapter.HandleRequest(context.Background(), &gateway.MCPRequest{
		JSONRPC: "2.0",
		ID:      200,
		Method:  "shutdown",
		Params:  json.RawMessage(`{}`),
	})

	require.NotNil(t, resp)
	assert.Nil(t, resp.Error)
}

func TestMCPServerIntegrationToolSchema(t *testing.T) {
	adapter := createTestMCPAdapter(t)

	adapter.HandleRequest(context.Background(), &gateway.MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  json.RawMessage(`{}`),
	})

	resp := adapter.HandleRequest(context.Background(), &gateway.MCPRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
		Params:  json.RawMessage(`{}`),
	})

	require.NotNil(t, resp)
	assert.Nil(t, resp.Error)

	result, ok := resp.Result.(*gateway.MCPToolsListResult)
	require.True(t, ok)

	for _, tool := range result.Tools {
		assert.NotEmpty(t, tool.Name)
		assert.NotNil(t, tool.InputSchema)

		schema := tool.InputSchema
		assert.Contains(t, schema, "type")
		assert.NotEmpty(t, schema["type"])
	}
}

func TestMCPServerIntegrationAllDomains(t *testing.T) {
	adapter := createTestMCPAdapter(t)

	adapter.HandleRequest(context.Background(), &gateway.MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  json.RawMessage(`{}`),
	})

	domainTools := []struct {
		name string
		args map[string]any
	}{
		{"model_list", map[string]any{}},
		{"device_detect", map[string]any{}},
		{"engine_list", map[string]any{}},
		{"inference_models", map[string]any{}},
		{"resource_status", map[string]any{}},
		{"service_list", map[string]any{}},
		{"app_list", map[string]any{}},
		{"pipeline_list", map[string]any{}},
		{"alert_list_rules", map[string]any{}},
		{"remote_status", map[string]any{}},
	}

	for i, tc := range domainTools {
		t.Run(tc.name, func(t *testing.T) {
			argsBytes, _ := json.Marshal(tc.args)
			params := map[string]any{
				"name":      tc.name,
				"arguments": tc.args,
			}
			paramsBytes, _ := json.Marshal(params)

			resp := adapter.HandleRequest(context.Background(), &gateway.MCPRequest{
				JSONRPC: "2.0",
				ID:      i + 100,
				Method:  "tools/call",
				Params:  paramsBytes,
			})

			require.NotNil(t, resp)
			if resp.Error != nil {
				t.Logf("Tool %s error: %s", tc.name, resp.Error.Message)
			}

			_ = argsBytes
		})
	}
}
