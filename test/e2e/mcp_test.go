package e2e

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/registry"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupMCPAdapter creates a fully-wired MCPAdapter backed by a real registry with
// mock providers, matching the pattern used by the rest of the e2e suite.
func setupMCPAdapter(t *testing.T) *gateway.MCPAdapter {
	t.Helper()
	reg := unit.NewRegistry()
	opts := []registry.Option{
		registry.WithEngineProvider(newMockEngineProvider()),
		registry.WithServiceProvider(newMockServiceProvider()),
		registry.WithInferenceProvider(newMockInferenceProvider()),
	}
	require.NoError(t, registry.RegisterAll(reg, opts...))
	gw := gateway.NewGateway(reg)
	return gateway.NewMCPAdapter(gw)
}

// mcpCall is a helper that sends a single JSON-RPC request to the adapter and
// returns the parsed response.
func mcpCall(t *testing.T, adapter *gateway.MCPAdapter, id any, method string, params any) *gateway.MCPResponse {
	t.Helper()
	var raw json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		require.NoError(t, err)
		raw = b
	}
	return adapter.HandleRequest(context.Background(), &gateway.MCPRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  raw,
	})
}

// initializeAdapter performs the MCP initialize handshake and asserts success.
func initializeAdapter(t *testing.T, adapter *gateway.MCPAdapter) {
	t.Helper()
	resp := mcpCall(t, adapter, 0, "initialize", map[string]any{
		"protocolVersion": gateway.MCPVersion,
		"clientInfo":      map[string]any{"name": "e2e-test-client", "version": "1.0.0"},
		"capabilities":    map[string]any{},
	})
	require.NotNil(t, resp)
	require.Nil(t, resp.Error, "initialize must succeed: %v", resp.Error)
}

// ─── 1. Initialize handshake ─────────────────────────────────────────────────

func TestMCPE2E_InitializeHandshake(t *testing.T) {
	adapter := setupMCPAdapter(t)

	params := map[string]any{
		"protocolVersion": gateway.MCPVersion,
		"clientInfo": map[string]any{
			"name":    "e2e-test-client",
			"version": "1.0.0",
		},
		"capabilities": map[string]any{},
	}
	resp := mcpCall(t, adapter, 1, "initialize", params)

	require.NotNil(t, resp)
	assert.Nil(t, resp.Error, "unexpected error: %v", resp.Error)
	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.Equal(t, 1, resp.ID)
	require.NotNil(t, resp.Result)

	result, ok := resp.Result.(*gateway.MCPInitializeResult)
	require.True(t, ok, "result should be *MCPInitializeResult, got %T", resp.Result)

	assert.Equal(t, gateway.MCPVersion, result.ProtocolVersion)
	assert.Equal(t, gateway.MCPServerName, result.ServerInfo.Name)
	assert.NotEmpty(t, result.ServerInfo.Version)
	assert.NotNil(t, result.Capabilities.Tools)
	assert.NotNil(t, result.Capabilities.Resources)
	assert.NotNil(t, result.Capabilities.Prompts)
	assert.NotEmpty(t, result.Instructions)
}

func TestMCPE2E_InitializeHandshake_NotificationNoResponse(t *testing.T) {
	adapter := setupMCPAdapter(t)
	initializeAdapter(t, adapter)

	// notifications/initialized must return nil (no response sent over wire)
	resp := adapter.HandleRequest(context.Background(), &gateway.MCPRequest{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
	})
	assert.Nil(t, resp, "notification must not generate a response")
	assert.True(t, adapter.IsInitialized())
}

func TestMCPE2E_InitializeHandshake_PingAfterInit(t *testing.T) {
	adapter := setupMCPAdapter(t)
	initializeAdapter(t, adapter)

	resp := mcpCall(t, adapter, 2, "ping", map[string]any{})
	require.NotNil(t, resp)
	assert.Nil(t, resp.Error)
	assert.Equal(t, "2.0", resp.JSONRPC)
	assert.Equal(t, 2, resp.ID)
}

// ─── 2. tools/list – schema correctness ───────────────────────────────────────

func TestMCPE2E_ToolsList_ReturnsAllTools(t *testing.T) {
	adapter := setupMCPAdapter(t)
	initializeAdapter(t, adapter)

	resp := mcpCall(t, adapter, 10, "tools/list", map[string]any{})
	require.NotNil(t, resp)
	assert.Nil(t, resp.Error, "unexpected error: %v", resp.Error)

	result, ok := resp.Result.(*gateway.MCPToolsListResult)
	require.True(t, ok, "expected *MCPToolsListResult, got %T", resp.Result)
	assert.Greater(t, len(result.Tools), 0, "tools list must be non-empty")
}

