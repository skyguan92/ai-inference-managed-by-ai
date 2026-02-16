package gateway

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestNewMCPAdapter(t *testing.T) {
	t.Run("with gateway", func(t *testing.T) {
		g := NewGateway(nil)
		adapter := NewMCPAdapter(g)
		if adapter == nil {
			t.Fatal("expected adapter, got nil")
		}
		if adapter.gateway != g {
			t.Error("expected same gateway instance")
		}
	})

	t.Run("with nil gateway", func(t *testing.T) {
		adapter := NewMCPAdapter(nil)
		if adapter == nil {
			t.Fatal("expected adapter, got nil")
		}
		if adapter.gateway != nil {
			t.Error("expected nil gateway")
		}
	})
}

func TestMCPAdapter_HandleRequest_Initialize(t *testing.T) {
	g := NewGateway(nil)
	adapter := NewMCPAdapter(g)

	t.Run("successful initialize", func(t *testing.T) {
		req := &MCPRequest{
			JSONRPC: JSONRPC,
			ID:      1,
			Method:  "initialize",
			Params:  json.RawMessage(`{"protocolVersion":"2024-11-05","clientInfo":{"name":"test-client","version":"1.0"},"capabilities":{}}`),
		}

		resp := adapter.HandleRequest(context.Background(), req)
		if resp == nil {
			t.Fatal("expected response, got nil")
		}

		if resp.Error != nil {
			t.Fatalf("unexpected error: %v", resp.Error)
		}

		result, ok := resp.Result.(*MCPInitializeResult)
		if !ok {
			t.Fatalf("expected MCPInitializeResult, got %T", resp.Result)
		}

		if result.ProtocolVersion != MCPVersion {
			t.Errorf("expected protocol version %s, got %s", MCPVersion, result.ProtocolVersion)
		}

		if result.ServerInfo.Name != MCPServerName {
			t.Errorf("expected server name %s, got %s", MCPServerName, result.ServerInfo.Name)
		}

		if !adapter.IsInitialized() {
			t.Error("expected adapter to be initialized")
		}
	})

	t.Run("invalid params", func(t *testing.T) {
		req := &MCPRequest{
			JSONRPC: JSONRPC,
			ID:      2,
			Method:  "initialize",
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

	t.Run("empty params", func(t *testing.T) {
		adapter2 := NewMCPAdapter(g)
		req := &MCPRequest{
			JSONRPC: JSONRPC,
			ID:      3,
			Method:  "initialize",
			Params:  nil,
		}

		resp := adapter2.HandleRequest(context.Background(), req)
		if resp == nil {
			t.Fatal("expected response, got nil")
		}

		if resp.Error != nil {
			t.Fatalf("unexpected error: %v", resp.Error)
		}

		result, ok := resp.Result.(*MCPInitializeResult)
		if !ok {
			t.Fatalf("expected MCPInitializeResult, got %T", resp.Result)
		}

		if result.ServerInfo.Name != MCPServerName {
			t.Errorf("expected server name %s, got %s", MCPServerName, result.ServerInfo.Name)
		}
	})
}

func TestMCPAdapter_HandleRequest_InvalidJSONRPC(t *testing.T) {
	g := NewGateway(nil)
	adapter := NewMCPAdapter(g)

	req := &MCPRequest{
		JSONRPC: "1.0",
		ID:      1,
		Method:  "initialize",
	}

	resp := adapter.HandleRequest(context.Background(), req)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	if resp.Error == nil {
		t.Fatal("expected error")
	}

	if resp.Error.Code != MCPErrorCodeInvalidRequest {
		t.Errorf("expected error code %d, got %d", MCPErrorCodeInvalidRequest, resp.Error.Code)
	}
}

func TestMCPAdapter_HandleRequest_EmptyMethod(t *testing.T) {
	g := NewGateway(nil)
	adapter := NewMCPAdapter(g)

	req := &MCPRequest{
		JSONRPC: JSONRPC,
		ID:      1,
		Method:  "",
	}

	resp := adapter.HandleRequest(context.Background(), req)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	if resp.Error == nil {
		t.Fatal("expected error")
	}

	if resp.Error.Code != MCPErrorCodeInvalidRequest {
		t.Errorf("expected error code %d, got %d", MCPErrorCodeInvalidRequest, resp.Error.Code)
	}
}

func TestMCPAdapter_HandleRequest_MethodNotFound(t *testing.T) {
	g := NewGateway(nil)
	adapter := NewMCPAdapter(g)

	req := &MCPRequest{
		JSONRPC: JSONRPC,
		ID:      1,
		Method:  "unknown/method",
	}

	resp := adapter.HandleRequest(context.Background(), req)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	if resp.Error == nil {
		t.Fatal("expected error")
	}

	if resp.Error.Code != MCPErrorCodeMethodNotFound {
		t.Errorf("expected error code %d, got %d", MCPErrorCodeMethodNotFound, resp.Error.Code)
	}
}

func TestMCPAdapter_HandleRequest_Ping(t *testing.T) {
	g := NewGateway(nil)
	adapter := NewMCPAdapter(g)

	req := &MCPRequest{
		JSONRPC: JSONRPC,
		ID:      1,
		Method:  "ping",
	}

	resp := adapter.HandleRequest(context.Background(), req)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	if resp.Result == nil {
		t.Error("expected result")
	}
}

func TestMCPAdapter_HandleRequest_Initialized(t *testing.T) {
	g := NewGateway(nil)
	adapter := NewMCPAdapter(g)

	req := &MCPRequest{
		JSONRPC: JSONRPC,
		Method:  "notifications/initialized",
	}

	resp := adapter.HandleRequest(context.Background(), req)
	if resp != nil {
		t.Error("expected nil response for notification")
	}

	if !adapter.IsInitialized() {
		t.Error("expected adapter to be initialized")
	}
}

func TestMCPAdapter_HandleRequest_Shutdown(t *testing.T) {
	g := NewGateway(nil)
	adapter := NewMCPAdapter(g)

	adapter.initialized = true

	req := &MCPRequest{
		JSONRPC: JSONRPC,
		ID:      1,
		Method:  "shutdown",
	}

	resp := adapter.HandleRequest(context.Background(), req)
	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	if adapter.IsInitialized() {
		t.Error("expected adapter to be not initialized after shutdown")
	}
}

func TestMCPAdapter_Gateway(t *testing.T) {
	g := NewGateway(nil)
	adapter := NewMCPAdapter(g)

	if adapter.Gateway() != g {
		t.Error("expected same gateway instance")
	}
}

func TestUnitNameToToolName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"model.pull", "model_pull"},
		{"inference.chat", "inference_chat"},
		{"device.detect", "device_detect"},
		{"simple", "simple"},
		{"multi.dot.name", "multi_dot_name"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := unitNameToToolName(tt.input)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestToolNameToUnitName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"model_pull", "model.pull"},
		{"inference_chat", "inference.chat"},
		{"device_detect", "device.detect"},
		{"simple", "simple"},
		{"multi_dot_name", "multi.dot_name"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toolNameToUnitName(tt.input)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestMCPError_Error(t *testing.T) {
	err := &MCPError{
		Code:    MCPErrorCodeInternalError,
		Message: "internal error",
	}

	expected := "MCP error [-32603]: internal error"
	if err.Error() != expected {
		t.Errorf("expected %s, got %s", expected, err.Error())
	}
}

func TestMCPAdapter_successResponse(t *testing.T) {
	g := NewGateway(nil)
	adapter := NewMCPAdapter(g)

	resp := adapter.successResponse(1, map[string]string{"status": "ok"})

	if resp.JSONRPC != JSONRPC {
		t.Errorf("expected JSONRPC %s, got %s", JSONRPC, resp.JSONRPC)
	}

	if resp.ID != 1 {
		t.Errorf("expected ID 1, got %v", resp.ID)
	}

	if resp.Error != nil {
		t.Errorf("unexpected error: %v", resp.Error)
	}

	if resp.Result == nil {
		t.Error("expected result")
	}
}

func TestMCPAdapter_errorResponse(t *testing.T) {
	g := NewGateway(nil)
	adapter := NewMCPAdapter(g)

	resp := adapter.errorResponse(1, MCPErrorCodeInternalError, "test error")

	if resp.JSONRPC != JSONRPC {
		t.Errorf("expected JSONRPC %s, got %s", JSONRPC, resp.JSONRPC)
	}

	if resp.ID != 1 {
		t.Errorf("expected ID 1, got %v", resp.ID)
	}

	if resp.Error == nil {
		t.Fatal("expected error")
	}

	if resp.Error.Code != MCPErrorCodeInternalError {
		t.Errorf("expected error code %d, got %d", MCPErrorCodeInternalError, resp.Error.Code)
	}

	if resp.Error.Message != "test error" {
		t.Errorf("expected error message 'test error', got %s", resp.Error.Message)
	}
}

func TestMCPAdapter_errorResponseWithData(t *testing.T) {
	g := NewGateway(nil)
	adapter := NewMCPAdapter(g)

	resp := adapter.errorResponseWithData(1, MCPErrorCodeInternalError, "test error", map[string]string{"detail": "info"})

	if resp.Error == nil {
		t.Fatal("expected error")
	}

	if resp.Error.Data == nil {
		t.Error("expected error data")
	}
}

func TestIsCommand(t *testing.T) {
	t.Run("with command", func(t *testing.T) {
		reg := unit.NewRegistry()
		cmd := &mockCommand{name: "test.cmd", domain: "test"}
		reg.RegisterCommand(cmd)
		g := NewGateway(reg)

		if !isCommand("test.cmd", g) {
			t.Error("expected true for command")
		}
	})

	t.Run("without command", func(t *testing.T) {
		g := NewGateway(nil)

		if isCommand("unknown.cmd", g) {
			t.Error("expected false for unknown command")
		}
	})
}
