//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/model"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRegression_ServiceIDParsing verifies that services with model IDs
// containing dashes and special characters work correctly through the full
// create → start → stop lifecycle (regression for Bug #11).
func TestRegression_ServiceIDParsing(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	models := []struct {
		id        string
		name      string
		modelType model.ModelType
		path      string
	}{
		{"model-qwen3-coder", "qwen3-coder-next-fp8", model.ModelTypeLLM, "/mnt/data/models/qwen3-coder-next-fp8"},
		{"model-llama-3.1", "llama-3.1-70b", model.ModelTypeLLM, "/mnt/data/models/llama-3.1-70b"},
		{"model-deepseek-v2.5", "deepseek-v2.5", model.ModelTypeLLM, "/mnt/data/models/deepseek-v2.5"},
	}

	// Seed models into the store.
	for _, m := range models {
		env.seedModel(t, m.id, m.name, m.modelType, m.path)
	}

	// Create a service for each model, then start and stop it.
	for _, m := range models {
		t.Run("create_start_stop/"+m.name, func(t *testing.T) {
			// Create service
			resp := env.command(ctx, "service.create", map[string]any{
				"model_id": m.id,
			})
			requireSuccess(t, resp, "service.create for "+m.name)
			data := dataMap(t, resp)
			serviceID, ok := data["service_id"].(string)
			require.True(t, ok, "service_id should be a string")
			require.NotEmpty(t, serviceID, "service_id should not be empty")

			// Seed the service into the store so start/stop can find it.
			// The create command already stores it, but the mock provider
			// returns status=running directly; we need to set it to stopped
			// so that start can proceed without "already running" guard.
			svc, err := env.ServiceStore.Get(ctx, serviceID)
			require.NoError(t, err)
			svc.Status = service.ServiceStatusStopped
			require.NoError(t, env.ServiceStore.Update(ctx, svc))

			// Start service
			resp = env.command(ctx, "service.start", map[string]any{
				"service_id": serviceID,
			})
			requireSuccess(t, resp, "service.start for "+m.name)

			// Stop service
			resp = env.command(ctx, "service.stop", map[string]any{
				"service_id": serviceID,
			})
			requireSuccess(t, resp, "service.stop for "+m.name)
		})
	}
}

// TestRegression_ContextCancellation verifies that cancelling a context
// mid-request does not cause a panic.
func TestRegression_ContextCancellation(t *testing.T) {
	env := newTestEnv(t)

	// Create a context and cancel it immediately.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Send a query with the cancelled context. The response may succeed
	// (if the handler completes before checking context) or fail, but
	// the critical thing is that no panic occurs.
	require.NotPanics(t, func() {
		resp := env.query(ctx, "model.list", nil)
		// We only check that a response was returned (not nil).
		require.NotNil(t, resp, "response should not be nil even with cancelled context")
	})
}

// TestRegression_InvalidRequestType verifies that the gateway correctly
// rejects requests with invalid or empty types.
func TestRegression_InvalidRequestType(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	t.Run("invalid_type", func(t *testing.T) {
		resp := env.Gateway.Handle(ctx, &gateway.Request{
			Type: "invalid",
			Unit: "model.list",
		})
		require.NotNil(t, resp)
		assert.False(t, resp.Success, "request with invalid type should fail")
		require.NotNil(t, resp.Error)
		assert.Equal(t, "INVALID_REQUEST", resp.Error.Code)
	})

	t.Run("empty_type", func(t *testing.T) {
		resp := env.Gateway.Handle(ctx, &gateway.Request{
			Type: "",
			Unit: "model.list",
		})
		require.NotNil(t, resp)
		assert.False(t, resp.Success, "request with empty type should fail")
		require.NotNil(t, resp.Error)
		assert.Equal(t, "INVALID_REQUEST", resp.Error.Code)
	})

	t.Run("nil_request", func(t *testing.T) {
		resp := env.Gateway.Handle(ctx, nil)
		require.NotNil(t, resp)
		assert.False(t, resp.Success, "nil request should fail")
		require.NotNil(t, resp.Error)
		assert.Equal(t, "INVALID_REQUEST", resp.Error.Code)
	})
}

// TestRegression_EmptyInputHandling verifies that each major command returns
// a meaningful error (not a panic) when called with missing required fields.
func TestRegression_EmptyInputHandling(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	tests := []struct {
		name string
		unit string
	}{
		{"model.create with empty input", "model.create"},
		{"model.delete with empty input", "model.delete"},
		{"service.create with empty input", "service.create"},
		{"service.start with empty input", "service.start"},
		{"inference.chat with empty input", "inference.chat"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp := env.command(ctx, tc.unit, map[string]any{})
			requireFailure(t, resp, tc.name)
			require.NotNil(t, resp.Error, "error should not be nil for %s", tc.name)
		})
	}
}

// TestRegression_UnitNotFound verifies that requesting nonexistent units
// returns UNIT_NOT_FOUND.
func TestRegression_UnitNotFound(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	t.Run("nonexistent_command", func(t *testing.T) {
		resp := env.command(ctx, "nonexistent.command", map[string]any{})
		requireFailure(t, resp, "nonexistent command")
		require.NotNil(t, resp.Error)
		assert.Equal(t, "UNIT_NOT_FOUND", resp.Error.Code)
	})

	t.Run("nonexistent_query", func(t *testing.T) {
		resp := env.query(ctx, "nonexistent.query", map[string]any{})
		requireFailure(t, resp, "nonexistent query")
		require.NotNil(t, resp.Error)
		assert.Equal(t, "UNIT_NOT_FOUND", resp.Error.Code)
	})
}
