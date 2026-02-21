//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestScenario2_PullModelAndInfer exercises the full pull-to-inference lifecycle:
// pull a model, verify it, get a service recommendation, create and start a service,
// run inference, inspect the service, then stop and delete it.
func TestScenario2_PullModelAndInfer(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	var modelID, serviceID string

	t.Run("pull_model", func(t *testing.T) {
		resp := env.command(ctx, "model.pull", map[string]any{
			"source": "ollama",
			"repo":   "llama3",
			"tag":    "latest",
		})
		requireSuccess(t, resp, "model.pull")

		data := dataMap(t, resp)
		id, ok := data["model_id"].(string)
		require.True(t, ok, "model_id should be a string")
		require.NotEmpty(t, id, "model_id should not be empty")
		modelID = id
	})

	t.Run("verify_model_ready", func(t *testing.T) {
		require.NotEmpty(t, modelID, "modelID must be set by pull_model")

		resp := env.query(ctx, "model.get", map[string]any{
			"model_id": modelID,
		})
		requireSuccess(t, resp, "model.get")

		data := dataMap(t, resp)
		_, ok := data["status"].(string)
		require.True(t, ok, "data should have a status field")
	})

	t.Run("get_recommendation", func(t *testing.T) {
		require.NotEmpty(t, modelID, "modelID must be set by pull_model")

		resp := env.query(ctx, "service.recommend", map[string]any{
			"model_id": modelID,
		})
		requireSuccess(t, resp, "service.recommend")

		data := dataMap(t, resp)
		_, ok := data["resource_class"].(string)
		require.True(t, ok, "data should have a resource_class field")
	})

	t.Run("create_service", func(t *testing.T) {
		require.NotEmpty(t, modelID, "modelID must be set by pull_model")

		resp := env.command(ctx, "service.create", map[string]any{
			"model_id": modelID,
		})
		requireSuccess(t, resp, "service.create")

		data := dataMap(t, resp)
		id, ok := data["service_id"].(string)
		require.True(t, ok, "service_id should be a string")
		require.NotEmpty(t, id, "service_id should not be empty")
		serviceID = id
	})

	t.Run("start_service", func(t *testing.T) {
		require.NotEmpty(t, serviceID, "serviceID must be set by create_service")

		resp := env.command(ctx, "service.start", map[string]any{
			"service_id": serviceID,
		})
		requireSuccess(t, resp, "service.start")
	})

	t.Run("check_status", func(t *testing.T) {
		require.NotEmpty(t, serviceID, "serviceID must be set by create_service")

		resp := env.query(ctx, "service.status", map[string]any{
			"service_id": serviceID,
		})
		requireSuccess(t, resp, "service.status")
	})

	t.Run("run_inference", func(t *testing.T) {
		resp := env.command(ctx, "inference.chat", map[string]any{
			"model": "llama3",
			"messages": []any{
				map[string]any{"role": "user", "content": "What is AIMA?"},
			},
		})
		requireSuccess(t, resp, "inference.chat")
	})

	t.Run("get_service_info", func(t *testing.T) {
		require.NotEmpty(t, serviceID, "serviceID must be set by create_service")

		resp := env.query(ctx, "service.get", map[string]any{
			"service_id": serviceID,
		})
		requireSuccess(t, resp, "service.get")

		data := dataMap(t, resp)
		require.NotEmpty(t, data, "service.get data should not be empty")
	})

	t.Run("stop_service", func(t *testing.T) {
		require.NotEmpty(t, serviceID, "serviceID must be set by create_service")

		resp := env.command(ctx, "service.stop", map[string]any{
			"service_id": serviceID,
		})
		requireSuccess(t, resp, "service.stop")
	})

	t.Run("delete_service", func(t *testing.T) {
		require.NotEmpty(t, serviceID, "serviceID must be set by create_service")

		resp := env.command(ctx, "service.delete", map[string]any{
			"service_id": serviceID,
		})
		requireSuccess(t, resp, "service.delete")
	})
}
