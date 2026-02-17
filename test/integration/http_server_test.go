package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	registrypkg "github.com/jguan/ai-inference-managed-by-ai/pkg/registry"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestGateway(t *testing.T) *gateway.Gateway {
	registry := unit.NewRegistry()
	err := registrypkg.RegisterAll(registry)
	require.NoError(t, err)
	return gateway.NewGateway(registry)
}

func TestHTTPServerIntegrationHealth(t *testing.T) {
	gw := createTestGateway(t)
	_ = gateway.NewServer(gw, gateway.ServerConfig{
		Addr: ":0",
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "healthy", resp["status"])
}

func TestHTTPServerIntegrationExecute(t *testing.T) {
	gw := createTestGateway(t)
	adapter := gateway.NewHTTPAdapter(gw)

	t.Run("execute model.list", func(t *testing.T) {
		body := map[string]any{
			"type":  "query",
			"unit":  "model.list",
			"input": map[string]any{},
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		adapter.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp gateway.Response
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Nil(t, resp.Error)
	})

	t.Run("execute engine.list", func(t *testing.T) {
		body := map[string]any{
			"type":  "query",
			"unit":  "engine.list",
			"input": map[string]any{},
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		adapter.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp gateway.Response
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("execute service.list", func(t *testing.T) {
		body := map[string]any{
			"type":  "query",
			"unit":  "service.list",
			"input": map[string]any{},
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		adapter.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp gateway.Response
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("execute app.list", func(t *testing.T) {
		body := map[string]any{
			"type":  "query",
			"unit":  "app.list",
			"input": map[string]any{},
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		adapter.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp gateway.Response
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("execute with trace id header", func(t *testing.T) {
		body := map[string]any{
			"type":  "query",
			"unit":  "model.list",
			"input": map[string]any{},
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Trace-ID", "test-trace-456")
		rec := httptest.NewRecorder()

		adapter.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var resp gateway.Response
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.True(t, resp.Success)
		assert.Equal(t, "test-trace-456", resp.Meta.TraceID)
	})
}

func TestHTTPServerIntegrationModelCRUD(t *testing.T) {
	gw := createTestGateway(t)
	adapter := gateway.NewHTTPAdapter(gw)

	t.Run("create model", func(t *testing.T) {
		body := map[string]any{
			"type": "command",
			"unit": "model.create",
			"input": map[string]any{
				"name":   "test-model",
				"source": "ollama",
			},
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		adapter.ServeHTTP(rec, req)

		var resp gateway.Response
		json.Unmarshal(rec.Body.Bytes(), &resp)
	})

	t.Run("get model", func(t *testing.T) {
		body := map[string]any{
			"type": "query",
			"unit": "model.get",
			"input": map[string]any{
				"id": "test-model",
			},
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		adapter.ServeHTTP(rec, req)

		var resp gateway.Response
		json.Unmarshal(rec.Body.Bytes(), &resp)
	})

	t.Run("delete model", func(t *testing.T) {
		body := map[string]any{
			"type": "command",
			"unit": "model.delete",
			"input": map[string]any{
				"id": "test-model",
			},
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		adapter.ServeHTTP(rec, req)

		var resp gateway.Response
		json.Unmarshal(rec.Body.Bytes(), &resp)
	})
}

func TestHTTPServerIntegrationEngineEndpoints(t *testing.T) {
	gw := createTestGateway(t)
	adapter := gateway.NewHTTPAdapter(gw)

	t.Run("list engines", func(t *testing.T) {
		body := map[string]any{
			"type":  "query",
			"unit":  "engine.list",
			"input": map[string]any{},
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		adapter.ServeHTTP(rec, req)

		var resp gateway.Response
		json.Unmarshal(rec.Body.Bytes(), &resp)
	})

	t.Run("get engine", func(t *testing.T) {
		body := map[string]any{
			"type": "query",
			"unit": "engine.get",
			"input": map[string]any{
				"name": "ollama",
			},
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		adapter.ServeHTTP(rec, req)

		var resp gateway.Response
		json.Unmarshal(rec.Body.Bytes(), &resp)
	})

	t.Run("start engine", func(t *testing.T) {
		body := map[string]any{
			"type": "command",
			"unit": "engine.start",
			"input": map[string]any{
				"name": "test-engine",
			},
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		adapter.ServeHTTP(rec, req)

		var resp gateway.Response
		json.Unmarshal(rec.Body.Bytes(), &resp)
	})

	t.Run("stop engine", func(t *testing.T) {
		body := map[string]any{
			"type": "command",
			"unit": "engine.stop",
			"input": map[string]any{
				"name": "test-engine",
			},
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		adapter.ServeHTTP(rec, req)

		var resp gateway.Response
		json.Unmarshal(rec.Body.Bytes(), &resp)
	})
}

func TestHTTPServerIntegrationNotFound(t *testing.T) {
	gw := createTestGateway(t)
	adapter := gateway.NewHTTPAdapter(gw)

	body := map[string]any{
		"type":  "query",
		"unit":  "nonexistent.unit",
		"input": map[string]any{},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	adapter.ServeHTTP(rec, req)

	assert.NotEqual(t, http.StatusOK, rec.Code)
}

func TestHTTPServerIntegrationMethodNotAllowed(t *testing.T) {
	gw := createTestGateway(t)
	adapter := gateway.NewHTTPAdapter(gw)

	req := httptest.NewRequest(http.MethodGet, "/execute", nil)
	rec := httptest.NewRecorder()

	adapter.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestHTTPServerIntegrationWithServer(t *testing.T) {
	gw := createTestGateway(t)
	server := gateway.NewServer(gw, gateway.ServerConfig{
		Addr:            "127.0.0.1:0",
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    5 * time.Second,
		ShutdownTimeout: 2 * time.Second,
	})

	go func() {
		server.Start()
	}()

	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := server.Stop(ctx)
	assert.NoError(t, err)
}

func TestHTTPServerIntegrationOpenAPI(t *testing.T) {
	gw := createTestGateway(t)
	_ = gateway.NewServer(gw, gateway.ServerConfig{
		Addr: ":0",
	})

	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	rec := httptest.NewRecorder()

	mux := http.NewServeMux()
	mux.HandleFunc("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"openapi": "3.0.0"}`))
	})
	mux.ServeHTTP(rec, req)

	var resp map[string]any
	if rec.Code == http.StatusOK {
		err := json.Unmarshal(rec.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.Contains(t, resp, "openapi")
	}
}

func TestHTTPServerIntegrationAllUnitsViaExecute(t *testing.T) {
	gw := createTestGateway(t)
	adapter := gateway.NewHTTPAdapter(gw)

	registry := gw.Registry()
	queries := registry.ListQueries()

	for _, q := range queries {
		t.Run("query/"+q.Name(), func(t *testing.T) {
			body := map[string]any{
				"type":  "query",
				"unit":  q.Name(),
				"input": map[string]any{},
			}
			bodyBytes, _ := json.Marshal(body)

			req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			adapter.ServeHTTP(rec, req)

			var resp gateway.Response
			err := json.Unmarshal(rec.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.NotNil(t, resp.Meta)
		})
	}
}
