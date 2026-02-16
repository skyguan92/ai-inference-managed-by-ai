package middleware

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRecovery(t *testing.T) {
	t.Run("recovers from panic", func(t *testing.T) {
		var logBuf strings.Builder
		logger := slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelError}))

		panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("something went wrong")
		})

		wrappedHandler := Recovery(logger)(panicHandler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", rec.Code)
		}

		body := rec.Body.String()
		if !strings.Contains(body, `"success":false`) {
			t.Error("expected success false in response")
		}
		if !strings.Contains(body, `"code":"INTERNAL_ERROR"`) {
			t.Error("expected INTERNAL_ERROR code in response")
		}

		logOutput := logBuf.String()
		if !strings.Contains(logOutput, "panic recovered") {
			t.Error("expected panic recovered in log")
		}
	})

	t.Run("recovers from panic with error type", func(t *testing.T) {
		var logBuf strings.Builder
		logger := slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelError}))

		panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic(errors.New("explicit error"))
		})

		wrappedHandler := Recovery(logger)(panicHandler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", rec.Code)
		}

		logOutput := logBuf.String()
		if !strings.Contains(logOutput, "explicit error") {
			t.Error("expected error message in log")
		}
	})

	t.Run("passes through normal requests", func(t *testing.T) {
		logger := slog.New(slog.NewJSONHandler(nil, nil))

		normalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		})

		wrappedHandler := Recovery(logger)(normalHandler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		if rec.Body.String() != "ok" {
			t.Errorf("expected body 'ok', got '%s'", rec.Body.String())
		}
	})

	t.Run("handles nil logger", func(t *testing.T) {
		panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("test panic")
		})

		wrappedHandler := Recovery(nil)(panicHandler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", rec.Code)
		}
	})
}
