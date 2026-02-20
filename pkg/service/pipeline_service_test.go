package service

import (
	"context"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/eventbus"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/pipeline"
)

func TestPipelineService_NewPipelineService(t *testing.T) {
	store := pipeline.NewMemoryStore()
	executor := pipeline.NewExecutor(store, pipeline.MockStepExecutor)
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	tests := []struct {
		name     string
		registry *unit.Registry
		store    pipeline.PipelineStore
		executor *pipeline.Executor
		bus      *eventbus.InMemoryEventBus
	}{
		{
			name:     "with all dependencies",
			registry: unit.NewRegistry(),
			store:    store,
			executor: executor,
			bus:      bus,
		},
		{
			name:     "with nil bus",
			registry: unit.NewRegistry(),
			store:    store,
			executor: executor,
			bus:      nil,
		},
		{
			name:     "with nil executor",
			registry: unit.NewRegistry(),
			store:    store,
			executor: nil,
			bus:      bus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewPipelineService(tt.registry, tt.store, tt.executor, tt.bus)
			if svc == nil {
				t.Error("expected non-nil PipelineService")
			}
		})
	}
}

func TestPipelineService_CreateWithValidation_Success(t *testing.T) {
	store := pipeline.NewMemoryStore()
	executor := pipeline.NewExecutor(store, pipeline.MockStepExecutor)
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	_ = registry.RegisterCommand(&mockCommand{
		name: "pipeline.create",
		execute: func(ctx context.Context, input any) (any, error) {
			p := &pipeline.Pipeline{
				ID:        "pipeline-test123",
				Name:      "test-pipeline",
				Steps:     []pipeline.PipelineStep{{ID: "step1", Name: "Step 1", Type: "inference.chat"}},
				Status:    pipeline.PipelineStatusIdle,
				CreatedAt: time.Now().Unix(),
				UpdatedAt: time.Now().Unix(),
			}
			store.CreatePipeline(ctx, p)
			return map[string]any{"pipeline_id": p.ID}, nil
		},
	})

	svc := NewPipelineService(registry, store, executor, bus)

	steps := []pipeline.PipelineStep{
		{ID: "step1", Name: "Step 1", Type: "inference.chat"},
	}

	result, err := svc.CreateWithValidation(context.Background(), "test-pipeline", steps, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.Valid {
		t.Error("expected valid=true")
	}
	if result.PipelineID == "" {
		t.Error("expected non-empty pipeline_id")
	}
}

func TestPipelineService_CreateWithValidation_InvalidSteps(t *testing.T) {
	store := pipeline.NewMemoryStore()
	executor := pipeline.NewExecutor(store, pipeline.MockStepExecutor)
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	svc := NewPipelineService(registry, store, executor, bus)

	steps := []pipeline.PipelineStep{
		{ID: "", Name: "Invalid Step", Type: "inference.chat"},
	}

	result, err := svc.CreateWithValidation(context.Background(), "test-pipeline", steps, nil)
	if err == nil {
		t.Fatal("expected error for invalid steps")
	}
	if result != nil && result.Valid {
		t.Error("expected valid=false for invalid steps")
	}
}

func TestPipelineService_CreateWithValidation_CommandNotFound(t *testing.T) {
	store := pipeline.NewMemoryStore()
	executor := pipeline.NewExecutor(store, pipeline.MockStepExecutor)
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	svc := NewPipelineService(registry, store, executor, bus)

	steps := []pipeline.PipelineStep{
		{ID: "step1", Name: "Step 1", Type: "inference.chat"},
	}

	result, err := svc.CreateWithValidation(context.Background(), "test-pipeline", steps, nil)
	if err == nil {
		t.Fatal("expected error for missing command")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

func TestPipelineService_RunAsync_Success(t *testing.T) {
	store := pipeline.NewMemoryStore()
	executor := pipeline.NewExecutor(store, pipeline.MockStepExecutor)
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	_ = registry.RegisterCommand(&mockCommand{
		name: "pipeline.run",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{"run_id": "run-123", "status": "running"}, nil
		},
	})

	svc := NewPipelineService(registry, store, executor, bus)

	result, err := svc.RunAsync(context.Background(), "pipeline-123", map[string]any{"model": "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.RunID == "" {
		t.Error("expected non-empty run_id")
	}
}

func TestPipelineService_RunAsync_CommandNotFound(t *testing.T) {
	store := pipeline.NewMemoryStore()
	executor := pipeline.NewExecutor(store, pipeline.MockStepExecutor)
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	svc := NewPipelineService(registry, store, executor, bus)

	result, err := svc.RunAsync(context.Background(), "pipeline-123", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

func TestPipelineService_GetRunWithResults_Success(t *testing.T) {
	store := pipeline.NewMemoryStore()
	executor := pipeline.NewExecutor(store, pipeline.MockStepExecutor)
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	now := time.Now()
	store.CreateRun(context.Background(), &pipeline.PipelineRun{
		ID:          "run-123",
		PipelineID:  "pipeline-123",
		Status:      pipeline.RunStatusCompleted,
		StepResults: map[string]any{"step1": map[string]any{"result": "success"}},
		StartedAt:   now,
	})

	registry := unit.NewRegistry()
	_ = registry.RegisterQuery(&mockQuery{
		name: "pipeline.status",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"status":       "completed",
				"step_results": map[string]any{"step1": map[string]any{"result": "success"}},
			}, nil
		},
	})

	svc := NewPipelineService(registry, store, executor, bus)

	result, err := svc.GetRunWithResults(context.Background(), "run-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.RunID != "run-123" {
		t.Errorf("expected run_id=run-123, got %s", result.RunID)
	}
	if result.Status != pipeline.RunStatusCompleted {
		t.Errorf("expected status=completed, got %s", result.Status)
	}
}

func TestPipelineService_CancelRun_Success(t *testing.T) {
	store := pipeline.NewMemoryStore()
	executor := pipeline.NewExecutor(store, pipeline.MockStepExecutor)
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	store.CreateRun(context.Background(), &pipeline.PipelineRun{
		ID:         "run-123",
		PipelineID: "pipeline-123",
		Status:     pipeline.RunStatusRunning,
		StartedAt:  time.Now(),
	})

	registry := unit.NewRegistry()
	_ = registry.RegisterCommand(&mockCommand{
		name: "pipeline.cancel",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{"success": true}, nil
		},
	})

	svc := NewPipelineService(registry, store, executor, bus)

	result, err := svc.CancelRun(context.Background(), "run-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected success=true")
	}
}

func TestPipelineService_CancelRun_NotCancellable(t *testing.T) {
	store := pipeline.NewMemoryStore()
	executor := pipeline.NewExecutor(store, pipeline.MockStepExecutor)
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	now := time.Now()
	store.CreateRun(context.Background(), &pipeline.PipelineRun{
		ID:          "run-123",
		PipelineID:  "pipeline-123",
		Status:      pipeline.RunStatusCompleted,
		StartedAt:   now,
		CompletedAt: &now,
	})

	registry := unit.NewRegistry()
	svc := NewPipelineService(registry, store, executor, bus)

	result, err := svc.CancelRun(context.Background(), "run-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Success {
		t.Error("expected success=false for completed run")
	}
}

func TestPipelineService_ListByStatus_Success(t *testing.T) {
	store := pipeline.NewMemoryStore()
	executor := pipeline.NewExecutor(store, pipeline.MockStepExecutor)
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	_ = registry.RegisterQuery(&mockQuery{
		name: "pipeline.list",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"pipelines": []any{
					map[string]any{"id": "p1", "name": "Pipeline 1", "status": "idle", "created_at": int64(1000), "updated_at": int64(1000)},
					map[string]any{"id": "p2", "name": "Pipeline 2", "status": "running", "created_at": int64(1001), "updated_at": int64(1001)},
				},
				"total": 2,
			}, nil
		},
	})

	svc := NewPipelineService(registry, store, executor, bus)

	summaries, total, err := svc.ListByStatus(context.Background(), "", 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(summaries) != 2 {
		t.Errorf("expected 2 pipelines, got %d", len(summaries))
	}
	if total != 2 {
		t.Errorf("expected total=2, got %d", total)
	}
}

