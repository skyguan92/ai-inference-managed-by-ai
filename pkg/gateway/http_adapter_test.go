package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestHTTPAdapter_ServeHTTP(t *testing.T) {
	reg := unit.NewRegistry()
	cmd := &mockCommand{name: "test.echo", domain: "test"}
	_ = reg.RegisterCommand(cmd)
	g := NewGateway(reg)
	adapter := NewHTTPAdapter(g)

	t.Run("successful request", func(t *testing.T) {
		body := `{"type":"command","unit":"test.echo","input":{"msg":"hello"}}`
		req := httptest.NewRequest(http.MethodPost, "/api/v2/execute", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", ContentTypeJSON)
		rec := httptest.NewRecorder()

		adapter.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var resp Response
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if !resp.Success {
			t.Errorf("expected success, got error: %v", resp.Error)
		}

		if resp.Meta.RequestID == "" {
			t.Error("expected request_id in response")
		}
		if rec.Header().Get(HeaderRequestID) == "" {
			t.Error("expected X-Request-ID header")
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v2/execute", nil)
		rec := httptest.NewRecorder()

		adapter.ServeHTTP(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status 405, got %d", rec.Code)
		}
	})

	t.Run("invalid content type", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v2/execute", bytes.NewBufferString("{}"))
		req.Header.Set("Content-Type", "text/plain")
		rec := httptest.NewRecorder()

		adapter.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnsupportedMediaType {
			t.Errorf("expected status 415, got %d", rec.Code)
		}
	})

	t.Run("invalid JSON body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v2/execute", bytes.NewBufferString("not json"))
		req.Header.Set("Content-Type", ContentTypeJSON)
		rec := httptest.NewRecorder()

		adapter.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
	})

	t.Run("empty body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v2/execute", bytes.NewBufferString(""))
		req.Header.Set("Content-Type", ContentTypeJSON)
		rec := httptest.NewRecorder()

		adapter.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
	})

	t.Run("unit not found", func(t *testing.T) {
		body := `{"type":"command","unit":"unknown.unit"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v2/execute", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", ContentTypeJSON)
		rec := httptest.NewRecorder()

		adapter.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rec.Code)
		}
	})

	t.Run("trace id from header", func(t *testing.T) {
		body := `{"type":"command","unit":"test.echo"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v2/execute", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", ContentTypeJSON)
		req.Header.Set(HeaderTraceID, "custom-trace-123")
		rec := httptest.NewRecorder()

		adapter.ServeHTTP(rec, req)

		var resp Response
		json.Unmarshal(rec.Body.Bytes(), &resp)

		if resp.Meta.TraceID != "custom-trace-123" {
			t.Errorf("expected trace_id custom-trace-123, got %s", resp.Meta.TraceID)
		}
	})

	t.Run("no content type header", func(t *testing.T) {
		body := `{"type":"command","unit":"test.echo"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v2/execute", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()

		adapter.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})
}

func TestHTTPAdapter_WriteResponse(t *testing.T) {
	g := NewGateway(nil)
	adapter := NewHTTPAdapter(g)

	t.Run("success response", func(t *testing.T) {
		rec := httptest.NewRecorder()
		resp := &Response{
			Success: true,
			Data:    map[string]any{"result": "ok"},
			Meta: &ResponseMeta{
				RequestID: "req-123",
				TraceID:   "trc-456",
			},
		}

		adapter.writeResponse(rec, resp)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		if rec.Header().Get(HeaderRequestID) != "req-123" {
			t.Error("expected X-Request-ID header")
		}
		if rec.Header().Get(HeaderTraceID) != "trc-456" {
			t.Error("expected X-Trace-ID header")
		}
	})
}

func TestErrorToStatusCode(t *testing.T) {
	tests := []struct {
		code     string
		expected int
	}{
		{ErrCodeInvalidRequest, http.StatusBadRequest},
		{ErrCodeValidationFailed, http.StatusBadRequest},
		{ErrCodeUnauthorized, http.StatusUnauthorized},
		{ErrCodeRateLimited, http.StatusTooManyRequests},
		{ErrCodeUnitNotFound, http.StatusNotFound},
		{ErrCodeResourceNotFound, http.StatusNotFound},
		{ErrCodeTimeout, http.StatusRequestTimeout},
		{ErrCodeExecutionFailed, http.StatusInternalServerError},
		{ErrCodeInternalError, http.StatusInternalServerError},
		{"UNKNOWN", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			result := errorToStatusCode(&ErrorInfo{Code: tt.code})
			if result != tt.expected {
				t.Errorf("errorToStatusCode(%s) = %d, expected %d", tt.code, result, tt.expected)
			}
		})
	}
}

func TestHTTPAdapter_Gateway(t *testing.T) {
	g := NewGateway(nil)
	adapter := NewHTTPAdapter(g)

	if adapter.Gateway() != g {
		t.Error("expected same gateway instance")
	}
}

func BenchmarkHTTPAdapter_ServeHTTP(b *testing.B) {
	reg := unit.NewRegistry()
	cmd := &mockCommand{name: "test.echo", domain: "test"}
	_ = reg.RegisterCommand(cmd)
	g := NewGateway(reg)
	adapter := NewHTTPAdapter(g)

	body := `{"type":"command","unit":"test.echo","input":{"msg":"hello"}}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v2/execute", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", ContentTypeJSON)
		rec := httptest.NewRecorder()
		adapter.ServeHTTP(rec, req)
	}
}

func TestHTTPAdapter_Timeout(t *testing.T) {
	reg := unit.NewRegistry()
	slowCmd := &mockCommand{
		name:   "test.slow",
		domain: "test",
		execute: func(ctx context.Context, input any) (any, error) {
			select {
			case <-time.After(5 * time.Second):
				return "done", nil
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		},
	}
	_ = reg.RegisterCommand(slowCmd)

	g := NewGateway(reg, WithTimeout(100*time.Millisecond))
	adapter := NewHTTPAdapter(g)

	body := `{"type":"command","unit":"test.slow"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/execute", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", ContentTypeJSON)
	rec := httptest.NewRecorder()

	adapter.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}
}
