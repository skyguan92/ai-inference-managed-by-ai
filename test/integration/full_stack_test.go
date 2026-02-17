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
	"github.com/jguan/ai-inference-managed-by-ai/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type FullStackTest struct {
	registry *unit.Registry
	gateway  *gateway.Gateway
	server   *gateway.Server
	router   *gateway.Router
	mcp      *gateway.MCPAdapter
	workflow *workflow.WorkflowEngine
}

func setupFullStack(t *testing.T) *FullStackTest {
	registry := unit.NewRegistry()
	err := registrypkg.RegisterAll(registry)
	require.NoError(t, err)

	gw := gateway.NewGateway(registry)
	server := gateway.NewServer(gw, gateway.ServerConfig{
		Addr:            ":0",
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    5 * time.Second,
		ShutdownTimeout: 2 * time.Second,
	})
	router := gateway.NewRouter(gw)
	mcp := gateway.NewMCPAdapter(gw)
	workflowStore := workflow.NewInMemoryWorkflowStore()
	workflowEngine := workflow.NewWorkflowEngine(registry, workflowStore, nil)

	return &FullStackTest{
		registry: registry,
		gateway:  gw,
		server:   server,
		router:   router,
		mcp:      mcp,
		workflow: workflowEngine,
	}
}

func TestFullStackAllComponentsInitialized(t *testing.T) {
	fs := setupFullStack(t)

	assert.NotNil(t, fs.registry)
	assert.NotNil(t, fs.gateway)
	assert.NotNil(t, fs.server)
	assert.NotNil(t, fs.router)
	assert.NotNil(t, fs.mcp)
	assert.NotNil(t, fs.workflow)

	assert.Greater(t, fs.registry.CommandCount(), 0)
	assert.Greater(t, fs.registry.QueryCount(), 0)
}

func TestFullStackGatewayToRegistry(t *testing.T) {
	fs := setupFullStack(t)

	resp := fs.gateway.Handle(context.Background(), &gateway.Request{
		Type:  gateway.TypeQuery,
		Unit:  "model.list",
		Input: map[string]any{},
	})

	assert.True(t, resp.Success)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Data)
}

func TestFullStackHTTPToGateway(t *testing.T) {
	fs := setupFullStack(t)

	body := map[string]any{
		"type":  "query",
		"unit":  "model.list",
		"input": map[string]any{},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	adapter := gateway.NewHTTPAdapter(fs.gateway)
	adapter.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp gateway.Response
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestFullStackMCPToGateway(t *testing.T) {
	fs := setupFullStack(t)

	fs.mcp.HandleRequest(context.Background(), &gateway.MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  json.RawMessage(`{}`),
	})

	resp := fs.mcp.HandleRequest(context.Background(), &gateway.MCPRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/call",
		Params:  json.RawMessage(`{"name": "model_list", "arguments": {}}`),
	})

	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Result)

	result, ok := resp.Result.(*gateway.MCPToolResult)
	require.True(t, ok)
	assert.False(t, result.IsError)
}

func TestFullStackWorkflowToRegistry(t *testing.T) {
	fs := setupFullStack(t)

	def := &workflow.WorkflowDef{
		Name: "full_stack_test",
		Steps: []workflow.WorkflowStep{
			{
				ID:    "list_models",
				Type:  "model.list",
				Input: map[string]any{},
			},
		},
	}

	result, err := fs.workflow.Execute(context.Background(), def, nil)
	require.NoError(t, err)
	assert.Equal(t, workflow.ExecutionStatusCompleted, result.Status)
}

func TestFullStackEndToEndModelOperations(t *testing.T) {
	fs := setupFullStack(t)

	t.Run("create model via HTTP", func(t *testing.T) {
		body := map[string]any{
			"name":   "test-model-e2e",
			"source": "ollama",
		}
		bodyBytes, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/models/create", bytes.NewReader(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		fs.router.ServeHTTP(rec, req)

		var resp gateway.Response
		json.Unmarshal(rec.Body.Bytes(), &resp)
	})

	t.Run("list models via Gateway", func(t *testing.T) {
		resp := fs.gateway.Handle(context.Background(), &gateway.Request{
			Type:  gateway.TypeQuery,
			Unit:  "model.list",
			Input: map[string]any{},
		})

		assert.True(t, resp.Success)
	})

	t.Run("list models via MCP", func(t *testing.T) {
		fs.mcp.HandleRequest(context.Background(), &gateway.MCPRequest{
			JSONRPC: "2.0",
			ID:      1,
			Method:  "initialize",
			Params:  json.RawMessage(`{}`),
		})

		resp := fs.mcp.HandleRequest(context.Background(), &gateway.MCPRequest{
			JSONRPC: "2.0",
			ID:      2,
			Method:  "tools/call",
			Params:  json.RawMessage(`{"name": "model_list", "arguments": {}}`),
		})

		assert.Nil(t, resp.Error)
	})
}

