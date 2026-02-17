package e2e

import (
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
)

func TestEngineLifecycleE2E(t *testing.T) {
	env := SetupTestEnv(t)

	t.Run("list engines initially empty", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type:  "query",
			Unit:  "engine.list",
			Input: map[string]any{},
		})
		assertSuccess(t, resp)

		data := getMapField(resp.Data, "")
		items := getSliceField(data, "items")
		total := getIntField(data, "total")
		if total != len(items) {
			t.Errorf("expected total %d to match items count %d", total, len(items))
		}
	})

	t.Run("install engine", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "engine.install",
			Input: map[string]any{
				"name":    "ollama",
				"version": "0.1.26",
			},
		})
		assertSuccess(t, resp)

		data := getMapField(resp.Data, "")
		success := data["success"]
		if success != true {
			t.Errorf("expected success to be true")
		}
	})

	t.Run("list engines after install", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type:  "query",
			Unit:  "engine.list",
			Input: map[string]any{},
		})
		assertSuccess(t, resp)

		data := getMapField(resp.Data, "")
		total := getIntField(data, "total")
		if total < 1 {
			t.Errorf("expected at least 1 engine after install, got: %d", total)
		}
	})

	t.Run("install and get engine", func(t *testing.T) {
		installResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "engine.install",
			Input: map[string]any{
				"name": "vllm",
			},
		})
		assertSuccess(t, installResp)

		getResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "query",
			Unit: "engine.get",
			Input: map[string]any{
				"name": "vllm",
			},
		})
		assertSuccess(t, getResp)

		data := getMapField(getResp.Data, "")
		name := getStringField(data, "name")
		if name != "vllm" {
			t.Errorf("expected name to be 'vllm', got: %s", name)
		}

		status := getStringField(data, "status")
		if status != "stopped" {
			t.Errorf("expected status to be 'stopped', got: %s", status)
		}
	})

	t.Run("install and start engine", func(t *testing.T) {
		installResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "engine.install",
			Input: map[string]any{
				"name": "test-start-engine",
			},
		})
		assertSuccess(t, installResp)

		startResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "engine.start",
			Input: map[string]any{
				"name": "test-start-engine",
			},
		})
		assertSuccess(t, startResp)

		data := getMapField(startResp.Data, "")
		status := getStringField(data, "status")
		if status != "running" {
			t.Errorf("expected status to be 'running', got: %s", status)
		}

		processID := getStringField(data, "process_id")
		if processID == "" {
			t.Errorf("expected process_id to be non-empty")
		}
	})

	t.Run("start then stop engine", func(t *testing.T) {
		installResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "engine.install",
			Input: map[string]any{
				"name": "test-stop-engine",
			},
		})
		assertSuccess(t, installResp)

		startResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "engine.start",
			Input: map[string]any{
				"name": "test-stop-engine",
			},
		})
		assertSuccess(t, startResp)

		getResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "query",
			Unit: "engine.get",
			Input: map[string]any{
				"name": "test-stop-engine",
			},
		})
		assertSuccess(t, getResp)
		data := getMapField(getResp.Data, "")
		if getStringField(data, "status") != "running" {
			t.Errorf("expected engine to be running before stop")
		}

		stopResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "engine.stop",
			Input: map[string]any{
				"name": "test-stop-engine",
			},
		})
		assertSuccess(t, stopResp)

		getAfterResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "query",
			Unit: "engine.get",
			Input: map[string]any{
				"name": "test-stop-engine",
			},
		})
		assertSuccess(t, getAfterResp)
		afterData := getMapField(getAfterResp.Data, "")
		if getStringField(afterData, "status") != "stopped" {
			t.Errorf("expected engine to be stopped after stop, got: %s", getStringField(afterData, "status"))
		}
	})

	t.Run("restart engine", func(t *testing.T) {
		installResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "engine.install",
			Input: map[string]any{
				"name": "test-restart-engine",
			},
		})
		assertSuccess(t, installResp)

		startResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "engine.start",
			Input: map[string]any{
				"name": "test-restart-engine",
			},
		})
		assertSuccess(t, startResp)

		restartResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "engine.restart",
			Input: map[string]any{
				"name": "test-restart-engine",
			},
		})
		assertSuccess(t, restartResp)

		data := getMapField(restartResp.Data, "")
		status := getStringField(data, "status")
		if status != "running" {
			t.Errorf("expected status to be 'running' after restart, got: %s", status)
		}
	})

	t.Run("get engine features", func(t *testing.T) {
		installResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "engine.install",
			Input: map[string]any{
				"name": "test-features-engine",
			},
		})
		assertSuccess(t, installResp)

		featuresResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "query",
			Unit: "engine.features",
			Input: map[string]any{
				"name": "test-features-engine",
			},
		})
		assertSuccess(t, featuresResp)

		data := getMapField(featuresResp.Data, "")
		supportsStreaming, ok := data["supports_streaming"].(bool)
		if !ok || !supportsStreaming {
			t.Errorf("expected supports_streaming to be true")
		}
	})

	t.Run("get non-existent engine should fail", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "query",
			Unit: "engine.get",
			Input: map[string]any{
				"name": "non-existent-engine",
			},
		})
		assertError(t, resp)
	})

	t.Run("start non-existent engine should fail", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "engine.start",
			Input: map[string]any{
				"name": "non-existent-engine",
			},
		})
		assertError(t, resp)
	})

	t.Run("filter engines by status", func(t *testing.T) {
		installResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "engine.install",
			Input: map[string]any{
				"name": "filter-test-engine",
			},
		})
		assertSuccess(t, installResp)

		stoppedResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "query",
			Unit: "engine.list",
			Input: map[string]any{
				"status": "stopped",
			},
		})
		assertSuccess(t, stoppedResp)

		data := getMapField(stoppedResp.Data, "")
		if data == nil {
			t.Fatalf("expected data to be non-nil, got: %v", stoppedResp.Data)
		}
		total := getIntField(stoppedResp.Data, "total")
		if total < 0 {
			t.Errorf("expected non-negative total, got: %d", total)
		}
	})
}
