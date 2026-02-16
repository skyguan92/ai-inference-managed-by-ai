package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestRouter_ServeHTTP(t *testing.T) {
	reg := unit.NewRegistry()

	listQ := &mockQuery{name: "model.list", domain: "model"}
	getQ := &mockQuery{name: "model.get", domain: "model"}
	pullCmd := &mockCommand{name: "model.pull", domain: "model"}
	deleteCmd := &mockCommand{name: "model.delete", domain: "model"}

	reg.RegisterQuery(listQ)
	reg.RegisterQuery(getQ)
	reg.RegisterCommand(pullCmd)
	reg.RegisterCommand(deleteCmd)

	g := NewGateway(reg)
	router := NewRouter(g)

	t.Run("GET /api/v2/models", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v2/models", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("GET /api/v2/models/{id}", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v2/models/llama3", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("POST /api/v2/models/pull", func(t *testing.T) {
		body := `{"source":"ollama","repo":"llama3"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v2/models/pull", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", ContentTypeJSON)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("DELETE /api/v2/models/{id}", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v2/models/llama3", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})

	t.Run("route not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v2/unknown", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rec.Code)
		}
	})

	t.Run("wrong method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/v2/models", nil)
		rec := httptest.NewRecorder()

		router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", rec.Code)
		}
	})
}

func TestRouter_AddRoute(t *testing.T) {
	g := NewGateway(nil)
	router := NewRouter(g)

	customRoute := Route{
		Method: http.MethodGet,
		Path:   "/api/v2/custom/{id}",
		Unit:   "custom.get",
		Type:   TypeQuery,
	}
	router.AddRoute(customRoute)

	routes := router.Routes()
	found := false
	for _, r := range routes {
		if r.Path == "/api/v2/custom/{id}" {
			found = true
			break
		}
	}

	if !found {
		t.Error("expected custom route to be added")
	}
}

func TestRouter_QueryParams(t *testing.T) {
	reg := unit.NewRegistry()
	listQ := &mockQuery{
		name:   "model.list",
		domain: "model",
		execute: func(ctx context.Context, input any) (any, error) {
			return input, nil
		},
	}
	reg.RegisterQuery(listQ)

	g := NewGateway(reg)
	router := NewRouter(g)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/models?type=llm&status=ready", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	var resp Response
	json.Unmarshal(rec.Body.Bytes(), &resp)

	data, ok := resp.Data.(map[string]any)
	if !ok {
		t.Fatal("expected data to be map")
	}

	if data["type"] != "llm" {
		t.Errorf("expected type=llm, got %v", data["type"])
	}
	if data["status"] != "ready" {
		t.Errorf("expected status=ready, got %v", data["status"])
	}
}

func TestPathParamExtractor(t *testing.T) {
	extractor := newPathParamExtractor()

	tests := []struct {
		pattern     string
		path        string
		expected    map[string]string
		shouldMatch bool
	}{
		{
			pattern:     "/api/v2/models/{id}",
			path:        "/api/v2/models/llama3",
			expected:    map[string]string{"id": "llama3"},
			shouldMatch: true,
		},
		{
			pattern:     "/api/v2/models/{id}",
			path:        "/api/v2/models/",
			expected:    nil,
			shouldMatch: false,
		},
		{
			pattern:     "/api/v2/engines/{name}/start",
			path:        "/api/v2/engines/ollama/start",
			expected:    map[string]string{"name": "ollama"},
			shouldMatch: true,
		},
		{
			pattern:     "/api/v2/models",
			path:        "/api/v2/models",
			expected:    map[string]string{},
			shouldMatch: true,
		},
		{
			pattern:     "/api/v2/models",
			path:        "/api/v2/models/extra",
			expected:    nil,
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+" "+tt.path, func(t *testing.T) {
			params, ok := extractor.match(tt.pattern, tt.path)

			if ok != tt.shouldMatch {
				t.Errorf("expected match=%v, got match=%v", tt.shouldMatch, ok)
			}

			if tt.shouldMatch {
				for k, v := range tt.expected {
					if params[k] != v {
						t.Errorf("expected params[%s]=%s, got %s", k, v, params[k])
					}
				}
			}
		})
	}
}

func TestDefaultRoutes(t *testing.T) {
	routes := defaultRoutes()

	if len(routes) == 0 {
		t.Fatal("expected default routes")
	}

	expectedRoutes := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v2/models/pull"},
		{http.MethodGet, "/api/v2/models"},
		{http.MethodGet, "/api/v2/models/{id}"},
		{http.MethodPost, "/api/v2/inference/chat"},
		{http.MethodGet, "/api/v2/devices"},
		{http.MethodGet, "/api/v2/engines"},
	}

	for _, expected := range expectedRoutes {
		found := false
		for _, route := range routes {
			if route.Method == expected.method && route.Path == expected.path {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected route %s %s not found", expected.method, expected.path)
		}
	}
}

func TestInputMappers(t *testing.T) {
	t.Run("bodyInputMapper", func(t *testing.T) {
		body := `{"key":"value","number":123}`
		req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", ContentTypeJSON)

		result := bodyInputMapper(req, nil)

		if result["key"] != "value" {
			t.Errorf("expected key=value, got %v", result["key"])
		}
		if result["number"].(float64) != 123 {
			t.Errorf("expected number=123, got %v", result["number"])
		}
	})

	t.Run("queryInputMapper", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test?foo=bar&baz=qux&list=a&list=b", nil)

		result := queryInputMapper(req, nil)

		if result["foo"] != "bar" {
			t.Errorf("expected foo=bar, got %v", result["foo"])
		}
		list, ok := result["list"].([]string)
		if !ok || len(list) != 2 {
			t.Errorf("expected list with 2 elements, got %v", result["list"])
		}
	})

	t.Run("idInputMapper", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		pathParams := map[string]string{"id": "test-123"}

		result := idInputMapper(req, pathParams)

		if result["id"] != "test-123" {
			t.Errorf("expected id=test-123, got %v", result["id"])
		}
	})

	t.Run("nameInputMapper", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		pathParams := map[string]string{"name": "ollama"}

		result := nameInputMapper(req, pathParams)

		if result["name"] != "ollama" {
			t.Errorf("expected name=ollama, got %v", result["name"])
		}
	})

	t.Run("emptyInputMapper", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)

		result := emptyInputMapper(req, nil)

		if len(result) != 0 {
			t.Errorf("expected empty map, got %v", result)
		}
	})
}
