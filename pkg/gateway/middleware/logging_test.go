package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestLogging(t *testing.T) {
	t.Run("logs request details", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success":true}`))
		})

		wrappedHandler := Logging(logger)(handler)

		req := httptest.NewRequest(http.MethodPost, "/api/v2/execute", bytes.NewBufferString(`{"type":"command"}`))
		req.RemoteAddr = "192.168.1.1:12345"
		ctx := unit.WithRequestID(req.Context(), "test-req-123")
		ctx = unit.WithTraceID(ctx, "test-trace-456")
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rec, req)

		logOutput := buf.String()
		if !strings.Contains(logOutput, `"method":"POST"`) {
			t.Error("expected method in log output")
		}
		if !strings.Contains(logOutput, `"path":"/api/v2/execute"`) {
			t.Error("expected path in log output")
		}
		if !strings.Contains(logOutput, `"status":200`) {
			t.Error("expected status in log output")
		}
		if !strings.Contains(logOutput, `"request_id":"test-req-123"`) {
			t.Error("expected request_id in log output")
		}
		if !strings.Contains(logOutput, `"trace_id":"test-trace-456"`) {
			t.Error("expected trace_id in log output")
		}
		if !strings.Contains(logOutput, `"remote_addr":"192.168.1.1:12345"`) {
			t.Error("expected remote_addr in log output")
		}
	})

	t.Run("logs warning for 4xx status", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		})

		wrappedHandler := Logging(logger)(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rec, req)

		logOutput := buf.String()
		if !strings.Contains(logOutput, `"status":400`) {
			t.Error("expected status 400 in log output")
		}
	})

	t.Run("logs error for 5xx status", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError}))

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		wrappedHandler := Logging(logger)(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rec, req)

		logOutput := buf.String()
		if !strings.Contains(logOutput, `"status":500`) {
			t.Error("expected status 500 in log output")
		}
	})

	t.Run("handles nil logger", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		wrappedHandler := Logging(nil)(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("defaults to 200 status", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ok"))
		})

		wrappedHandler := Logging(logger)(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rec, req)

		logOutput := buf.String()
		if !strings.Contains(logOutput, `"status":200`) {
			t.Error("expected default status 200 in log output")
		}
	})
}