func TestMCPE2E_ToolsList_SchemaIsValid(t *testing.T) {
	adapter := setupMCPAdapter(t)
	initializeAdapter(t, adapter)

	resp := mcpCall(t, adapter, 11, "tools/list", map[string]any{})
	require.NotNil(t, resp)
	require.Nil(t, resp.Error)

	result, ok := resp.Result.(*gateway.MCPToolsListResult)
	require.True(t, ok)

	for _, tool := range result.Tools {
		assert.NotEmpty(t, tool.Name, "tool name must not be empty")
		assert.NotNil(t, tool.InputSchema, "tool %q must have an inputSchema", tool.Name)
		assert.Equal(t, "object", tool.InputSchema["type"],
			"tool %q inputSchema.type must be 'object'", tool.Name)
		_, hasProps := tool.InputSchema["properties"]
		assert.True(t, hasProps, "tool %q inputSchema must have 'properties'", tool.Name)
	}
}

func TestMCPE2E_ToolsList_CoreToolsPresent(t *testing.T) {
	adapter := setupMCPAdapter(t)
	initializeAdapter(t, adapter)

	resp := mcpCall(t, adapter, 12, "tools/list", map[string]any{})
	require.NotNil(t, resp)
	require.Nil(t, resp.Error)

	result, ok := resp.Result.(*gateway.MCPToolsListResult)
	require.True(t, ok)

	index := make(map[string]gateway.MCPToolDefinition, len(result.Tools))
	for _, tool := range result.Tools {
		index[tool.Name] = tool
	}

	required := []string{
		"model_list",
		"model_get",
		"model_create",
		"device_detect",
		"engine_list",
		"inference_chat",
	}
	for _, name := range required {
		_, found := index[name]
		assert.True(t, found, "expected tool %q to be in tools/list", name)
	}
}

// ─── 3. Tool execution round-trip ─────────────────────────────────────────────

func TestMCPE2E_ToolCall_ModelList(t *testing.T) {
	adapter := setupMCPAdapter(t)
	initializeAdapter(t, adapter)

	resp := mcpCall(t, adapter, 20, "tools/call", map[string]any{
		"name":      "model_list",
		"arguments": map[string]any{},
	})
	require.NotNil(t, resp)
	assert.Nil(t, resp.Error, "unexpected error: %v", resp.Error)

	result, ok := resp.Result.(*gateway.MCPToolResult)
	require.True(t, ok, "expected *MCPToolResult, got %T", resp.Result)
	assert.False(t, result.IsError, "IsError must be false on success")
	require.Greater(t, len(result.Content), 0, "content must not be empty")
	assert.Equal(t, "text", result.Content[0].Type)
	assert.NotEmpty(t, result.Content[0].Text, "content text must not be empty")

	// Verify content is valid JSON
	var data map[string]any
	assert.NoError(t, json.Unmarshal([]byte(result.Content[0].Text), &data),
		"tool result text must be valid JSON")
}

func TestMCPE2E_ToolCall_EngineList(t *testing.T) {
	adapter := setupMCPAdapter(t)
	initializeAdapter(t, adapter)

	resp := mcpCall(t, adapter, 21, "tools/call", map[string]any{
		"name":      "engine_list",
		"arguments": map[string]any{},
	})
	require.NotNil(t, resp)
	assert.Nil(t, resp.Error)

	result, ok := resp.Result.(*gateway.MCPToolResult)
	require.True(t, ok)
	assert.False(t, result.IsError)
	require.Greater(t, len(result.Content), 0)
	assert.Equal(t, "text", result.Content[0].Type)
}

func TestMCPE2E_ToolCall_DeviceDetect(t *testing.T) {
	adapter := setupMCPAdapter(t)
	initializeAdapter(t, adapter)

	resp := mcpCall(t, adapter, 22, "tools/call", map[string]any{
		"name":      "device_detect",
		"arguments": map[string]any{},
	})
	require.NotNil(t, resp)
	// device_detect may return IsError=true in non-GPU environments; we only
	// verify the round-trip completes with a well-formed MCPToolResult.
	assert.Nil(t, resp.Error, "tools/call protocol error must be nil")

	result, ok := resp.Result.(*gateway.MCPToolResult)
	require.True(t, ok, "result must be *MCPToolResult")
	require.Greater(t, len(result.Content), 0, "content must not be empty")
	assert.Equal(t, "text", result.Content[0].Type)
}