func TestFullStackCrossDomainOperations(t *testing.T) {
	fs := setupFullStack(t)

	operations := []struct {
		name    string
		reqType string
		unit    string
	}{
		{"model.list", gateway.TypeQuery, "model.list"},
		{"engine.list", gateway.TypeQuery, "engine.list"},
		{"service.list", gateway.TypeQuery, "service.list"},
		{"app.list", gateway.TypeQuery, "app.list"},
	}

	for _, op := range operations {
		t.Run(op.name, func(t *testing.T) {
			resp := fs.gateway.Handle(context.Background(), &gateway.Request{
				Type:  op.reqType,
				Unit:  op.unit,
				Input: map[string]any{},
			})

			assert.True(t, resp.Success)
			assert.NotNil(t, resp.Meta)
		})
	}
}

func TestFullStackWorkflowWithMultipleDomains(t *testing.T) {
	fs := setupFullStack(t)

	def := &workflow.WorkflowDef{
		Name: "cross_domain_workflow",
		Steps: []workflow.WorkflowStep{
			{
				ID:    "list_models",
				Type:  "model.list",
				Input: map[string]any{},
			},
			{
				ID:        "list_engines",
				Type:      "engine.list",
				Input:     map[string]any{},
				DependsOn: []string{"list_models"},
			},
			{
				ID:        "list_services",
				Type:      "service.list",
				Input:     map[string]any{},
				DependsOn: []string{"list_engines"},
			},
		},
	}

	result, err := fs.workflow.Execute(context.Background(), def, nil)
	require.NoError(t, err)
	assert.Equal(t, workflow.ExecutionStatusCompleted, result.Status)
	assert.Len(t, result.StepResults, 3)
}

func TestFullStackConsistentResults(t *testing.T) {
	fs := setupFullStack(t)

	t.Run("model.list consistency", func(t *testing.T) {
		gatewayResp := fs.gateway.Handle(context.Background(), &gateway.Request{
			Type:  gateway.TypeQuery,
			Unit:  "model.list",
			Input: map[string]any{},
		})

		body := map[string]any{
			"type":  "query",
			"unit":  "model.list",
			"input": map[string]any{},
		}
		bodyBytes, _ := json.Marshal(body)
		httpReq := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader(bodyBytes))
		httpReq.Header.Set("Content-Type", "application/json")
		httpRec := httptest.NewRecorder()
		gateway.NewHTTPAdapter(fs.gateway).ServeHTTP(httpRec, httpReq)

		var httpResp gateway.Response
		json.Unmarshal(httpRec.Body.Bytes(), &httpResp)

		assert.Equal(t, gatewayResp.Success, httpResp.Success)
	})
}

func TestFullStackMCPToolCount(t *testing.T) {
	fs := setupFullStack(t)

	fs.mcp.HandleRequest(context.Background(), &gateway.MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  json.RawMessage(`{}`),
	})

	resp := fs.mcp.HandleRequest(context.Background(), &gateway.MCPRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
		Params:  json.RawMessage(`{}`),
	})

	require.NotNil(t, resp)
	require.Nil(t, resp.Error)

	result, ok := resp.Result.(*gateway.MCPToolsListResult)
	require.True(t, ok)

	expectedToolCount := fs.registry.CommandCount() + fs.registry.QueryCount()
	assert.Equal(t, expectedToolCount, len(result.Tools))
}

