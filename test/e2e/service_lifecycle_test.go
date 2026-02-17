package e2e

import (
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
)

func TestServiceLifecycleE2E(t *testing.T) {
	env := SetupTestEnv(t)

	t.Run("create service", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "service.create",
			Input: map[string]any{
				"model_id":       "test-model-llm",
				"resource_class": "medium",
				"replicas":       1,
			},
		})
		assertSuccess(t, resp)

		data := getMapField(resp.Data, "")
		serviceID := getStringField(data, "service_id")
		if serviceID == "" {
			t.Errorf("expected service_id to be non-empty")
		}
	})

	t.Run("create and get service", func(t *testing.T) {
		createResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "service.create",
			Input: map[string]any{
				"model_id":       "test-model-get",
				"resource_class": "large",
				"replicas":       2,
			},
		})
		assertSuccess(t, createResp)

		data := getMapField(createResp.Data, "")
		serviceID := getStringField(data, "service_id")

		getResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "query",
			Unit: "service.get",
			Input: map[string]any{
				"service_id": serviceID,
			},
		})
		assertSuccess(t, getResp)

		serviceData := getMapField(getResp.Data, "")
		if getStringField(serviceData, "model_id") != "test-model-get" {
			t.Errorf("expected model_id 'test-model-get'")
		}
		if getIntField(serviceData, "replicas") != 2 {
			t.Errorf("expected replicas 2, got: %d", getIntField(serviceData, "replicas"))
		}
	})

	t.Run("list services", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
				Type: "command",
				Unit: "service.create",
				Input: map[string]any{
					"model_id": "list-test-model",
					"replicas": 1,
				},
			})
			assertSuccess(t, resp)
		}

		listResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type:  "query",
			Unit:  "service.list",
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

	t.Run("start service", func(t *testing.T) {
		createResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "service.create",
			Input: map[string]any{
				"model_id": "start-test-model",
				"replicas": 1,
			},
		})
		assertSuccess(t, createResp)

		data := getMapField(createResp.Data, "")
		serviceID := getStringField(data, "service_id")

		startResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "service.start",
			Input: map[string]any{
				"service_id": serviceID,
			},
		})
		assertSuccess(t, startResp)

		successData := getMapField(startResp.Data, "")
		if successData["success"] != true {
			t.Errorf("expected success to be true")
		}
	})

	t.Run("scale service", func(t *testing.T) {
		createResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "service.create",
			Input: map[string]any{
				"model_id": "scale-test-model",
				"replicas": 1,
			},
		})
		assertSuccess(t, createResp)

		data := getMapField(createResp.Data, "")
		serviceID := getStringField(data, "service_id")

		scaleResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "service.scale",
			Input: map[string]any{
				"service_id": serviceID,
				"replicas":   5,
			},
		})
		assertSuccess(t, scaleResp)

		getResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "query",
			Unit: "service.get",
			Input: map[string]any{
				"service_id": serviceID,
			},
		})
		assertSuccess(t, getResp)

		serviceData := getMapField(getResp.Data, "")
		if getIntField(serviceData, "replicas") != 5 {
			t.Errorf("expected replicas to be 5 after scale, got: %d", getIntField(serviceData, "replicas"))
		}
	})

	t.Run("stop service", func(t *testing.T) {
		createResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "service.create",
			Input: map[string]any{
				"model_id": "stop-test-model",
				"replicas": 1,
			},
		})
		assertSuccess(t, createResp)

		data := getMapField(createResp.Data, "")
		serviceID := getStringField(data, "service_id")

		startResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "service.start",
			Input: map[string]any{
				"service_id": serviceID,
			},
		})
		assertSuccess(t, startResp)

		stopResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "service.stop",
			Input: map[string]any{
				"service_id": serviceID,
			},
		})
		assertSuccess(t, stopResp)

		getResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "query",
			Unit: "service.get",
			Input: map[string]any{
				"service_id": serviceID,
			},
		})
		assertSuccess(t, getResp)

		serviceData := getMapField(getResp.Data, "")
		if getStringField(serviceData, "status") != "stopped" {
			t.Errorf("expected status to be 'stopped', got: %s", getStringField(serviceData, "status"))
		}
	})

	t.Run("delete service", func(t *testing.T) {
		createResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "service.create",
			Input: map[string]any{
				"model_id": "delete-test-model",
				"replicas": 1,
			},
		})
		assertSuccess(t, createResp)

		data := getMapField(createResp.Data, "")
		serviceID := getStringField(data, "service_id")

		deleteResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "service.delete",
			Input: map[string]any{
				"service_id": serviceID,
			},
		})
		assertSuccess(t, deleteResp)

		deleteData := getMapField(deleteResp.Data, "")
		if deleteData["success"] != true {
			t.Errorf("expected success to be true")
		}
	})

	t.Run("get deleted service should fail", func(t *testing.T) {
		createResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "service.create",
			Input: map[string]any{
				"model_id": "temp-delete-model",
				"replicas": 1,
			},
		})
		assertSuccess(t, createResp)

		data := getMapField(createResp.Data, "")
		serviceID := getStringField(data, "service_id")

		deleteResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "service.delete",
			Input: map[string]any{
				"service_id": serviceID,
			},
		})
		assertSuccess(t, deleteResp)

		getResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "query",
			Unit: "service.get",
			Input: map[string]any{
				"service_id": serviceID,
			},
		})
		assertError(t, getResp)
	})

	t.Run("full service lifecycle", func(t *testing.T) {
		createResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "service.create",
			Input: map[string]any{
				"model_id":       "full-lifecycle-model",
				"resource_class": "large",
				"replicas":       2,
			},
		})
		assertSuccess(t, createResp)

		data := getMapField(createResp.Data, "")
		serviceID := getStringField(data, "service_id")

		getResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "query",
			Unit: "service.get",
			Input: map[string]any{
				"service_id": serviceID,
			},
		})
		assertSuccess(t, getResp)

		startResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "service.start",
			Input: map[string]any{
				"service_id": serviceID,
			},
		})
		assertSuccess(t, startResp)

		scaleResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "service.scale",
			Input: map[string]any{
				"service_id": serviceID,
				"replicas":   5,
			},
		})
		assertSuccess(t, scaleResp)

		stopResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "service.stop",
			Input: map[string]any{
				"service_id": serviceID,
			},
		})
		assertSuccess(t, stopResp)

		deleteResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "service.delete",
			Input: map[string]any{
				"service_id": serviceID,
			},
		})
		assertSuccess(t, deleteResp)
	})

	t.Run("filter services by status", func(t *testing.T) {
		createResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "service.create",
			Input: map[string]any{
				"model_id": "filter-status-model",
				"replicas": 1,
			},
		})
		assertSuccess(t, createResp)

		filterResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "query",
			Unit: "service.list",
			Input: map[string]any{
				"status": "creating",
			},
		})
		assertSuccess(t, filterResp)

		data := getMapField(filterResp.Data, "")
		if data == nil {
			t.Fatalf("expected data to be non-nil, got: %v", filterResp.Data)
		}
		total := getIntField(filterResp.Data, "total")
		if total < 0 {
			t.Errorf("expected non-negative total, got: %d", total)
		}
	})
}
