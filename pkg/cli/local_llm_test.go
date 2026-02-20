package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProbeOpenAI_Found(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{{"id": "my-model"}},
			})
		}
	}))
	defer srv.Close()

	model, ok := probeOpenAI(context.Background(), srv.URL, 2*time.Second)
	require.True(t, ok)
	assert.Equal(t, "my-model", model)
}

func TestProbeOpenAI_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, ok := probeOpenAI(context.Background(), srv.URL, 2*time.Second)
	assert.False(t, ok)
}

func TestProbeOpenAI_EmptyData(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			_ = json.NewEncoder(w).Encode(map[string]any{"data": []any{}})
		}
	}))
	defer srv.Close()

	_, ok := probeOpenAI(context.Background(), srv.URL, 2*time.Second)
	assert.False(t, ok)
}

func TestProbeOllama_Found(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"models": []map[string]any{{"name": "llama3.2:latest"}},
			})
		}
	}))
	defer srv.Close()

	model, ok := probeOllama(context.Background(), srv.URL, 2*time.Second)
	require.True(t, ok)
	assert.Equal(t, "llama3.2:latest", model)
}

func TestProbeOllama_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			_ = json.NewEncoder(w).Encode(map[string]any{"models": []any{}})
		}
	}))
	defer srv.Close()

	_, ok := probeOllama(context.Background(), srv.URL, 2*time.Second)
	assert.False(t, ok)
}

func TestProbeOllama_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, ok := probeOllama(context.Background(), srv.URL, 2*time.Second)
	assert.False(t, ok)
}

func TestDetectLocalLLM_RunningServiceOpenAI(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{{"id": "qwen2:7b"}},
			})
		}
	}))
	defer srv.Close()

	services := []service.ModelService{
		{Name: "my-svc", Status: service.ServiceStatusRunning, Endpoints: []string{srv.URL}, ModelID: "qwen2:7b"},
	}

	client, desc, err := detectLocalLLM(context.Background(), services, "")
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, "openai", client.Name())
	assert.Equal(t, "qwen2:7b", client.ModelName())
	assert.Contains(t, desc, "qwen2:7b")
}

func TestDetectLocalLLM_RunningServiceOllama(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"models": []map[string]any{{"name": "llama3.2:latest"}},
			})
		}
	}))
	defer srv.Close()

	services := []service.ModelService{
		{Name: "ollama-svc", Status: service.ServiceStatusRunning, Endpoints: []string{srv.URL}, ModelID: "llama3.2:latest"},
	}

	client, desc, err := detectLocalLLM(context.Background(), services, "")
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, "ollama", client.Name())
	assert.Contains(t, desc, "llama3.2:latest")
}

func TestDetectLocalLLM_SkipsStoppedServices(t *testing.T) {
	services := []service.ModelService{
		{Name: "stopped-svc", Status: service.ServiceStatusStopped, Endpoints: []string{"http://localhost:9999"}, ModelID: "some-model"},
	}

	client, _, err := detectLocalLLM(context.Background(), services, "")
	require.NoError(t, err)
	assert.Nil(t, client)
}

func TestDetectLocalLLM_OllamaFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"models": []map[string]any{{"name": "llama3.2:latest"}},
			})
		}
	}))
	defer srv.Close()

	// No running services; fallback to Ollama at srv.Listener.Addr().String()
	client, desc, err := detectLocalLLM(context.Background(), nil, srv.Listener.Addr().String())
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, "ollama", client.Name())
	assert.Contains(t, desc, "llama3.2:latest")
}

func TestDetectLocalLLM_NoneAvailable(t *testing.T) {
	client, _, err := detectLocalLLM(context.Background(), nil, "")
	require.NoError(t, err)
	assert.Nil(t, client)
}