func TestPipelineService_ValidateDefinition_Success(t *testing.T) {
	store := pipeline.NewMemoryStore()
	executor := pipeline.NewExecutor(store, pipeline.MockStepExecutor)
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	registry := unit.NewRegistry()
	_ = registry.RegisterQuery(&mockQuery{
		name: "pipeline.validate",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{"valid": true, "issues": []string{}}, nil
		},
	})

	svc := NewPipelineService(registry, store, executor, bus)

	steps := []pipeline.PipelineStep{
		{ID: "step1", Name: "Step 1", Type: "inference.chat"},
	}

	result, err := svc.ValidateDefinition(context.Background(), steps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Valid {
		t.Error("expected valid=true")
	}
	if result.HasCycle {
		t.Error("expected has_cycle=false")
	}
}

func TestPipelineService_DeleteWithCleanup_Success(t *testing.T) {
	store := pipeline.NewMemoryStore()
	executor := pipeline.NewExecutor(store, pipeline.MockStepExecutor)
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	now := time.Now().Unix()
	store.CreatePipeline(context.Background(), &pipeline.Pipeline{
		ID:        "pipeline-123",
		Name:      "test-pipeline",
		Status:    pipeline.PipelineStatusIdle,
		CreatedAt: now,
		UpdatedAt: now,
	})

	registry := unit.NewRegistry()
	_ = registry.RegisterCommand(&mockCommand{
		name: "pipeline.delete",
		execute: func(ctx context.Context, input any) (any, error) {
			inputMap := input.(map[string]any)
			_ = store.DeletePipeline(ctx, inputMap["pipeline_id"].(string))
			return map[string]any{"success": true}, nil
		},
	})

	svc := NewPipelineService(registry, store, executor, bus)

	result, err := svc.DeleteWithCleanup(context.Background(), "pipeline-123", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected success=true")
	}
}