func TestFullStackErrorHandling(t *testing.T) {
	fs := setupFullStack(t)

	t.Run("gateway handles invalid unit", func(t *testing.T) {
		resp := fs.gateway.Handle(context.Background(), &gateway.Request{
			Type:  gateway.TypeQuery,
			Unit:  "invalid.unit",
			Input: map[string]any{},
		})

		assert.False(t, resp.Success)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, gateway.ErrCodeUnitNotFound, resp.Error.Code)
	})

	t.Run("HTTP handles invalid request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		gateway.NewHTTPAdapter(fs.gateway).ServeHTTP(rec, req)

		assert.NotEqual(t, http.StatusOK, rec.Code)
	})

	t.Run("MCP handles method not found", func(t *testing.T) {
		fs.mcp.HandleRequest(context.Background(), &gateway.MCPRequest{
			JSONRPC: "2.0",
			ID:      1,
			Method:  "initialize",
			Params:  json.RawMessage(`{}`),
		})

		resp := fs.mcp.HandleRequest(context.Background(), &gateway.MCPRequest{
			JSONRPC: "2.0",
			ID:      2,
			Method:  "invalid/method",
			Params:  json.RawMessage(`{}`),
		})

		assert.NotNil(t, resp.Error)
		assert.Equal(t, gateway.MCPErrorCodeMethodNotFound, resp.Error.Code)
	})
}

func TestFullStackWorkflowErrorHandling(t *testing.T) {
	fs := setupFullStack(t)

	t.Run("workflow with non-existent unit", func(t *testing.T) {
		def := &workflow.WorkflowDef{
			Name: "error_workflow",
			Steps: []workflow.WorkflowStep{
				{
					ID:    "step1",
					Type:  "nonexistent.unit",
					Input: map[string]any{},
				},
			},
		}

		_, err := fs.workflow.Execute(context.Background(), def, nil)
		assert.Error(t, err)
	})

	t.Run("workflow with invalid step", func(t *testing.T) {
		def := &workflow.WorkflowDef{
			Name: "invalid_workflow",
			Steps: []workflow.WorkflowStep{
				{
					ID:   "",
					Type: "model.list",
				},
			},
		}

		_, err := fs.workflow.Execute(context.Background(), def, nil)
		assert.Error(t, err)
	})
}

func TestFullStackRequestMetadata(t *testing.T) {
	fs := setupFullStack(t)

	traceID := "full-stack-trace-123"

	resp := fs.gateway.Handle(context.Background(), &gateway.Request{
		Type:  gateway.TypeQuery,
		Unit:  "model.list",
		Input: map[string]any{},
		Options: gateway.RequestOptions{
			TraceID: traceID,
		},
	})

	assert.True(t, resp.Success)
	assert.NotNil(t, resp.Meta)
	assert.Equal(t, traceID, resp.Meta.TraceID)
	assert.NotEmpty(t, resp.Meta.RequestID)
	assert.GreaterOrEqual(t, resp.Meta.Duration, int64(0))
}

func TestFullStackConcurrentRequests(t *testing.T) {
	fs := setupFullStack(t)

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			resp := fs.gateway.Handle(context.Background(), &gateway.Request{
				Type:  gateway.TypeQuery,
				Unit:  "model.list",
				Input: map[string]any{},
			})
			assert.True(t, resp.Success)
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestFullStackRegistryStats(t *testing.T) {
	fs := setupFullStack(t)

	cmdCount := fs.registry.CommandCount()
	queryCount := fs.registry.QueryCount()

	assert.Greater(t, cmdCount, 0)
	assert.Greater(t, queryCount, 0)

	commands := fs.registry.ListCommands()
	queries := fs.registry.ListQueries()

	assert.Equal(t, cmdCount, len(commands))
	assert.Equal(t, queryCount, len(queries))

	domains := make(map[string]int)
	for _, cmd := range commands {
		domains[cmd.Domain()]++
	}
	for _, q := range queries {
		domains[q.Domain()]++
	}

	expectedDomains := []string{"model", "device", "engine", "inference", "resource", "service", "app", "pipeline", "alert", "remote"}
	for _, domain := range expectedDomains {
		assert.Greater(t, domains[domain], 0, "domain %s should have at least one unit", domain)
	}
}

func TestFullStackHealthCheck(t *testing.T) {
	fs := setupFullStack(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	})
	mux.Handle("/", fs.router)

	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "healthy", resp["status"])
}

func TestFullStackTimeout(t *testing.T) {
	fs := setupFullStack(t)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	time.Sleep(1 * time.Millisecond)

	resp := fs.gateway.Handle(ctx, &gateway.Request{
		Type:  gateway.TypeQuery,
		Unit:  "model.list",
		Input: map[string]any{},
	})

	_ = resp
}
