package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/config"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/metrics"
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

func TestHandlePrometheusMetrics_Get(t *testing.T) {
	rm := metrics.NewRequestMetrics()
	sc := metrics.NewCollector()

	handler := handlePrometheusMetrics(rm, sc)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/metrics", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	ct := rec.Header().Get("Content-Type")
	assert.Contains(t, ct, "text/plain")

	body := rec.Body.String()
	assert.Contains(t, body, "aima_http_requests_total")
	assert.Contains(t, body, "aima_http_errors_total")
	assert.Contains(t, body, "aima_http_request_duration_ms_sum")
}

func TestHandlePrometheusMetrics_InvalidMethod(t *testing.T) {
	rm := metrics.NewRequestMetrics()
	sc := metrics.NewCollector()

	handler := handlePrometheusMetrics(rm, sc)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/metrics", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestHandlePrometheusMetrics_PrometheusFormat(t *testing.T) {
	rm := metrics.NewRequestMetrics()
	// Record some test data
	rm.Record(10*time.Millisecond, false)
	rm.Record(20*time.Millisecond, false)
	rm.Record(5*time.Millisecond, true)

	sc := metrics.NewCollector()
	handler := handlePrometheusMetrics(rm, sc)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/metrics", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	body := rec.Body.String()

	// Verify Prometheus text format conventions
	assert.True(t, strings.Contains(body, "# HELP"), "expected HELP lines")
	assert.True(t, strings.Contains(body, "# TYPE"), "expected TYPE lines")
	assert.Contains(t, body, "aima_http_requests_total 3")
	assert.Contains(t, body, "aima_http_errors_total 1")
}

func TestInstrumentHandler(t *testing.T) {
	rm := metrics.NewRequestMetrics()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := instrumentHandler(inner, rm)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	snap := rm.Snapshot()
	assert.Equal(t, int64(1), snap.TotalRequests)
	assert.Equal(t, int64(0), snap.TotalErrors)
}

func TestInstrumentHandler_ServerError(t *testing.T) {
	rm := metrics.NewRequestMetrics()

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	handler := instrumentHandler(inner, rm)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler(rec, req)

	snap := rm.Snapshot()
	assert.Equal(t, int64(1), snap.TotalRequests)
	assert.Equal(t, int64(1), snap.TotalErrors)
}

func TestResponseWriter_DefaultStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rec, statusCode: http.StatusOK}
	// Write without calling WriteHeader explicitly
	_, _ = rw.Write([]byte("hello"))
	assert.Equal(t, http.StatusOK, rw.statusCode)
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rec, statusCode: http.StatusOK}
	rw.WriteHeader(http.StatusCreated)
	assert.Equal(t, http.StatusCreated, rw.statusCode)
}
