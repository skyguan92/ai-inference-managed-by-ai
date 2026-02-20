package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	agentllm "github.com/jguan/ai-inference-managed-by-ai/pkg/agent/llm"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/service"
)

const localLLMProbeTimeout = 3 * time.Second

// probeOpenAI checks whether baseURL exposes an OpenAI-compatible /v1/models endpoint.
// Returns the first model ID and true on success.
func probeOpenAI(ctx context.Context, baseURL string, timeout time.Duration) (string, bool) {
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, baseURL+"/v1/models", nil)
	if err != nil {
		return "", false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return "", false
	}
	defer func() { _ = resp.Body.Close() }()

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || len(result.Data) == 0 {
		return "", false
	}
	return result.Data[0].ID, true
}

// probeOllama checks whether baseURL exposes an Ollama /api/tags endpoint.
// Returns the first model name and true on success.
func probeOllama(ctx context.Context, baseURL string, timeout time.Duration) (string, bool) {
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, baseURL+"/api/tags", nil)
	if err != nil {
		return "", false
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return "", false
	}
	defer func() { _ = resp.Body.Close() }()

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || len(result.Models) == 0 {
		return "", false
	}
	return result.Models[0].Name, true
}

// detectLocalLLM finds a locally running LLM endpoint usable as an agent backend.
//
// Detection order:
//  1. Running ModelServices with endpoints — OpenAI-compatible probed first, then Ollama.
//  2. Ollama at ollamaAddr (e.g. "localhost:11434") as final fallback.
//
// Returns a configured LLMClient, a human-readable description, and nil error on success.
// Returns (nil, "", nil) when no local service is reachable.
func detectLocalLLM(ctx context.Context, services []service.ModelService, ollamaAddr string) (agentllm.LLMClient, string, error) {
	for _, svc := range services {
		if svc.Status != service.ServiceStatusRunning || len(svc.Endpoints) == 0 {
			continue
		}
		endpoint := svc.Endpoints[0]

		// Try OpenAI-compatible first (vLLM, lmdeploy, etc.)
		if model, ok := probeOpenAI(ctx, endpoint, localLLMProbeTimeout); ok {
			if svc.ModelID != "" {
				model = svc.ModelID
			}
			// Use "local" as placeholder so OpenAIClient doesn't reject the request.
			// Most local inference servers (e.g. vLLM --no-auth) accept any non-empty key.
			client := agentllm.NewOpenAIClient(model, "local", endpoint)
			return client, fmt.Sprintf("local service %q at %s (model: %s)", svc.Name, endpoint, model), nil
		}

		// Try Ollama protocol
		if model, ok := probeOllama(ctx, endpoint, localLLMProbeTimeout); ok {
			if svc.ModelID != "" {
				model = svc.ModelID
			}
			client := agentllm.NewOllamaClient(model, endpoint)
			return client, fmt.Sprintf("local Ollama service %q at %s (model: %s)", svc.Name, endpoint, model), nil
		}
	}

	// Fallback: probe the configured Ollama address
	if ollamaAddr != "" {
		baseURL := "http://" + ollamaAddr
		if model, ok := probeOllama(ctx, baseURL, localLLMProbeTimeout); ok {
			client := agentllm.NewOllamaClient(model, baseURL)
			return client, fmt.Sprintf("local Ollama at %s (model: %s)", ollamaAddr, model), nil
		}
	}

	return nil, "", nil
}