func TestMCPE2E_ToolCall_InferenceChat(t *testing.T) {
	adapter := setupMCPAdapter(t)
	initializeAdapter(t, adapter)

	resp := mcpCall(t, adapter, 23, "tools/call", map[string]any{
		"name": "inference_chat",
		"arguments": map[string]any{
			"model": "llama3",
			"messages": []any{
				map[string]any{"role": "user", "content": "Hello!"},
			},
		},
	})
	require.NotNil(t, resp)
	assert.Nil(t, resp.Error)

	result, ok := resp.Result.(*gateway.MCPToolResult)
	require.True(t, ok)
	assert.False(t, result.IsError)
	require.Greater(t, len(result.Content), 0)
	assert.Equal(t, "text", result.Content[0].Type)
}

func TestMCPE2E_ToolCall_ServiceList(t *testing.T) {
	adapter := setupMCPAdapter(t)
	initializeAdapter(t, adapter)

	resp := mcpCall(t, adapter, 24, "tools/call", map[string]any{
		"name":      "service_list",
		"arguments": map[string]any{},
	})
	require.NotNil(t, resp)
	assert.Nil(t, resp.Error)

	result, ok := resp.Result.(*gateway.MCPToolResult)
	require.True(t, ok)
	assert.False(t, result.IsError)
}

// ─── 4. Error handling ────────────────────────────────────────────────────────

func TestMCPE2E_Error_InvalidJSONRPCVersion(t *testing.T) {
	adapter := setupMCPAdapter(t)

	resp := adapter.HandleRequest(context.Background(), &gateway.MCPRequest{
		JSONRPC: "1.0",
		ID:      100,
		Method:  "ping",
	})
	require.NotNil(t, resp)
	require.NotNil(t, resp.Error)
	assert.Equal(t, gateway.MCPErrorCodeInvalidRequest, resp.Error.Code)
}

func TestMCPE2E_Error_EmptyMethod(t *testing.T) {
	adapter := setupMCPAdapter(t)

	resp := adapter.HandleRequest(context.Background(), &gateway.MCPRequest{
		JSONRPC: "2.0",
		ID:      101,
		Method:  "",
	})
	require.NotNil(t, resp)
	require.NotNil(t, resp.Error)
	assert.Equal(t, gateway.MCPErrorCodeInvalidRequest, resp.Error.Code)
}

func TestMCPE2E_Error_MethodNotFound(t *testing.T) {
	adapter := setupMCPAdapter(t)
	initializeAdapter(t, adapter)

	resp := mcpCall(t, adapter, 102, "unknown/method", map[string]any{})
	require.NotNil(t, resp)
	require.NotNil(t, resp.Error)
	assert.Equal(t, gateway.MCPErrorCodeMethodNotFound, resp.Error.Code)
	assert.Contains(t, resp.Error.Message, "method not found")
}

func TestMCPE2E_Error_ToolNotFound(t *testing.T) {
	adapter := setupMCPAdapter(t)
	initializeAdapter(t, adapter)

	resp := mcpCall(t, adapter, 103, "tools/call", map[string]any{
		"name":      "nonexistent_tool_xyz",
		"arguments": map[string]any{},
	})
	require.NotNil(t, resp)
	// The implementation returns an error response when the tool is not found
	require.NotNil(t, resp.Error)
}

func TestMCPE2E_Error_MalformedToolCallParams(t *testing.T) {
	adapter := setupMCPAdapter(t)
	initializeAdapter(t, adapter)

	// Pass deliberately invalid JSON as params
	resp := adapter.HandleRequest(context.Background(), &gateway.MCPRequest{
		JSONRPC: "2.0",
		ID:      104,
		Method:  "tools/call",
		Params:  json.RawMessage(`{invalid json`),
	})
	require.NotNil(t, resp)
	require.NotNil(t, resp.Error)
	assert.Equal(t, gateway.MCPErrorCodeInvalidParams, resp.Error.Code)
}

func TestMCPE2E_Error_MissingToolName(t *testing.T) {
	adapter := setupMCPAdapter(t)
	initializeAdapter(t, adapter)

	resp := mcpCall(t, adapter, 105, "tools/call", map[string]any{
		"arguments": map[string]any{},
	})
	require.NotNil(t, resp)
	require.NotNil(t, resp.Error)
	assert.Equal(t, gateway.MCPErrorCodeInvalidParams, resp.Error.Code)
}

