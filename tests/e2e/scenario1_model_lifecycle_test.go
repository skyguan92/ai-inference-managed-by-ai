//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScenario1_ModelScanAndInference(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	var modelID string
	var serviceID string

	t.Run("list_models_empty", func(t *testing.T) {
		resp := env.query(ctx, "model.list", map[string]any{})
		requireSuccess(t, resp, "model.list should succeed on empty store")
		data := dataMap(t, resp)
		_, ok := data["items"]
		require.True(t, ok, "response should have 'items' key")
	})

	t.Run("create_model", func(t *testing.T) {
		resp := env.command(ctx, "model.create", map[string]any{
			"name": "test-llm",
			"type": "llm",
			"path": "/models/test",
		})
		requireSuccess(t, resp, "model.create should succeed")
		data := dataMap(t, resp)
		id, ok := data["model_id"].(string)
		require.True(t, ok, "model_id should be a string")
		require.NotEmpty(t, id, "model_id should not be empty")
		modelID = id
	})

	t.Run("get_model", func(t *testing.T) {
		require.NotEmpty(t, modelID, "modelID must be set by create_model")
		resp := env.query(ctx, "model.get", map[string]any{
			"model_id": modelID,
		})
		requireSuccess(t, resp, "model.get should succeed")
		data := dataMap(t, resp)
		require.Equal(t, "test-llm", data["name"], "name should match")
		require.Equal(t, "llm", data["type"], "type should match")
	})

	t.Run("list_models_has_one", func(t *testing.T) {
		resp := env.query(ctx, "model.list", map[string]any{})
		requireSuccess(t, resp, "model.list should succeed")
		data := dataMap(t, resp)
		items, ok := data["items"].([]map[string]any)
		require.True(t, ok, "items should be []map[string]any")
		require.Len(t, items, 1, "should have exactly one model")
	})

	t.Run("create_service", func(t *testing.T) {
		require.NotEmpty(t, modelID, "modelID must be set by create_model")
		resp := env.command(ctx, "service.create", map[string]any{
			"model_id": modelID,
		})
		requireSuccess(t, resp, "service.create should succeed")
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
		requireSuccess(t, resp, "service.start should succeed")
	})

	t.Run("check_service_status", func(t *testing.T) {
		require.NotEmpty(t, serviceID, "serviceID must be set by create_service")
		resp := env.query(ctx, "service.status", map[string]any{
			"service_id": serviceID,
		})
		requireSuccess(t, resp, "service.status should succeed")
	})

	t.Run("run_inference", func(t *testing.T) {
		resp := env.command(ctx, "inference.chat", map[string]any{
			"model": "test-model",
			"messages": []any{
				map[string]any{"role": "user", "content": "hello"},
			},
		})
		requireSuccess(t, resp, "inference.chat should succeed")
	})

	t.Run("stop_service", func(t *testing.T) {
		require.NotEmpty(t, serviceID, "serviceID must be set by create_service")
		resp := env.command(ctx, "service.stop", map[string]any{
			"service_id": serviceID,
		})
		requireSuccess(t, resp, "service.stop should succeed")
	})

	t.Run("delete_model", func(t *testing.T) {
		require.NotEmpty(t, modelID, "modelID must be set by create_model")
		resp := env.command(ctx, "model.delete", map[string]any{
			"model_id": modelID,
		})
		requireSuccess(t, resp, "model.delete should succeed")
	})

	t.Run("verify_model_gone", func(t *testing.T) {
		require.NotEmpty(t, modelID, "modelID must be set by create_model")
		resp := env.query(ctx, "model.get", map[string]any{
			"model_id": modelID,
		})
		requireFailure(t, resp, "model.get should fail for deleted model")
	})
}
