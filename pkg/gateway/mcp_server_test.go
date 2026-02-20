package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestNewMCPServer(t *testing.T) {
	g := NewGateway(nil)
	adapter := NewMCPAdapter(g)

	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	server := NewMCPServer(adapter, stdin, stdout, stderr)
	if server == nil {
		t.Fatal("expected server, got nil")
	}

	if server.adapter != adapter {
		t.Error("expected same adapter instance")
	}
}

func TestMCPServer_handleLine(t *testing.T) {
	g := NewGateway(nil)
	adapter := NewMCPAdapter(g)

	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	server := NewMCPServer(adapter, stdin, stdout, stderr)

	t.Run("valid request", func(t *testing.T) {
		stdout.Reset()

		req := &MCPRequest{
			JSONRPC: JSONRPC,
			ID:      1,
			Method:  "ping",
		}
		reqJSON, _ := json.Marshal(req)

		server.handleLine(context.Background(), reqJSON)

		if stdout.String() == "" {
			t.Error("expected output")
		}

		var resp MCPResponse
		if err := json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if resp.Error != nil {
			t.Errorf("unexpected error: %v", resp.Error)
		}
	})

	t.Run("initialize request", func(t *testing.T) {
		stdout.Reset()

		req := &MCPRequest{
			JSONRPC: JSONRPC,
			ID:      1,
			Method:  "initialize",
			Params:  json.RawMessage(`{"protocolVersion":"2024-11-05","clientInfo":{"name":"test","version":"1.0"}}`),
		}
		reqJSON, _ := json.Marshal(req)

		server.handleLine(context.Background(), reqJSON)

		var resp MCPResponse
		if err := json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		result, ok := resp.Result.(map[string]any)
		if !ok {
			t.Fatalf("expected map result, got %T", resp.Result)
		}

		serverInfo, ok := result["serverInfo"].(map[string]any)
		if !ok {
			t.Fatalf("expected serverInfo map, got %T", result["serverInfo"])
		}

		if serverInfo["name"] != MCPServerName {
			t.Errorf("expected server name %s, got %s", MCPServerName, serverInfo["name"])
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		stdout.Reset()

		server.handleLine(context.Background(), []byte("invalid"))

		var resp MCPResponse
		if err := json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if resp.Error == nil {
			t.Fatal("expected error")
		}

		if resp.Error.Code != MCPErrorCodeParseError {
			t.Errorf("expected error code %d, got %d", MCPErrorCodeParseError, resp.Error.Code)
		}
	})

	t.Run("notification returns nil", func(t *testing.T) {
		stdout.Reset()

		req := &MCPRequest{
			JSONRPC: JSONRPC,
			Method:  "notifications/initialized",
		}
		reqJSON, _ := json.Marshal(req)

		server.handleLine(context.Background(), reqJSON)

		if stdout.String() != "" {
			t.Errorf("expected no output for notification, got: %s", stdout.String())
		}
	})

	t.Run("tool call with registered command", func(t *testing.T) {
		stdout.Reset()

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
		server := NewMCPServer(adapter, stdin, stdout, stderr)

		req := &MCPRequest{
			JSONRPC: JSONRPC,
			ID:      1,
			Method:  "tools/call",
			Params:  json.RawMessage(`{"name":"test_echo","arguments":{"msg":"hello"}}`),
		}
		reqJSON, _ := json.Marshal(req)

		server.handleLine(context.Background(), reqJSON)

		var resp MCPResponse
		if err := json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if resp.Error != nil {
			t.Fatalf("unexpected error: %v", resp.Error)
		}
	})

	t.Run("resource read", func(t *testing.T) {
		stdout.Reset()

		reg := unit.NewRegistry()
		res := &mockResource{
			uri: "asms://test/resource",
			get: func(ctx context.Context) (any, error) {
				return map[string]any{"data": "test"}, nil
			},
		}
		_ = reg.RegisterResource(res)
		g := NewGateway(reg)
		adapter := NewMCPAdapter(g)
		server := NewMCPServer(adapter, stdin, stdout, stderr)

		req := &MCPRequest{
			JSONRPC: JSONRPC,
			ID:      1,
			Method:  "resources/read",
			Params:  json.RawMessage(`{"uri":"asms://test/resource"}`),
		}
		reqJSON, _ := json.Marshal(req)

		server.handleLine(context.Background(), reqJSON)

		var resp MCPResponse
		if err := json.Unmarshal([]byte(strings.TrimSpace(stdout.String())), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if resp.Error != nil {
			t.Fatalf("unexpected error: %v", resp.Error)
		}
	})
}

func TestMCPServer_writeResponse(t *testing.T) {
	g := NewGateway(nil)
	adapter := NewMCPAdapter(g)

	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	server := NewMCPServer(adapter, stdin, stdout, stderr)

	resp := &MCPResponse{
		JSONRPC: JSONRPC,
		ID:      1,
		Result:  map[string]string{"status": "ok"},
	}

	server.writeResponse(resp)

	output := stdout.String()
	if output == "" {
		t.Error("expected output")
	}

	if !strings.HasSuffix(output, "\n") {
		t.Error("expected output to end with newline")
	}

	var parsed MCPResponse
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("failed to parse output: %v", err)
	}

	if parsed.JSONRPC != JSONRPC {
		t.Errorf("expected JSONRPC %s, got %s", JSONRPC, parsed.JSONRPC)
	}
}

func TestMCPServer_IsRunning(t *testing.T) {
	g := NewGateway(nil)
	adapter := NewMCPAdapter(g)

	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	server := NewMCPServer(adapter, stdin, stdout, stderr)

	if server.IsRunning() {
		t.Error("expected server to not be running initially")
	}
}

func TestMCPSSEServer_NewMCPSSEServer(t *testing.T) {
	g := NewGateway(nil)
	adapter := NewMCPAdapter(g)

	server := NewMCPSSEServer(adapter, ":0")
	if server == nil {
		t.Fatal("expected server, got nil")
	}

	if server.adapter != adapter {
		t.Error("expected same adapter instance")
	}
}

func TestMCPSSEServer_handleHealth(t *testing.T) {
	g := NewGateway(nil)
	adapter := NewMCPAdapter(g)

	server := NewMCPSSEServer(adapter, ":0")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	server.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var result map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if result["status"] != "healthy" {
		t.Errorf("expected status healthy, got %v", result["status"])
	}
}

func TestMCPSSEServer_handleSSE(t *testing.T) {
	g := NewGateway(nil)
	adapter := NewMCPAdapter(g)

	server := NewMCPSSEServer(adapter, ":0")

	t.Run("method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/sse", nil)
		rec := httptest.NewRecorder()

		server.handleSSE(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status 405, got %d", rec.Code)
		}
	})
}

