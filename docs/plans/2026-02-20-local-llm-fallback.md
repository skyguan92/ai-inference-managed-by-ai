# Local LLM Fallback Detection Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Auto-detect locally running AIMA inference services and use them as the fallback LLM backend when no API key is configured.

**Architecture:**
- New file `pkg/cli/local_llm.go` holds standalone probe helpers + `detectLocalLLM(ctx, services, ollamaAddr)`.
- `detectLocalLLM` tries each running service's endpoint (OpenAI-compat first, then Ollama protocol), then falls back to the configured Ollama address.
- `RootCommand` gets a `serviceStore` field so `setupAgent` can query running services before giving up.
- When a local service is found, the agent is set up with a no-key LLM client pointing at the local endpoint.

**Tech Stack:** Go 1.23, `net/http`, `net/http/httptest` (tests), `pkg/agent/llm` clients

---

### Task 1: Probe helper functions + `detectLocalLLM`

**Files:**
- Create: `pkg/cli/local_llm.go`
- Create: `pkg/cli/local_llm_test.go`

**Step 1: Write the failing tests**

```go
// pkg/cli/local_llm_test.go
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
	assert.Contains(t, desc, "qwen2:7b")
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

	// No running services; fallback to Ollama at srv.Host
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
```

**Step 2: Run to confirm failure**

```bash
go test ./pkg/cli/... -run TestProbeOpenAI -v 2>&1 | head -20
```
Expected: compile error — `probeOpenAI` undefined.

**Step 3: Write `pkg/cli/local_llm.go`**

```go
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
// Priority:
//  1. Running ModelServices (from serviceStore) with endpoints — OpenAI-compat probed first,
//     then Ollama protocol.
//  2. Ollama at ollamaAddr (e.g. "localhost:11434").
//
// Returns a configured LLMClient, a human-readable description string, and nil error.
// Returns (nil, "", nil) when no local service is found.
func detectLocalLLM(ctx context.Context, services []service.ModelService, ollamaAddr string) (agentllm.LLMClient, string, error) {
	for _, svc := range services {
		if svc.Status != service.ServiceStatusRunning || len(svc.Endpoints) == 0 {
			continue
		}
		endpoint := svc.Endpoints[0]

		// Try OpenAI-compatible (vLLM, lmdeploy, etc.)
		if model, ok := probeOpenAI(ctx, endpoint, localLLMProbeTimeout); ok {
			if svc.ModelID != "" {
				model = svc.ModelID
			}
			client := agentllm.NewOpenAIClient(model, "", endpoint)
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

	// Fallback: probe configured Ollama address
	if ollamaAddr != "" {
		baseURL := "http://" + ollamaAddr
		if model, ok := probeOllama(ctx, baseURL, localLLMProbeTimeout); ok {
			client := agentllm.NewOllamaClient(model, baseURL)
			return client, fmt.Sprintf("local Ollama at %s (model: %s)", ollamaAddr, model), nil
		}
	}

	return nil, "", nil
}
```

**Step 4: Run tests to verify they pass**

```bash
go test ./pkg/cli/... -run "TestProbe|TestDetectLocal" -v 2>&1 | tail -20
```
Expected: all 7 tests PASS.

**Step 5: Commit**

```bash
git add pkg/cli/local_llm.go pkg/cli/local_llm_test.go
git commit -m "feat: add local LLM endpoint probe helpers and detectLocalLLM"
```

---

### Task 2: Wire detectLocalLLM into setupAgent

**Files:**
- Modify: `pkg/cli/root.go`

**Step 1: Write the failing test**

```go
// In pkg/cli/root_test.go or new pkg/cli/setup_agent_test.go
// This test is more of an integration smoke test — hard to unit-test without
// a full RootCommand. Instead, rely on the detectLocalLLM tests above.
// Just verify setupAgent returns nil (no error) when both key and local services are absent.
func TestSetupAgent_NoKeyNoLocalService(t *testing.T) {
    r := &RootCommand{
        cfg: config.Default(),
        // serviceStore: nil  (left nil intentionally)
    }
    r.registry = unit.NewRegistry()
    r.gateway = gateway.NewGateway(r.registry)
    err := r.setupAgent(context.Background())
    assert.NoError(t, err)
    // Agent domain should NOT be registered when there's nothing to use
}
```

(Add this to `pkg/cli/root_test.go` if it exists, otherwise create it.)

**Step 2: Implement the changes to root.go**