func TestMCPE2E_Error_MalformedInitializeParams(t *testing.T) {
	adapter := setupMCPAdapter(t)

	resp := adapter.HandleRequest(context.Background(), &gateway.MCPRequest{
		JSONRPC: "2.0",
		ID:      106,
		Method:  "initialize",
		Params:  json.RawMessage(`{bad json`),
	})
	require.NotNil(t, resp)
	require.NotNil(t, resp.Error)
	assert.Equal(t, gateway.MCPErrorCodeInvalidParams, resp.Error.Code)
}

func TestMCPE2E_Error_ResourceNotFound(t *testing.T) {
	adapter := setupMCPAdapter(t)
	initializeAdapter(t, adapter)

	resp := mcpCall(t, adapter, 107, "resources/read", map[string]any{
		"uri": "asms://nonexistent/resource/path",
	})
	require.NotNil(t, resp)
	require.NotNil(t, resp.Error)
	assert.Equal(t, gateway.MCPErrorCodeResourceNotFound, resp.Error.Code)
}

func TestMCPE2E_Error_ResourceReadMissingURI(t *testing.T) {
	adapter := setupMCPAdapter(t)
	initializeAdapter(t, adapter)

	resp := mcpCall(t, adapter, 108, "resources/read", map[string]any{})
	require.NotNil(t, resp)
	require.NotNil(t, resp.Error)
	assert.Equal(t, gateway.MCPErrorCodeInvalidParams, resp.Error.Code)
}

// ─── 5a. Stdio server session: request/response round-trip ───────────────────

// TestMCPE2E_StdioServer_InitializeAndToolsList exercises the full stdio
// transport: JSON-RPC messages are written to a pipe, MCPServer reads them,
// dispatches through the adapter, and writes responses back to a buffer.
func TestMCPE2E_StdioServer_InitializeAndToolsList(t *testing.T) {
	adapter := setupMCPAdapter(t)

	pr, pw := io.Pipe()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	server := gateway.NewMCPServer(adapter, pr, stdout, stderr)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- server.Serve(ctx)
	}()

	// Send initialize
	sendStdioRequest(t, pw, &gateway.MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  json.RawMessage(`{"protocolVersion":"2024-11-05","clientInfo":{"name":"stdio-e2e","version":"1.0"},"capabilities":{}}`),
	})

	// Send tools/list
	sendStdioRequest(t, pw, &gateway.MCPRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
		Params:  json.RawMessage(`{}`),
	})

	// Close stdin to signal EOF and let the server stop
	pw.Close()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("MCPServer did not stop after stdin EOF within 3 s")
	}

	// Wait for all handler goroutines to finish writing responses before reading
	// stdout. MCPServer.Shutdown calls wg.Wait() which synchronises the goroutines.
	server.Shutdown()

	// Parse all newline-delimited JSON responses from stdout
	responses := parseStdioResponses(t, stdout.String())
	require.GreaterOrEqual(t, len(responses), 2, "expected at least 2 responses (initialize + tools/list)")

	// First response: initialize
	initResp := findResponseByID(responses, float64(1))
	require.NotNil(t, initResp, "initialize response not found")
	assert.Nil(t, initResp["error"], "initialize must succeed")
	resultMap, ok := initResp["result"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, gateway.MCPVersion, resultMap["protocolVersion"])

	// Second response: tools/list
	listResp := findResponseByID(responses, float64(2))
	require.NotNil(t, listResp, "tools/list response not found")
	assert.Nil(t, listResp["error"], "tools/list must succeed")
	listResult, ok := listResp["result"].(map[string]any)
	require.True(t, ok)
	tools, ok := listResult["tools"].([]any)
	require.True(t, ok)
	assert.Greater(t, len(tools), 0, "tools list must be non-empty")
}

func TestMCPE2E_StdioServer_InvalidJSONProducesParseError(t *testing.T) {
	adapter := setupMCPAdapter(t)

	pr, pw := io.Pipe()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	server := gateway.NewMCPServer(adapter, pr, stdout, stderr)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- server.Serve(ctx) }()

	// Write a non-JSON line
	fmt.Fprintln(pw, `this is not json`)
	pw.Close()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("server did not stop")
	}
	server.Shutdown()

	responses := parseStdioResponses(t, stdout.String())
	require.GreaterOrEqual(t, len(responses), 1)
	first := responses[0]
	errField, ok := first["error"].(map[string]any)
	require.True(t, ok, "expected an error field in parse-error response")
	code, _ := errField["code"].(float64)
	assert.Equal(t, float64(gateway.MCPErrorCodeParseError), code)
}

