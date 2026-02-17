package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/config"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestNewStartCommand(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)
	cfg := config.Default()

	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     NewOutputOptions(),
		cfg:      cfg,
	}

	cmd := NewStartCommand(root)
	assert.NotNil(t, cmd)
	assert.Equal(t, "start", cmd.Use)
	assert.NotNil(t, cmd.RunE)
}

func TestStartCommand_Flags(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)
	cfg := config.Default()

	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     NewOutputOptions(),
		cfg:      cfg,
	}

	cmd := NewStartCommand(root)

	portFlag := cmd.Flags().Lookup("port")
	assert.NotNil(t, portFlag)
	assert.Equal(t, "p", portFlag.Shorthand)

	addrFlag := cmd.Flags().Lookup("addr")
	assert.NotNil(t, addrFlag)

	tlsCertFlag := cmd.Flags().Lookup("tls-cert")
	assert.NotNil(t, tlsCertFlag)

	tlsKeyFlag := cmd.Flags().Lookup("tls-key")
	assert.NotNil(t, tlsKeyFlag)
}

func TestHandleHealth(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	handler := handleHealth(gw)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/health", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var resp map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "healthy", resp["status"])
	assert.NotEmpty(t, resp["timestamp"])
}

func TestHandleExecute_InvalidMethod(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	handler := handleExecute(gw)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/execute", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestHandleExecute_InvalidBody(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	handler := handleExecute(gw)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/execute", bytes.NewBufferString("invalid json"))
	rec := httptest.NewRecorder()

	handler(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
}

func TestHandleExecute_ValidRequest(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	handler := handleExecute(gw)

	body := map[string]any{
		"type":  "query",
		"unit":  "model.list",
		"input": map[string]any{},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/execute", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	handler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
}

func TestWriteJSONError(t *testing.T) {
	rec := httptest.NewRecorder()

	writeJSONError(rec, http.StatusBadRequest, "invalid_request", "test error message")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var resp map[string]any
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp["success"].(bool))

	errMap := resp["error"].(map[string]any)
	assert.Equal(t, "invalid_request", errMap["code"])
	assert.Equal(t, "test error message", errMap["message"])
}

func TestRunStart_ContextCancellation(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)
	cfg := config.Default()
	cfg.API.ListenAddr = "127.0.0.1:0"

	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     NewOutputOptions(),
		cfg:      cfg,
	}
	root.opts.Writer = buf

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := runStart(ctx, root, "", 0, "", "")
	require.NoError(t, err)
}

func TestRunStart_WithAddrOverride(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)
	cfg := config.Default()

	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     NewOutputOptions(),
		cfg:      cfg,
	}
	root.opts.Writer = buf

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := runStart(ctx, root, "127.0.0.1:0", 0, "", "")
	require.NoError(t, err)
}
