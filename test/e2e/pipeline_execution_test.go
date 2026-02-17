package e2e

import (
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
)

func TestPipelineExecutionE2E(t *testing.T) {
	env := SetupTestEnv(t)

	t.Run("create pipeline", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "pipeline.create",
			Input: map[string]any{
				"name": "test-pipeline",
				"steps": []any{
					map[string]any{
						"id":    "step1",
						"name":  "List Models",
						"type":  "model.list",
						"input": map[string]any{},
					},
				},
			},
		})
		assertSuccess(t, resp)

		data := getMapField(resp.Data, "")
		pipelineID := getStringField(data, "pipeline_id")
		if pipelineID == "" {
			t.Errorf("expected pipeline_id to be non-empty")
		}
	})

	t.Run("create and get pipeline", func(t *testing.T) {
		createResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "pipeline.create",
			Input: map[string]any{
				"name": "get-test-pipeline",
				"steps": []any{
					map[string]any{
						"id":    "step1",
						"name":  "Step 1",
						"type":  "model.list",
						"input": map[string]any{},
					},
				},
			},
		})
		assertSuccess(t, createResp)

		data := getMapField(createResp.Data, "")
		pipelineID := getStringField(data, "pipeline_id")

		getResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "query",
			Unit: "pipeline.get",
			Input: map[string]any{
				"pipeline_id": pipelineID,
			},
		})
		assertSuccess(t, getResp)

		pipelineData := getMapField(getResp.Data, "")
		if getStringField(pipelineData, "name") != "get-test-pipeline" {
			t.Errorf("expected name 'get-test-pipeline'")
		}
		if getStringField(pipelineData, "status") != "idle" {
			t.Errorf("expected status 'idle', got: %s", getStringField(pipelineData, "status"))
		}
	})

	t.Run("list pipelines", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
				Type: "command",
				Unit: "pipeline.create",
				Input: map[string]any{
					"name": "list-test-pipeline",
					"steps": []any{
						map[string]any{
							"id":    "step1",
							"name":  "Step 1",
							"type":  "model.list",
							"input": map[string]any{},
						},
					},
				},
			})
			assertSuccess(t, resp)
		}

		listResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type:  "query",
			Unit:  "pipeline.list",
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

	t.Run("validate pipeline steps", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "query",
			Unit: "pipeline.validate",
			Input: map[string]any{
				"steps": []any{
					map[string]any{
						"id":    "step1",
						"name":  "Step 1",
						"type":  "model.list",
						"input": map[string]any{},
					},
					map[string]any{
						"id":         "step2",
						"name":       "Step 2",
						"type":       "model.get",
						"depends_on": []any{"step1"},
						"input":      map[string]any{},
					},
				},
			},
		})
		assertSuccess(t, resp)

		data := getMapField(resp.Data, "")
		if data["valid"] != true {
			t.Errorf("expected valid to be true")
		}
	})

	t.Run("validate invalid pipeline steps", func(t *testing.T) {
		resp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "query",
			Unit: "pipeline.validate",
			Input: map[string]any{
				"steps": []any{
					map[string]any{
						"id":    "",
						"name":  "Empty ID Step",
						"type":  "model.list",
						"input": map[string]any{},
					},
				},
			},
		})
		assertSuccess(t, resp)

		data := getMapField(resp.Data, "")
		if data["valid"] == true {
			t.Errorf("expected valid to be false for invalid steps")
		}
		if data["valid"] == false {
			issues := getSliceField(data, "issues")
			if len(issues) == 0 {
				t.Log("validation correctly returned invalid, but no issues listed")
			}
		}
	})

	t.Run("run pipeline", func(t *testing.T) {
		createResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "pipeline.create",
			Input: map[string]any{
				"name": "run-test-pipeline",
				"steps": []any{
					map[string]any{
						"id":    "step1",
						"name":  "List Models",
						"type":  "model.list",
						"input": map[string]any{},
					},
				},
			},
		})
		assertSuccess(t, createResp)

		data := getMapField(createResp.Data, "")
		pipelineID := getStringField(data, "pipeline_id")

		runResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "pipeline.run",
			Input: map[string]any{
				"pipeline_id": pipelineID,
				"input":       map[string]any{},
			},
		})
		assertSuccess(t, runResp)

		runData := getMapField(runResp.Data, "")
		runID := getStringField(runData, "run_id")
		if runID == "" {
			t.Errorf("expected run_id to be non-empty")
		}
	})

	t.Run("get run status", func(t *testing.T) {
		createResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "pipeline.create",
			Input: map[string]any{
				"name": "status-test-pipeline",
				"steps": []any{
					map[string]any{
						"id":    "step1",
						"name":  "List Models",
						"type":  "model.list",
						"input": map[string]any{},
					},
				},
			},
		})
		assertSuccess(t, createResp)

		data := getMapField(createResp.Data, "")
		pipelineID := getStringField(data, "pipeline_id")

		runResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "pipeline.run",
			Input: map[string]any{
				"pipeline_id": pipelineID,
				"input":       map[string]any{},
			},
		})
		assertSuccess(t, runResp)

		runData := getMapField(runResp.Data, "")
		runID := getStringField(runData, "run_id")

		statusResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "query",
			Unit: "pipeline.status",
			Input: map[string]any{
				"run_id": runID,
			},
		})
		assertSuccess(t, statusResp)

		statusData := getMapField(statusResp.Data, "")
		status := getStringField(statusData, "status")
		if status == "" {
			t.Errorf("expected status to be non-empty")
		}
	})

	t.Run("cancel pipeline run", func(t *testing.T) {
		createResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "pipeline.create",
			Input: map[string]any{
				"name": "cancel-test-pipeline",
				"steps": []any{
					map[string]any{
						"id":    "step1",
						"name":  "List Models",
						"type":  "model.list",
						"input": map[string]any{},
					},
				},
			},
		})
		assertSuccess(t, createResp)

		data := getMapField(createResp.Data, "")
		pipelineID := getStringField(data, "pipeline_id")

		runResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "pipeline.run",
			Input: map[string]any{
				"pipeline_id": pipelineID,
				"input":       map[string]any{},
			},
		})
		assertSuccess(t, runResp)

		runData := getMapField(runResp.Data, "")
		runID := getStringField(runData, "run_id")

		cancelResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "pipeline.cancel",
			Input: map[string]any{
				"run_id": runID,
			},
		})
		assertSuccess(t, cancelResp)

		cancelData := getMapField(cancelResp.Data, "")
		if cancelData["success"] != true {
			t.Errorf("expected success to be true")
		}
	})

	t.Run("delete pipeline", func(t *testing.T) {
		createResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "pipeline.create",
			Input: map[string]any{
				"name": "delete-test-pipeline",
				"steps": []any{
					map[string]any{
						"id":    "step1",
						"name":  "Step 1",
						"type":  "model.list",
						"input": map[string]any{},
					},
				},
			},
		})
		assertSuccess(t, createResp)

		data := getMapField(createResp.Data, "")
		pipelineID := getStringField(data, "pipeline_id")

		deleteResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "pipeline.delete",
			Input: map[string]any{
				"pipeline_id": pipelineID,
			},
		})
		assertSuccess(t, deleteResp)

		deleteData := getMapField(deleteResp.Data, "")
		if deleteData["success"] != true {
			t.Errorf("expected success to be true")
		}
	})

	t.Run("get deleted pipeline should fail", func(t *testing.T) {
		createResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "pipeline.create",
			Input: map[string]any{
				"name": "temp-delete-pipeline",
				"steps": []any{
					map[string]any{
						"id":    "step1",
						"name":  "Step 1",
						"type":  "model.list",
						"input": map[string]any{},
					},
				},
			},
		})
		assertSuccess(t, createResp)

		data := getMapField(createResp.Data, "")
		pipelineID := getStringField(data, "pipeline_id")

		deleteResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "pipeline.delete",
			Input: map[string]any{
				"pipeline_id": pipelineID,
			},
		})
		assertSuccess(t, deleteResp)

		getResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "query",
			Unit: "pipeline.get",
			Input: map[string]any{
				"pipeline_id": pipelineID,
			},
		})
		assertError(t, getResp)
	})

	t.Run("multi-step pipeline with dependencies", func(t *testing.T) {
		createResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "pipeline.create",
			Input: map[string]any{
				"name": "multi-step-pipeline",
				"steps": []any{
					map[string]any{
						"id":    "detect",
						"name":  "Detect Devices",
						"type":  "model.list",
						"input": map[string]any{},
					},
					map[string]any{
						"id":         "list",
						"name":       "List Models",
						"type":       "model.list",
						"depends_on": []any{"detect"},
						"input":      map[string]any{},
					},
				},
			},
		})
		assertSuccess(t, createResp)

		data := getMapField(createResp.Data, "")
		pipelineID := getStringField(data, "pipeline_id")

		runResp := env.Gateway.Handle(env.Ctx, &gateway.Request{
			Type: "command",
			Unit: "pipeline.run",
			Input: map[string]any{
				"pipeline_id": pipelineID,
				"input":       map[string]any{},
			},
		})
		assertSuccess(t, runResp)
	})
}