func TestPipelineService_DeleteWithCleanup_ActiveRuns(t *testing.T) {
	store := pipeline.NewMemoryStore()
	executor := pipeline.NewExecutor(store, pipeline.MockStepExecutor)
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	now := time.Now().Unix()
	store.CreatePipeline(context.Background(), &pipeline.Pipeline{
		ID:        "pipeline-123",
		Name:      "test-pipeline",
		Status:    pipeline.PipelineStatusRunning,
		CreatedAt: now,
		UpdatedAt: now,
	})
	store.CreateRun(context.Background(), &pipeline.PipelineRun{
		ID:         "run-123",
		PipelineID: "pipeline-123",
		Status:     pipeline.RunStatusRunning,
		StartedAt:  time.Now(),
	})

	registry := unit.NewRegistry()
	svc := NewPipelineService(registry, store, executor, bus)

	result, err := svc.DeleteWithCleanup(context.Background(), "pipeline-123", false)
	if err == nil {
		t.Fatal("expected error for active runs without force")
	}
	if result != nil {
		t.Errorf("expected nil result, got %+v", result)
	}
}

func TestPipelineService_DeleteWithCleanup_Force(t *testing.T) {
	store := pipeline.NewMemoryStore()
	executor := pipeline.NewExecutor(store, pipeline.MockStepExecutor)
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	now := time.Now().Unix()
	store.CreatePipeline(context.Background(), &pipeline.Pipeline{
		ID:        "pipeline-123",
		Name:      "test-pipeline",
		Status:    pipeline.PipelineStatusRunning,
		CreatedAt: now,
		UpdatedAt: now,
	})
	store.CreateRun(context.Background(), &pipeline.PipelineRun{
		ID:         "run-123",
		PipelineID: "pipeline-123",
		Status:     pipeline.RunStatusRunning,
		StartedAt:  time.Now(),
	})

	registry := unit.NewRegistry()
	_ = registry.RegisterCommand(&mockCommand{
		name: "pipeline.cancel",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{"success": true}, nil
		},
	})
	_ = registry.RegisterCommand(&mockCommand{
		name: "pipeline.delete",
		execute: func(ctx context.Context, input any) (any, error) {
			inputMap := input.(map[string]any)
			_ = store.DeletePipeline(ctx, inputMap["pipeline_id"].(string))
			return map[string]any{"success": true}, nil
		},
	})

	svc := NewPipelineService(registry, store, executor, bus)

	result, err := svc.DeleteWithCleanup(context.Background(), "pipeline-123", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Error("expected success=true")
	}
	if !result.Cancelled {
		t.Error("expected cancelled=true when force deleting with active runs")
	}
}

func TestPipelineService_GetPipeline(t *testing.T) {
	store := pipeline.NewMemoryStore()
	executor := pipeline.NewExecutor(store, pipeline.MockStepExecutor)
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	now := time.Now().Unix()
	store.CreatePipeline(context.Background(), &pipeline.Pipeline{
		ID:        "pipeline-123",
		Name:      "test-pipeline",
		Status:    pipeline.PipelineStatusIdle,
		CreatedAt: now,
		UpdatedAt: now,
	})

	registry := unit.NewRegistry()
	svc := NewPipelineService(registry, store, executor, bus)

	p, err := svc.GetPipeline(context.Background(), "pipeline-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.ID != "pipeline-123" {
		t.Errorf("expected ID=pipeline-123, got %s", p.ID)
	}
}

func TestPipelineService_ListRuns(t *testing.T) {
	store := pipeline.NewMemoryStore()
	executor := pipeline.NewExecutor(store, pipeline.MockStepExecutor)
	bus := eventbus.NewInMemoryEventBus()
	defer func() { _ = bus.Close() }()

	store.CreateRun(context.Background(), &pipeline.PipelineRun{
		ID:         "run-1",
		PipelineID: "pipeline-123",
		Status:     pipeline.RunStatusCompleted,
		StartedAt:  time.Now(),
	})
	store.CreateRun(context.Background(), &pipeline.PipelineRun{
		ID:         "run-2",
		PipelineID: "pipeline-123",
		Status:     pipeline.RunStatusCompleted,
		StartedAt:  time.Now(),
	})

	registry := unit.NewRegistry()
	svc := NewPipelineService(registry, store, executor, bus)

	runs, err := svc.ListRuns(context.Background(), "pipeline-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(runs) != 2 {
		t.Errorf("expected 2 runs, got %d", len(runs))
	}
}