func TestMCPE2E_StdioServer_ToolCallRoundTrip(t *testing.T) {
	adapter := setupMCPAdapter(t)

	pr, pw := io.Pipe()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	server := gateway.NewMCPServer(adapter, pr, stdout, stderr)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- server.Serve(ctx) }()

	sendStdioRequest(t, pw, &gateway.MCPRequest{
		JSONRPC: "2.0", ID: 1, Method: "initialize",
		Params: json.RawMessage(`{}`),
	})
	sendStdioRequest(t, pw, &gateway.MCPRequest{
		JSONRPC: "2.0", ID: 2, Method: "tools/call",
		Params: json.RawMessage(`{"name":"model_list","arguments":{}}`),
	})
	pw.Close()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("server did not stop")
	}
	server.Shutdown()

	responses := parseStdioResponses(t, stdout.String())
	require.GreaterOrEqual(t, len(responses), 2)

	toolResp := findResponseByID(responses, float64(2))
	require.NotNil(t, toolResp, "tools/call response not found")
	assert.Nil(t, toolResp["error"], "tools/call must succeed")

	resultMap, ok := toolResp["result"].(map[string]any)
	require.True(t, ok)
	content, ok := resultMap["content"].([]any)
	require.True(t, ok)
	assert.Greater(t, len(content), 0)
	firstBlock, ok := content[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "text", firstBlock["type"])
	assert.NotEmpty(t, firstBlock["text"])
}

// ─── 5b. SSE server session lifecycle ─────────────────────────────────────────

// startSSEServer starts a real MCPSSEServer on a random OS-assigned port and
// returns its base URL. The server is shut down when the test finishes.
func startSSEServer(t *testing.T, adapter *gateway.MCPAdapter) string {
	t.Helper()

	// Pick a free port by binding then closing a listener.
	// There is an inherent TOCTOU race here (the OS may reassign the port),
	// but it is acceptable in practice for local test environments.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	ln.Close()

	sseServer := gateway.NewMCPSSEServer(adapter, addr)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		shutCtx, shutCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer shutCancel()
		sseServer.Shutdown(shutCtx) //nolint:errcheck
	})

	go func() {
		sseServer.Serve(ctx) //nolint:errcheck
	}()

	// Wait until the server is accepting connections (up to 2 seconds).
	base := "http://" + addr
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(base + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return base
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("SSE server at %s did not become ready within 2 s", base)
	return ""
}

// TestMCPE2E_SSEServer_HealthEndpoint verifies the /health endpoint of the SSE
// server works correctly.
func TestMCPE2E_SSEServer_HealthEndpoint(t *testing.T) {
	adapter := setupMCPAdapter(t)
	base := startSSEServer(t, adapter)

	resp, err := http.Get(base + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "healthy", body["status"])
	assert.NotEmpty(t, body["timestamp"])
}