func TestMCPSSEServer_handleMessage(t *testing.T) {
	g := NewGateway(nil)
	adapter := NewMCPAdapter(g)

	server := NewMCPSSEServer(adapter, ":0")

	t.Run("method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/message", nil)
		rec := httptest.NewRecorder()

		server.handleMessage(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status 405, got %d", rec.Code)
		}
	})

	t.Run("missing session", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/message", nil)
		rec := httptest.NewRecorder()

		server.handleMessage(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
	})

	t.Run("session not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/message?session=unknown", nil)
		rec := httptest.NewRecorder()

		server.handleMessage(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rec.Code)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		session := &sseSession{
			id:      "test-session",
			events:  make(chan []byte, 1),
			closeCh: make(chan struct{}),
			adapter: adapter,
		}
		server.mu.Lock()
		server.sessions["test-session"] = session
		server.mu.Unlock()

		req := httptest.NewRequest(http.MethodPost, "/message?session=test-session", strings.NewReader("invalid json"))
		rec := httptest.NewRecorder()

		server.handleMessage(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}

		server.mu.Lock()
		delete(server.sessions, "test-session")
		server.mu.Unlock()
	})

	t.Run("valid request", func(t *testing.T) {
		session := &sseSession{
			id:      "test-session-2",
			events:  make(chan []byte, 10),
			closeCh: make(chan struct{}),
			adapter: adapter,
		}
		server.mu.Lock()
		server.sessions["test-session-2"] = session
		server.mu.Unlock()

		mcpReq := MCPRequest{
			JSONRPC: JSONRPC,
			ID:      1,
			Method:  "ping",
		}
		reqBody, _ := json.Marshal(mcpReq)

		req := httptest.NewRequest(http.MethodPost, "/message?session=test-session-2", bytes.NewReader(reqBody))
		rec := httptest.NewRecorder()

		server.handleMessage(rec, req)

		if rec.Code != http.StatusAccepted {
			t.Errorf("expected status 202, got %d", rec.Code)
		}

		server.mu.Lock()
		delete(server.sessions, "test-session-2")
		server.mu.Unlock()
	})

	t.Run("buffer full", func(t *testing.T) {
		eventsCh := make(chan []byte)
		session := &sseSession{
			id:      "test-session-3",
			events:  eventsCh,
			closeCh: make(chan struct{}),
			adapter: adapter,
		}
		server.mu.Lock()
		server.sessions["test-session-3"] = session
		server.mu.Unlock()

		mcpReq := MCPRequest{
			JSONRPC: JSONRPC,
			ID:      1,
			Method:  "ping",
		}
		reqBody, _ := json.Marshal(mcpReq)

		req := httptest.NewRequest(http.MethodPost, "/message?session=test-session-3", bytes.NewReader(reqBody))
		rec := httptest.NewRecorder()

		server.handleMessage(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status 503, got %d", rec.Code)
		}

		server.mu.Lock()
		delete(server.sessions, "test-session-3")
		server.mu.Unlock()
	})
}

func TestGenerateSessionID(t *testing.T) {
	id := generateSessionID()

	if !strings.HasPrefix(id, "sess_") {
		t.Errorf("expected session ID to start with 'sess_', got %s", id)
	}

	if len(id) < 10 {
		t.Errorf("expected session ID to be at least 10 chars, got %d", len(id))
	}
}

func TestMCPServer_Serve_ContextCancel(t *testing.T) {
	g := NewGateway(nil)
	adapter := NewMCPAdapter(g)

	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	server := NewMCPServer(adapter, stdin, stdout, stderr)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- server.Serve(ctx)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != context.Canceled && err != io.EOF {
			t.Logf("serve returned: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("server did not stop on context cancel")
	}
}

func TestMCPSSEServer_Shutdown(t *testing.T) {
	g := NewGateway(nil)
	adapter := NewMCPAdapter(g)

	server := NewMCPSSEServer(adapter, ":0")

	session := &sseSession{
		id:      "test-session",
		events:  make(chan []byte, 1),
		closeCh: make(chan struct{}),
		adapter: adapter,
	}
	server.mu.Lock()
	server.sessions["test-session"] = session
	server.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	server.mu.RLock()
	count := len(server.sessions)
	server.mu.RUnlock()

	if count != 0 {
		t.Errorf("expected 0 sessions after shutdown, got %d", count)
	}
}
