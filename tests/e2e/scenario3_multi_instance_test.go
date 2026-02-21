//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScenario3_MultiInstanceLifecycle(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	var modelIDs []string
	var serviceIDs []string

	// Step 1: Create three models with different types.
	t.Run("create_three_models", func(t *testing.T) {
		models := []map[string]any{
			{"name": "test-llm", "type": "llm"},
			{"name": "test-asr", "type": "asr"},
			{"name": "test-tts", "type": "tts"},
		}

		for _, m := range models {
			resp := env.command(ctx, "model.create", m)
			requireSuccess(t, resp, "model.create for "+m["name"].(string))
			data := dataMap(t, resp)
			modelID, ok := data["model_id"].(string)
			require.True(t, ok, "model_id should be a string")
			require.NotEmpty(t, modelID, "model_id should not be empty")
			modelIDs = append(modelIDs, modelID)
		}

		// All model IDs must be unique.
		require.Len(t, modelIDs, 3)
		assert.NotEqual(t, modelIDs[0], modelIDs[1], "model IDs 0 and 1 should differ")
		assert.NotEqual(t, modelIDs[1], modelIDs[2], "model IDs 1 and 2 should differ")
		assert.NotEqual(t, modelIDs[0], modelIDs[2], "model IDs 0 and 2 should differ")
	})

	// Step 2: Create three services, one per model.
	t.Run("create_three_services", func(t *testing.T) {
		require.Len(t, modelIDs, 3, "need 3 model IDs from previous step")

		for _, modelID := range modelIDs {
			resp := env.command(ctx, "service.create", map[string]any{
				"model_id": modelID,
			})
			requireSuccess(t, resp, "service.create for model "+modelID)
			data := dataMap(t, resp)
			serviceID, ok := data["service_id"].(string)
			require.True(t, ok, "service_id should be a string")
			require.NotEmpty(t, serviceID, "service_id should not be empty")
			serviceIDs = append(serviceIDs, serviceID)
		}

		// All service IDs must be unique.
		require.Len(t, serviceIDs, 3)
		assert.NotEqual(t, serviceIDs[0], serviceIDs[1], "service IDs 0 and 1 should differ")
		assert.NotEqual(t, serviceIDs[1], serviceIDs[2], "service IDs 1 and 2 should differ")
		assert.NotEqual(t, serviceIDs[0], serviceIDs[2], "service IDs 0 and 2 should differ")
	})

	// Step 3: Start all three services.
	t.Run("start_all_services", func(t *testing.T) {
		require.Len(t, serviceIDs, 3, "need 3 service IDs from previous step")

		for _, serviceID := range serviceIDs {
			resp := env.command(ctx, "service.start", map[string]any{
				"service_id": serviceID,
			})
			requireSuccess(t, resp, "service.start for "+serviceID)
		}
	})

	// Step 4: List services and verify data exists.
	t.Run("list_services", func(t *testing.T) {
		resp := env.query(ctx, "service.list", map[string]any{})
		requireSuccess(t, resp, "service.list")
		data := dataMap(t, resp)
		require.Contains(t, data, "services", "response should contain services key")
		require.Contains(t, data, "total", "response should contain total key")
	})

	// Step 5: Run three concurrent inference.chat commands.
	t.Run("concurrent_inference", func(t *testing.T) {
		require.Len(t, modelIDs, 3, "need 3 model IDs from previous step")

		inputs := []map[string]any{
			{
				"model":    "test-llm",
				"messages": []any{map[string]any{"role": "user", "content": "hello from LLM"}},
			},
			{
				"model":    "test-asr",
				"messages": []any{map[string]any{"role": "user", "content": "hello from ASR"}},
			},
			{
				"model":    "test-tts",
				"messages": []any{map[string]any{"role": "user", "content": "hello from TTS"}},
			},
		}

		var wg sync.WaitGroup
		errors := make([]error, 3)
		responses := make([]*bool, 3)

		for i, input := range inputs {
			wg.Add(1)
			go func(idx int, in map[string]any) {
				defer wg.Done()
				resp := env.command(ctx, "inference.chat", in)
				if resp.Success {
					success := true
					responses[idx] = &success
				} else {
					errMsg := "unknown error"
					if resp.Error != nil {
						errMsg = resp.Error.Message
					}
					errors[idx] = fmt.Errorf("inference.chat failed for index %d: %s", idx, errMsg)
				}
			}(i, input)
		}

		wg.Wait()

		for i := 0; i < 3; i++ {
			assert.NoError(t, errors[i], "inference.chat[%d] should succeed", i)
			assert.NotNil(t, responses[i], "inference.chat[%d] should return success", i)
		}
	})

	// Step 6: Scale one service to 3 replicas.
	t.Run("scale_one_service", func(t *testing.T) {
		require.Len(t, serviceIDs, 3, "need 3 service IDs from previous step")

		resp := env.command(ctx, "service.scale", map[string]any{
			"service_id": serviceIDs[0],
			"replicas":   3,
		})
		requireSuccess(t, resp, "service.scale")
	})

	// Step 7: Stop all three services.
	t.Run("stop_all_services", func(t *testing.T) {
		require.Len(t, serviceIDs, 3, "need 3 service IDs from previous step")

		for _, serviceID := range serviceIDs {
			resp := env.command(ctx, "service.stop", map[string]any{
				"service_id": serviceID,
			})
			requireSuccess(t, resp, "service.stop for "+serviceID)
		}
	})

	// Step 8: Delete all services and models.
	t.Run("delete_all", func(t *testing.T) {
		require.Len(t, serviceIDs, 3, "need 3 service IDs")
		require.Len(t, modelIDs, 3, "need 3 model IDs")

		// Delete all services first.
		for _, serviceID := range serviceIDs {
			resp := env.command(ctx, "service.delete", map[string]any{
				"service_id": serviceID,
			})
			requireSuccess(t, resp, "service.delete for "+serviceID)
		}

		// Then delete all models.
		for _, modelID := range modelIDs {
			resp := env.command(ctx, "model.delete", map[string]any{
				"model_id": modelID,
			})
			requireSuccess(t, resp, "model.delete for "+modelID)
		}
	})
}