// TestMCPE2E_SSEServer_MessageEndpointErrors checks the /message endpoint
// rejects bad requests before a session exists.
func TestMCPE2E_SSEServer_MessageEndpointErrors(t *testing.T) {
	adapter := setupMCPAdapter(t)
	base := startSSEServer(t, adapter)

	t.Run("method not allowed on /message GET", func(t *testing.T) {
		resp, err := http.Get(base + "/message")
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})

	t.Run("missing session param", func(t *testing.T) {
		resp, err := http.Post(base+"/message", "application/json", strings.NewReader("{}"))
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("session not found", func(t *testing.T) {
		resp, err := http.Post(base+"/message?session=does-not-exist",
			"application/json", strings.NewReader("{}"))
		require.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("ghost session invalid JSON", func(t *testing.T) {
		resp, err := http.Post(base+"/message?session=ghost",
			"application/json", strings.NewReader("{bad json}"))
		require.NoError(t, err)
		resp.Body.Close()
		// Session not found takes priority over JSON parse error
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

// TestMCPE2E_SSEServer_MethodNotAllowedOnSSE verifies that POST to /sse is rejected.
func TestMCPE2E_SSEServer_MethodNotAllowedOnSSE(t *testing.T) {
	adapter := setupMCPAdapter(t)
	base := startSSEServer(t, adapter)

	resp, err := http.Post(base+"/sse", "application/json", nil)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}

// TestMCPE2E_SSEServer_FullSessionLifecycle establishes an SSE session, sends a
// tools/call request over /message and reads the async response from the event
// stream.
func TestMCPE2E_SSEServer_FullSessionLifecycle(t *testing.T) {
	adapter := setupMCPAdapter(t)
	base := startSSEServer(t, adapter)

	// Open SSE stream (streaming, so we use a client with no timeout on read)
	sseResp, err := http.Get(base + "/sse")
	require.NoError(t, err)
	defer sseResp.Body.Close()

	assert.Equal(t, http.StatusOK, sseResp.StatusCode)
	assert.Equal(t, "text/event-stream", sseResp.Header.Get("Content-Type"))

	// Use a single shared scanner for the entire SSE stream so buffered data
	// is preserved between successive reads.
	sseScanner := bufio.NewScanner(sseResp.Body)

	// Read the endpoint event which contains the session ID
	sessionURL := readSSEEndpointEventFromScanner(t, sseScanner, base)
	require.NotEmpty(t, sessionURL, "expected session URL from SSE endpoint event")

	// Initialize over /message
	initReq := &gateway.MCPRequest{
		JSONRPC: "2.0", ID: 1, Method: "initialize",
		Params: json.RawMessage(`{"protocolVersion":"2024-11-05","clientInfo":{"name":"sse-e2e","version":"1.0"},"capabilities":{}}`),
	}
	postMCPRequest(t, sessionURL, initReq, http.StatusAccepted)

	// Read the initialize response from the SSE stream
	initEvent := readSSEMessageEventFromScanner(t, sseScanner)
	var initEventResp map[string]any
	require.NoError(t, json.Unmarshal([]byte(initEvent), &initEventResp))
	assert.Nil(t, initEventResp["error"], "initialize must succeed over SSE")
	assert.NotNil(t, initEventResp["result"])

	// tools/call model_list
	listReq := &gateway.MCPRequest{
		JSONRPC: "2.0", ID: 2, Method: "tools/call",
		Params: json.RawMessage(`{"name":"model_list","arguments":{}}`),
	}
	postMCPRequest(t, sessionURL, listReq, http.StatusAccepted)

	// Read the tools/call response from the SSE stream
	listEvent := readSSEMessageEventFromScanner(t, sseScanner)
	var listEventResp map[string]any
	require.NoError(t, json.Unmarshal([]byte(listEvent), &listEventResp))
	assert.Nil(t, listEventResp["error"], "tools/call must succeed over SSE")

	resultMap, ok := listEventResp["result"].(map[string]any)
	require.True(t, ok)
	content, ok := resultMap["content"].([]any)
	require.True(t, ok)
	assert.Greater(t, len(content), 0)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func sendStdioRequest(t *testing.T, w io.Writer, req *gateway.MCPRequest) {
	t.Helper()
	data, err := json.Marshal(req)
	require.NoError(t, err)
	data = append(data, '\n')
	_, err = w.Write(data)
	require.NoError(t, err)
}

func parseStdioResponses(t *testing.T, output string) []map[string]any {
	t.Helper()
	var results []map[string]any
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Logf("skipping non-JSON line: %s", line)
			continue
		}
		results = append(results, m)
	}
	return results
}

func findResponseByID(responses []map[string]any, id float64) map[string]any {
	for _, r := range responses {
		if rid, ok := r["id"].(float64); ok && rid == id {
			return r
		}
	}
	return nil
}

// readSSEEndpointEventFromScanner reads lines from a shared SSE scanner until
// it finds the "endpoint" event and returns the full message URL (base + path).
func readSSEEndpointEventFromScanner(t *testing.T, scanner *bufio.Scanner, base string) string {
	t.Helper()
	var eventType string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") && eventType == "endpoint" {
			path := strings.TrimPrefix(line, "data: ")
			return base + path
		}
	}
	return ""
}

// readSSEMessageEventFromScanner reads lines from a shared SSE scanner until
// it finds the next "message" event and returns the raw JSON data payload.
func readSSEMessageEventFromScanner(t *testing.T, scanner *bufio.Scanner) string {
	t.Helper()
	var eventType string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") && eventType == "message" {
			return strings.TrimPrefix(line, "data: ")
		}
	}
	t.Fatal("no SSE message event received")
	return ""
}

// postMCPRequest POSTs a JSON-encoded MCPRequest to sessionURL and asserts the
// expected HTTP status code.
func postMCPRequest(t *testing.T, url string, req *gateway.MCPRequest, expectedStatus int) {
	t.Helper()
	data, err := json.Marshal(req)
	require.NoError(t, err)

	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, expectedStatus, resp.StatusCode)
}
