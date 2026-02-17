package e2e

import (
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
)

func TestModelLifecycleE2E(t *testing.T) {
	env := SetupTestEnv(t)

	t.Run("create model", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "model.create",
			Input: map[string]any{
				"name":   "test-model",
				"type":   "llm",
				"format": "gguf",
			},
		})
		assertSuccess(t, resp)

		data := getMapField(resp.Data, "")
		if data == nil {
			t.Fatalf("expected data to be map, got: %v", resp.Data)
		}
		modelID := getStringField(data, "model_id")
		if modelID == "" {
			t.Errorf("expected model_id to be non-empty")
		}
	})

	t.Run("create and get model", func(t *testing.T) {
		createResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "model.create",
			Input: map[string]any{
				"name":   "test-model-get",
				"type":   "llm",
				"format": "gguf",
			},
		})
		assertSuccess(t, createResp)

		data := getMapField(createResp.Data, "")
		modelID := getStringField(data, "model_id")
		if modelID == "" {
			t.Fatalf("expected model_id")
		}

		getResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "query",
			Unit: "model.get",
			Input: map[string]any{
				"model_id": modelID,
			},
		})
		assertSuccess(t, getResp)

		modelData := getMapField(getResp.Data, "")
		if modelData == nil {
			t.Fatalf("expected model data")
		}

		name := getStringField(modelData, "name")
		if name != "test-model-get" {
			t.Errorf("expected name to be 'test-model-get', got: %s", name)
		}

		modelType := getStringField(modelData, "type")
		if modelType != "llm" {
			t.Errorf("expected type to be 'llm', got: %s", modelType)
		}
	})

	t.Run("list models", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
				Type: "command",
				Unit: "model.create",
				Input: map[string]any{
					"name":   "list-test-model",
					"type":   "llm",
					"format": "gguf",
				},
			})
			assertSuccess(t, resp)
		}

		listResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type:  "query",
			Unit:  "model.list",
			Input: map[string]any{},
		})
		assertSuccess(t, listResp)

		data := getMapField(listResp.Data, "")
		if data == nil {
			t.Fatalf("expected data to be non-nil, got: %v", listResp.Data)
		}
		total := getIntField(listResp.Data, "total")
		if total < 0 {
			t.Errorf("expected non-negative total, got: %d", total)
		}
	})

	t.Run("delete model", func(t *testing.T) {
		createResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "model.create",
			Input: map[string]any{
				"name":   "delete-test-model",
				"type":   "llm",
				"format": "gguf",
			},
		})
		assertSuccess(t, createResp)

		data := getMapField(createResp.Data, "")
		modelID := getStringField(data, "model_id")

		deleteResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "model.delete",
			Input: map[string]any{
				"model_id": modelID,
			},
		})
		assertSuccess(t, deleteResp)

		deleteData := getMapField(deleteResp.Data, "")
		success := deleteData["success"]
		if success != true {
			t.Errorf("expected success to be true")
		}
	})

	t.Run("get deleted model should fail", func(t *testing.T) {
		createResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "model.create",
			Input: map[string]any{
				"name":   "temp-model",
				"type":   "llm",
				"format": "gguf",
			},
		})
		assertSuccess(t, createResp)

		data := getMapField(createResp.Data, "")
		modelID := getStringField(data, "model_id")

		deleteResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "model.delete",
			Input: map[string]any{
				"model_id": modelID,
			},
		})
		assertSuccess(t, deleteResp)

		getResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "query",
			Unit: "model.get",
			Input: map[string]any{
				"model_id": modelID,
			},
		})
		assertError(t, getResp)
	})

	t.Run("full model lifecycle", func(t *testing.T) {
		createResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "model.create",
			Input: map[string]any{
				"name":   "lifecycle-model",
				"type":   "embedding",
				"format": "safetensors",
			},
		})
		assertSuccess(t, createResp)

		data := getMapField(createResp.Data, "")
		modelID := getStringField(data, "model_id")
		if modelID == "" {
			t.Fatalf("expected model_id")
		}

		getResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "query",
			Unit: "model.get",
			Input: map[string]any{
				"model_id": modelID,
			},
		})
		assertSuccess(t, getResp)

		modelData := getMapField(getResp.Data, "")
		if getStringField(modelData, "name") != "lifecycle-model" {
			t.Errorf("expected name 'lifecycle-model'")
		}
		if getStringField(modelData, "type") != "embedding" {
			t.Errorf("expected type 'embedding'")
		}
		if getStringField(modelData, "format") != "safetensors" {
			t.Errorf("expected format 'safetensors'")
		}

		deleteResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "model.delete",
			Input: map[string]any{
				"model_id": modelID,
			},
		})
		assertSuccess(t, deleteResp)
	})
}
