package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS(t *testing.T) {
	t.Run("handles preflight request", func(t *testing.T) {
		cfg := DefaultCORSConfig()
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		wrappedHandler := CORS(cfg)(handler)

		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Origin", "http://example.com")
		rec := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent {
			t.Errorf("expected status 204, got %d", rec.Code)
		}

		if rec.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
			t.Errorf("expected Allow-Origin header, got %s", rec.Header().Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("adds CORS headers to normal request", func(t *testing.T) {
		cfg := DefaultCORSConfig()
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		})

		wrappedHandler := CORS(cfg)(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "http://example.com")
		rec := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		if rec.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
			t.Errorf("expected Allow-Origin header, got %s", rec.Header().Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("passes through request without origin", func(t *testing.T) {
		cfg := DefaultCORSConfig()
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		wrappedHandler := CORS(cfg)(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		if rec.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Error("expected no Allow-Origin header for request without origin")
		}
	})

	t.Run("blocks disallowed origin", func(t *testing.T) {
		cfg := CORSConfig{
			AllowedOrigins: []string{"http://allowed.com"},
		}
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		wrappedHandler := CORS(cfg)(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "http://blocked.com")
		rec := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		if rec.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Error("expected no Allow-Origin header for blocked origin")
		}
	})

	t.Run("allows wildcard origin", func(t *testing.T) {
		cfg := CORSConfig{
			AllowedOrigins: []string{"*"},
		}
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		wrappedHandler := CORS(cfg)(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "http://any.com")
		rec := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rec, req)

		if rec.Header().Get("Access-Control-Allow-Origin") != "http://any.com" {
			t.Errorf("expected Allow-Origin header, got %s", rec.Header().Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("supports credentials", func(t *testing.T) {
		cfg := CORSConfig{
			AllowedOrigins:   []string{"http://example.com"},
			AllowCredentials: true,
		}
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		wrappedHandler := CORS(cfg)(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "http://example.com")
		rec := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rec, req)

		if rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
			t.Error("expected Allow-Credentials header")
		}
	})

	t.Run("sets max age", func(t *testing.T) {
		cfg := CORSConfig{
			AllowedOrigins: []string{"*"},
			MaxAge:         3600,
		}
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		wrappedHandler := CORS(cfg)(handler)

		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Origin", "http://example.com")
		rec := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rec, req)

		if rec.Header().Get("Access-Control-Max-Age") != "3600" {
			t.Errorf("expected Max-Age 3600, got %s", rec.Header().Get("Access-Control-Max-Age"))
		}
	})

	t.Run("supports wildcard subdomain", func(t *testing.T) {
		cfg := CORSConfig{
			AllowedOrigins: []string{"*.example.com"},
		}
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})

		wrappedHandler := CORS(cfg)(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "http://sub.example.com")
		rec := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rec, req)

		if rec.Header().Get("Access-Control-Allow-Origin") != "http://sub.example.com" {
			t.Errorf("expected Allow-Origin for subdomain, got %s", rec.Header().Get("Access-Control-Allow-Origin"))
		}
	})
}

func TestDefaultCORSConfig(t *testing.T) {
	cfg := DefaultCORSConfig()

	if len(cfg.AllowedOrigins) == 0 {
		t.Error("expected allowed origins")
	}
	if len(cfg.AllowedMethods) == 0 {
		t.Error("expected allowed methods")
	}
	if len(cfg.AllowedHeaders) == 0 {
		t.Error("expected allowed headers")
	}
	if cfg.MaxAge <= 0 {
		t.Error("expected positive max age")
	}
}

func TestIntToStr(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{123, "123"},
		{86400, "86400"},
	}

	for _, tt := range tests {
		result := intToStr(tt.input)
		if result != tt.expected {
			t.Errorf("intToStr(%d) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}
