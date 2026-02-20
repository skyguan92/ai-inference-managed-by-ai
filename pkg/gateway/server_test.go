package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway/middleware"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestNewServer(t *testing.T) {
	g := NewGateway(nil)

	t.Run("default config", func(t *testing.T) {
		config := DefaultServerConfig()
		s := NewServer(g, config)

		if s == nil {
			t.Fatal("expected server")
		}
		if s.config.Addr != ":9090" {
			t.Errorf("expected addr :9090, got %s", s.config.Addr)
		}
	})

	t.Run("custom config", func(t *testing.T) {
		s := NewServer(g, ServerConfig{
			Addr:        ":8080",
			ReadTimeout: 10 * time.Second,
		})

		if s.config.Addr != ":8080" {
			t.Errorf("expected addr :8080, got %s", s.config.Addr)
		}
	})

	t.Run("empty config uses defaults", func(t *testing.T) {
		s := NewServer(g, ServerConfig{})

		if s.config.Addr != ":9090" {
			t.Errorf("expected default addr :9090, got %s", s.config.Addr)
		}
		if s.config.ReadTimeout != 15*time.Second {
			t.Errorf("expected default read timeout 15s, got %v", s.config.ReadTimeout)
		}
	})
}

func TestServer_HandleHealth(t *testing.T) {
	g := NewGateway(nil)
	s := NewServer(g, DefaultServerConfig())

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	s.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]string
	json.Unmarshal(rec.Body.Bytes(), &resp)

	if resp["status"] != "healthy" {
		t.Errorf("expected status healthy, got %s", resp["status"])
	}
}

func TestServer_HandleOpenAPI(t *testing.T) {
	g := NewGateway(nil)
	s := NewServer(g, DefaultServerConfig())

	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	rec := httptest.NewRecorder()

	s.handleOpenAPI(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	contentType := rec.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected content-type application/json, got %s", contentType)
	}

	var spec map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &spec); err != nil {
		t.Fatalf("failed to parse openapi spec: %v", err)
	}

	if spec["openapi"] != "3.0.0" {
		t.Errorf("expected openapi 3.0.0, got %v", spec["openapi"])
	}
}

func TestServer_BuildHandler(t *testing.T) {
	reg := unit.NewRegistry()
	cmd := &mockCommand{name: "test.echo", domain: "test"}
	_ = reg.RegisterCommand(cmd)

	g := NewGateway(reg)
	s := NewServer(g, DefaultServerConfig())

	handler := s.buildHandler()

	t.Run("execute endpoint", func(t *testing.T) {
		body := `{"type":"command","unit":"test.echo"}`
		req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", ContentTypeJSON)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rec.Code)
		}
	})
}

func TestServer_Accessors(t *testing.T) {
	g := NewGateway(nil)
	config := ServerConfig{
		Addr:        ":8888",
		ReadTimeout: 20 * time.Second,
	}
	s := NewServer(g, config)

	t.Run("Gateway", func(t *testing.T) {
		if s.Gateway() != g {
			t.Error("expected same gateway")
		}
	})

	t.Run("Config", func(t *testing.T) {
		cfg := s.Config()
		if cfg.Addr != ":8888" {
			t.Errorf("expected addr :8888, got %s", cfg.Addr)
		}
	})

	t.Run("Router", func(t *testing.T) {
		if s.Router() == nil {
			t.Error("expected router")
		}
	})
}

func TestServer_Stop(t *testing.T) {
	g := NewGateway(nil)
	s := NewServer(g, ServerConfig{
		Addr:            ":0",
		ShutdownTimeout: 1 * time.Second,
	})

	t.Run("stop without start", func(t *testing.T) {
		err := s.Stop(context.Background())
		if err != nil {
			t.Errorf("expected no error stopping non-started server, got %v", err)
		}
	})
}

func TestDefaultServerConfig(t *testing.T) {
	cfg := DefaultServerConfig()

	if cfg.Addr != ":9090" {
		t.Errorf("expected addr :9090, got %s", cfg.Addr)
	}
	if cfg.ReadTimeout != 15*time.Second {
		t.Errorf("expected read timeout 15s, got %v", cfg.ReadTimeout)
	}
	if cfg.WriteTimeout != 30*time.Second {
		t.Errorf("expected write timeout 30s, got %v", cfg.WriteTimeout)
	}
	if cfg.IdleTimeout != 60*time.Second {
		t.Errorf("expected idle timeout 60s, got %v", cfg.IdleTimeout)
	}
	if cfg.ShutdownTimeout != 10*time.Second {
		t.Errorf("expected shutdown timeout 10s, got %v", cfg.ShutdownTimeout)
	}
}

func TestServer_CORS(t *testing.T) {
	reg := unit.NewRegistry()
	cmd := &mockCommand{name: "test.echo", domain: "test"}
	_ = reg.RegisterCommand(cmd)

	g := NewGateway(reg)
	s := NewServer(g, ServerConfig{
		EnableCORS: true,
		CORSConfig: middlewareCORSConfig(),
	})

	handler := s.buildHandler()

	req := httptest.NewRequest(http.MethodOptions, "/execute", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status 204 for preflight, got %d", rec.Code)
	}

	if rec.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Error("expected CORS headers")
	}
}

func middlewareCORSConfig() middleware.CORSConfig {
	return middleware.CORSConfig{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: false,
		MaxAge:           86400,
	}
}

func TestServer_PanicRecovery(t *testing.T) {
	reg := unit.NewRegistry()
	panicCmd := &mockCommand{
		name:   "test.panic",
		domain: "test",
		execute: func(ctx context.Context, input any) (any, error) {
			panic("test panic")
		},
	}
	_ = reg.RegisterCommand(panicCmd)

	g := NewGateway(reg)
	s := NewServer(g, DefaultServerConfig())

	handler := s.buildHandler()

	body := `{"type":"command","unit":"test.panic"}`
	req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", ContentTypeJSON)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}

	respBody := rec.Body.String()
	if !strings.Contains(respBody, "INTERNAL_ERROR") {
		t.Errorf("expected internal error in response, got %s", respBody)
	}
}