Changes needed:
1. Add `serviceStore service.ServiceStore` field to `RootCommand` (after `registry`).
2. In `persistentPreRunE`, assign `r.serviceStore = serviceStore` (after the store is created).
3. Change `r.setupAgent()` call to `r.setupAgent(cmd.Context())`.
4. Change `setupAgent()` signature to `setupAgent(ctx context.Context) error`.
5. Replace the early `return nil` when `LLMAPIKey == ""` with:
   ```go
   if cfg.LLMAPIKey == "" {
       services := r.listRunningServices(ctx)
       var err error
       llmClient, info, err = detectLocalLLM(ctx, services, r.cfg.Engine.OllamaAddr)
       if err != nil || llmClient == nil {
           slog.Debug("no API key and no local LLM service detected, agent unavailable")
           return nil
       }
       slog.Info("agent using local inference service", "info", info)
       // Continue to set up agent with llmClient below
   } else {
       // existing switch cfg.LLMProvider logic
   }
   ```
6. Add helper method:
   ```go
   func (r *RootCommand) listRunningServices(ctx context.Context) []service.ModelService {
       if r.serviceStore == nil {
           return nil
       }
       svcs, _, err := r.serviceStore.List(ctx, service.ServiceFilter{
           Status: service.ServiceStatusRunning,
       })
       if err != nil {
           return nil
       }
       return svcs
   }
   ```

**Full diff for setupAgent:**

```go
func (r *RootCommand) setupAgent(ctx context.Context) error {
    cfg := r.cfg.Agent

    var llmClient agentllm.LLMClient

    if cfg.LLMAPIKey == "" {
        // No cloud API key — try to find a local inference service.
        svcs := r.listRunningServices(ctx)
        var info string
        var err error
        llmClient, info, err = detectLocalLLM(ctx, svcs, r.cfg.Engine.OllamaAddr)
        if err != nil || llmClient == nil {
            slog.Debug("no API key and no local LLM service detected, agent unavailable")
            return nil
        }
        slog.Info("agent using local inference service", "info", info)
    } else {
        switch cfg.LLMProvider {
        case "anthropic":
            llmClient = agentllm.NewAnthropicClient(cfg.LLMModel, cfg.LLMAPIKey)
        case "ollama":
            llmClient = agentllm.NewOllamaClient(cfg.LLMModel, cfg.LLMBaseURL)
        default:
            baseURL := strings.TrimRight(cfg.LLMBaseURL, "/")
            llmClient = agentllm.NewOpenAIClient(cfg.LLMModel, cfg.LLMAPIKey, baseURL)
        }
    }

    mcpAdapter := gateway.NewMCPAdapter(r.gateway)
    toolExecutor := gateway.NewAgentExecutorAdapter(mcpAdapter)
    agentInstance := coreagent.NewAgent(llmClient, toolExecutor, nil, coreagent.AgentOptions{
        MaxTokens: cfg.MaxTokens,
    })

    if err := registry.RegisterAgentDomain(r.registry, agentInstance); err != nil {
        return fmt.Errorf("register agent domain: %w", err)
    }

    slog.Info("agent operator ready",
        "provider", llmClient.Name(),
        "model", llmClient.ModelName(),
    )
    return nil
}

func (r *RootCommand) listRunningServices(ctx context.Context) []service.ModelService {
    if r.serviceStore == nil {
        return nil
    }
    svcs, _, err := r.serviceStore.List(ctx, service.ServiceFilter{
        Status: service.ServiceStatusRunning,
    })
    if err != nil {
        return nil
    }
    return svcs
}
```

**Step 3: Run all CLI tests**

```bash
go test ./pkg/cli/... -v 2>&1 | tail -40
```
Expected: all tests PASS.

**Step 4: Commit**

```bash
git add pkg/cli/root.go pkg/cli/root_test.go
git commit -m "feat: auto-detect local AIMA inference services as agent LLM fallback"
```

---

### Task 3: Integration verification

**Step 1: Build**

```bash
go build ./... 2>&1
```
Expected: no errors.

**Step 2: Manual smoke test (Ollama)**

If Ollama is running locally at `localhost:11434`:

```bash
# Unset any API key to force fallback path
unset OPENAI_API_KEY
unset AIMA_LLM_API_KEY
./aima agent status
# Expected: enabled: true, model: <first-ollama-model>, provider: ollama
```

**Step 3: Commit (if any additional fixes were needed)**

```bash
git add -A
git commit -m "fix: address any integration issues with local LLM fallback"
```
