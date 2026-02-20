package pipeline

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// --- helpers ---

func newPipelineWithSteps(steps []PipelineStep) *Pipeline {
	now := time.Now().Unix()
	return &Pipeline{
		ID:        "pipe-test",
		Name:      "test-pipeline",
		Status:    PipelineStatusIdle,
		Steps:     steps,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func newRun(id, pipelineID string) *PipelineRun {
	return &PipelineRun{
		ID:          id,
		PipelineID:  pipelineID,
		Status:      RunStatusPending,
		StepResults: make(map[string]any),
	}
}

// waitForRunStatus polls the store until the run reaches the expected status or the deadline expires.
func waitForRunStatus(t *testing.T, store PipelineStore, runID string, expected RunStatus, timeout time.Duration) *PipelineRun {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		run, err := store.GetRun(context.Background(), runID)
		if err == nil && run.Status == expected {
			return run
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("run %s did not reach status %q within %s", runID, expected, timeout)
	return nil
}

// --- NewExecutor ---

func TestNewExecutor_NotNil(t *testing.T) {
	store := NewMemoryStore()
	exec := NewExecutor(store, MockStepExecutor)
	if exec == nil {
		t.Fatal("expected non-nil Executor")
	}
}

func TestNewExecutor_NilStepExecutor(t *testing.T) {
	exec := NewExecutor(NewMemoryStore(), nil)
	if exec == nil {
		t.Fatal("expected non-nil Executor even with nil stepExecutor")
	}
}

// --- Execute: basic lifecycle ---

func TestExecutor_Execute_SingleStep_Success(t *testing.T) {
	store := NewMemoryStore()
	pipeline := newPipelineWithSteps([]PipelineStep{
		{ID: "step1", Name: "Step 1", Type: "inference.chat", Input: map[string]any{"model": "test"}},
	})
	_ = store.CreatePipeline(context.Background(), pipeline)

	run := newRun("run-1", pipeline.ID)
	_ = store.CreateRun(context.Background(), run)

	exec := NewExecutor(store, MockStepExecutor)
	err := exec.Execute(context.Background(), pipeline, run, nil)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	// Run starts as Running immediately
	if run.Status != RunStatusRunning {
		t.Errorf("expected status Running after Execute, got %s", run.Status)
	}

	// Wait for the goroutine to complete
	finalRun := waitForRunStatus(t, store, "run-1", RunStatusCompleted, 2*time.Second)
	if finalRun.CompletedAt == nil {
		t.Error("expected CompletedAt to be set on completion")
	}
}

func TestExecutor_Execute_MultiStep_Success(t *testing.T) {
	store := NewMemoryStore()
	pipeline := newPipelineWithSteps([]PipelineStep{
		{ID: "step1", Name: "Step 1", Type: "inference.chat"},
		{ID: "step2", Name: "Step 2", Type: "inference.embed"},
		{ID: "step3", Name: "Step 3", Type: "inference.summarize"},
	})
	_ = store.CreatePipeline(context.Background(), pipeline)

	run := newRun("run-1", pipeline.ID)
	_ = store.CreateRun(context.Background(), run)

	exec := NewExecutor(store, MockStepExecutor)
	err := exec.Execute(context.Background(), pipeline, run, nil)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	finalRun := waitForRunStatus(t, store, "run-1", RunStatusCompleted, 2*time.Second)

	// All step results should be populated
	for _, step := range pipeline.Steps {
		if _, ok := finalRun.StepResults[step.ID]; !ok {
			t.Errorf("expected step result for %q, not found", step.ID)
		}
	}
}

func TestExecutor_Execute_NilStepExecutor_UsesMockOutput(t *testing.T) {
	store := NewMemoryStore()
	pipeline := newPipelineWithSteps([]PipelineStep{
		{ID: "step1", Name: "Step 1", Type: "inference.chat"},
	})
	_ = store.CreatePipeline(context.Background(), pipeline)

	run := newRun("run-1", pipeline.ID)
	_ = store.CreateRun(context.Background(), run)

	// nil stepExecutor → executor uses built-in mock path
	exec := NewExecutor(store, nil)
	err := exec.Execute(context.Background(), pipeline, run, nil)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	finalRun := waitForRunStatus(t, store, "run-1", RunStatusCompleted, 2*time.Second)

	result, ok := finalRun.StepResults["step1"]
	if !ok {
		t.Fatal("expected step1 result in StepResults")
	}
	resultMap, ok := result.(map[string]any)
	if !ok {
		t.Fatal("expected map[string]any for step result")
	}
	if resultMap["mock"] != true {
		t.Errorf("expected mock=true in nil-executor output, got %v", resultMap["mock"])
	}
}

// --- Execute: input merging ---

func TestExecutor_Execute_StepInputMergedWithRunInput(t *testing.T) {
	var capturedInput map[string]any

	captureExecutor := func(ctx context.Context, stepType string, input map[string]any) (map[string]any, error) {
		capturedInput = input
		return map[string]any{"ok": true}, nil
	}

	store := NewMemoryStore()
	pipeline := newPipelineWithSteps([]PipelineStep{
		{ID: "step1", Type: "test", Input: map[string]any{"from_step": "yes"}},
	})
	_ = store.CreatePipeline(context.Background(), pipeline)

	run := newRun("run-1", pipeline.ID)
	_ = store.CreateRun(context.Background(), run)

	exec := NewExecutor(store, captureExecutor)
	runInput := map[string]any{"from_run": "yes"}
	if err := exec.Execute(context.Background(), pipeline, run, runInput); err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	waitForRunStatus(t, store, "run-1", RunStatusCompleted, 2*time.Second)

	if capturedInput["from_step"] != "yes" {
		t.Error("expected from_step=yes in merged input")
	}
	if capturedInput["from_run"] != "yes" {
		t.Error("expected from_run=yes in merged input")
	}
}

func TestExecutor_Execute_RunInputOverridesStepInput(t *testing.T) {
	var capturedInput map[string]any

	captureExecutor := func(ctx context.Context, stepType string, input map[string]any) (map[string]any, error) {
		capturedInput = input
		return map[string]any{"ok": true}, nil
	}

	store := NewMemoryStore()
	pipeline := newPipelineWithSteps([]PipelineStep{
		{ID: "step1", Type: "test", Input: map[string]any{"key": "step_value"}},
	})
	_ = store.CreatePipeline(context.Background(), pipeline)

	run := newRun("run-1", pipeline.ID)
	_ = store.CreateRun(context.Background(), run)

	exec := NewExecutor(store, captureExecutor)
	runInput := map[string]any{"key": "run_value"}
	if err := exec.Execute(context.Background(), pipeline, run, runInput); err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	waitForRunStatus(t, store, "run-1", RunStatusCompleted, 2*time.Second)

	// run input overwrites step input because it's applied last
	if capturedInput["key"] != "run_value" {
		t.Errorf("expected run input to override step input, got %v", capturedInput["key"])
	}
}

// --- Execute: status transitions ---

func TestExecutor_Execute_SetsRunningAndStartedAt(t *testing.T) {
	blockCh := make(chan struct{})
	blockingExecutor := func(ctx context.Context, stepType string, input map[string]any) (map[string]any, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-blockCh:
			return map[string]any{}, nil
		}
	}

	store := NewMemoryStore()
	pipeline := newPipelineWithSteps([]PipelineStep{
		{ID: "step1", Type: "test"},
	})
	_ = store.CreatePipeline(context.Background(), pipeline)

	run := newRun("run-1", pipeline.ID)
	_ = store.CreateRun(context.Background(), run)

	before := time.Now()
	exec := NewExecutor(store, blockingExecutor)
	_ = exec.Execute(context.Background(), pipeline, run, nil)

	// Goroutine is blocked at step executor — run.Status was set synchronously.
	if run.Status != RunStatusRunning {
		t.Errorf("expected RunStatusRunning immediately after Execute, got %s", run.Status)
	}
	if run.StartedAt.Before(before) {
		t.Error("expected StartedAt to be set after Execute was called")
	}

	close(blockCh)
	waitForRunStatus(t, store, "run-1", RunStatusCompleted, 2*time.Second)
}

func TestExecutor_Execute_StoreUpdateRunCalledOnRunning(t *testing.T) {
	store := NewMemoryStore()
	pipeline := newPipelineWithSteps([]PipelineStep{
		{ID: "step1", Type: "test"},
	})
	_ = store.CreatePipeline(context.Background(), pipeline)

	run := newRun("run-1", pipeline.ID)
	_ = store.CreateRun(context.Background(), run)

	exec := NewExecutor(store, MockStepExecutor)
	_ = exec.Execute(context.Background(), pipeline, run, nil)

	// After Execute returns, store should reflect Running status
	stored, err := store.GetRun(context.Background(), "run-1")
	if err != nil {
		t.Fatalf("GetRun error: %v", err)
	}
	if stored.Status != RunStatusRunning && stored.Status != RunStatusCompleted {
		t.Errorf("expected store to have Running or Completed status, got %s", stored.Status)
	}
}

func TestExecutor_Execute_UpdateRunFailure_ReturnsError(t *testing.T) {
	// If UpdateRun fails on first call (setting Running), Execute returns error.
	store := NewMemoryStore()
	pipeline := newPipelineWithSteps([]PipelineStep{
		{ID: "step1", Type: "test"},
	})
	_ = store.CreatePipeline(context.Background(), pipeline)

	// Don't add the run to the store so UpdateRun on "run-missing" will fail.
	run := &PipelineRun{
		ID:          "run-missing",
		PipelineID:  pipeline.ID,
		Status:      RunStatusPending,
		StepResults: make(map[string]any),
	}

	exec := NewExecutor(store, MockStepExecutor)
	err := exec.Execute(context.Background(), pipeline, run, nil)
	if err == nil {
		t.Fatal("expected error when UpdateRun fails (run not in store), got nil")
	}
}

// --- Execute: dependency resolution ---

func TestExecutor_Execute_DependencyNotSatisfied_FailsRun(t *testing.T) {
	store := NewMemoryStore()
	// step2 depends on step1 but step1 is listed AFTER step2 → dependency not yet in executedSteps
	pipeline := newPipelineWithSteps([]PipelineStep{
		{ID: "step2", Type: "test", DependsOn: []string{"step1"}},
		{ID: "step1", Type: "test"},
	})
	_ = store.CreatePipeline(context.Background(), pipeline)

	run := newRun("run-1", pipeline.ID)
	_ = store.CreateRun(context.Background(), run)

	exec := NewExecutor(store, MockStepExecutor)
	_ = exec.Execute(context.Background(), pipeline, run, nil)

	finalRun := waitForRunStatus(t, store, "run-1", RunStatusFailed, 2*time.Second)
	if finalRun.Error == "" {
		t.Error("expected non-empty Error on failed run")
	}
	if finalRun.CompletedAt == nil {
		t.Error("expected CompletedAt to be set on failed run")
	}
}

func TestExecutor_Execute_DependencySatisfied_Succeeds(t *testing.T) {
	store := NewMemoryStore()
	// Proper order: step1 first, step2 depends on step1
	pipeline := newPipelineWithSteps([]PipelineStep{
		{ID: "step1", Type: "test"},
		{ID: "step2", Type: "test", DependsOn: []string{"step1"}},
	})
	_ = store.CreatePipeline(context.Background(), pipeline)

	run := newRun("run-1", pipeline.ID)
	_ = store.CreateRun(context.Background(), run)

	exec := NewExecutor(store, MockStepExecutor)
	_ = exec.Execute(context.Background(), pipeline, run, nil)

	finalRun := waitForRunStatus(t, store, "run-1", RunStatusCompleted, 2*time.Second)
	if _, ok := finalRun.StepResults["step2"]; !ok {
		t.Error("expected step2 result after completed run")
	}
}

// --- Execute: step failure ---

func TestExecutor_Execute_StepExecutorError_FailsRun(t *testing.T) {
	failingExecutor := func(ctx context.Context, stepType string, input map[string]any) (map[string]any, error) {
		return nil, errors.New("step exploded")
	}

	store := NewMemoryStore()
	pipeline := newPipelineWithSteps([]PipelineStep{
		{ID: "step1", Type: "failing-step"},
	})
	_ = store.CreatePipeline(context.Background(), pipeline)

	run := newRun("run-1", pipeline.ID)
	_ = store.CreateRun(context.Background(), run)

	exec := NewExecutor(store, failingExecutor)
	_ = exec.Execute(context.Background(), pipeline, run, nil)

	finalRun := waitForRunStatus(t, store, "run-1", RunStatusFailed, 2*time.Second)
	if finalRun.Error == "" {
		t.Error("expected non-empty Error field on failed run")
	}
	if finalRun.CompletedAt == nil {
		t.Error("expected CompletedAt to be set after failure")
	}
}

func TestExecutor_Execute_StepErrorContainsStepID(t *testing.T) {
	failingExecutor := func(ctx context.Context, stepType string, input map[string]any) (map[string]any, error) {
		return nil, errors.New("step exploded")
	}

	store := NewMemoryStore()
	pipeline := newPipelineWithSteps([]PipelineStep{
		{ID: "my-special-step", Type: "failing-step"},
	})
	_ = store.CreatePipeline(context.Background(), pipeline)

	run := newRun("run-1", pipeline.ID)
	_ = store.CreateRun(context.Background(), run)

	exec := NewExecutor(store, failingExecutor)
	_ = exec.Execute(context.Background(), pipeline, run, nil)

	finalRun := waitForRunStatus(t, store, "run-1", RunStatusFailed, 2*time.Second)

	if finalRun.Error == "" {
		t.Fatal("expected non-empty error string")
	}
	if !strings.Contains(finalRun.Error, "my-special-step") {
		t.Errorf("expected error to contain step ID 'my-special-step', got: %s", finalRun.Error)
	}
}

// --- Execute: context cancellation ---

func TestExecutor_Execute_ContextCancellation_CancelsRun(t *testing.T) {
	// Use a blocking stepExecutor to keep the pipeline in-flight long enough to cancel.
	blockCh := make(chan struct{})
	blockingExecutor := func(ctx context.Context, stepType string, input map[string]any) (map[string]any, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-blockCh:
			return map[string]any{"ok": true}, nil
		}
	}

	store := NewMemoryStore()
	pipeline := newPipelineWithSteps([]PipelineStep{
		{ID: "step1", Type: "blocking"},
		{ID: "step2", Type: "after-block"},
	})
	_ = store.CreatePipeline(context.Background(), pipeline)

	run := newRun("run-1", pipeline.ID)
	_ = store.CreateRun(context.Background(), run)

	exec := NewExecutor(store, blockingExecutor)
	_ = exec.Execute(context.Background(), pipeline, run, nil)

	// Cancel via Executor.Cancel which triggers runCtx cancellation
	time.Sleep(20 * time.Millisecond) // give goroutine time to start
	cancelled := exec.Cancel("run-1")
	if !cancelled {
		t.Log("Cancel returned false — run may have already finished; skipping status check")
		close(blockCh)
		return
	}

	// Unblock the step executor so it can observe ctx.Done
	close(blockCh)

	// Run should end as either Cancelled or Failed (step error from ctx.Err)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		stored, err := store.GetRun(context.Background(), "run-1")
		if err == nil && (stored.Status == RunStatusCancelled || stored.Status == RunStatusFailed) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	stored, _ := store.GetRun(context.Background(), "run-1")
	t.Errorf("expected Cancelled or Failed run after Cancel(), got %s", stored.Status)
}

// --- IsRunning ---

func TestExecutor_IsRunning_TrueWhileRunning(t *testing.T) {
	blockCh := make(chan struct{})
	blockingExecutor := func(ctx context.Context, stepType string, input map[string]any) (map[string]any, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-blockCh:
			return map[string]any{"ok": true}, nil
		}
	}

	store := NewMemoryStore()
	pipeline := newPipelineWithSteps([]PipelineStep{
		{ID: "step1", Type: "blocking"},
	})
	_ = store.CreatePipeline(context.Background(), pipeline)

	run := newRun("run-1", pipeline.ID)
	_ = store.CreateRun(context.Background(), run)

	exec := NewExecutor(store, blockingExecutor)
	_ = exec.Execute(context.Background(), pipeline, run, nil)

	time.Sleep(20 * time.Millisecond)
	if !exec.IsRunning("run-1") {
		t.Error("expected IsRunning to return true while run is in-flight")
	}

	close(blockCh)
	waitForRunStatus(t, store, "run-1", RunStatusCompleted, 2*time.Second)

	if exec.IsRunning("run-1") {
		t.Error("expected IsRunning to return false after run completes")
	}
}

func TestExecutor_IsRunning_FalseForUnknownRun(t *testing.T) {
	exec := NewExecutor(NewMemoryStore(), MockStepExecutor)
	if exec.IsRunning("nonexistent") {
		t.Error("expected IsRunning to return false for unknown run ID")
	}
}

// --- Cancel ---

func TestExecutor_Cancel_ReturnsTrueForRunningRun(t *testing.T) {
	blockCh := make(chan struct{})
	blockingExecutor := func(ctx context.Context, stepType string, input map[string]any) (map[string]any, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-blockCh:
			return map[string]any{"ok": true}, nil
		}
	}

	store := NewMemoryStore()
	pipeline := newPipelineWithSteps([]PipelineStep{
		{ID: "step1", Type: "blocking"},
	})
	_ = store.CreatePipeline(context.Background(), pipeline)

	run := newRun("run-1", pipeline.ID)
	_ = store.CreateRun(context.Background(), run)

	exec := NewExecutor(store, blockingExecutor)
	_ = exec.Execute(context.Background(), pipeline, run, nil)

	time.Sleep(20 * time.Millisecond)
	result := exec.Cancel("run-1")
	if !result {
		t.Error("expected Cancel to return true for an in-flight run")
	}

	close(blockCh)
}

func TestExecutor_Cancel_ReturnsFalseForUnknownRun(t *testing.T) {
	exec := NewExecutor(NewMemoryStore(), MockStepExecutor)
	if exec.Cancel("nonexistent") {
		t.Error("expected Cancel to return false for unknown run ID")
	}
}

func TestExecutor_Cancel_ReturnsFalseAfterCompletion(t *testing.T) {
	store := NewMemoryStore()
	pipeline := newPipelineWithSteps([]PipelineStep{
		{ID: "step1", Type: "test"},
	})
	_ = store.CreatePipeline(context.Background(), pipeline)

	run := newRun("run-1", pipeline.ID)
	_ = store.CreateRun(context.Background(), run)

	exec := NewExecutor(store, MockStepExecutor)
	_ = exec.Execute(context.Background(), pipeline, run, nil)

	waitForRunStatus(t, store, "run-1", RunStatusCompleted, 2*time.Second)

	// After completion the entry is removed from runningRuns
	if exec.Cancel("run-1") {
		t.Error("expected Cancel to return false after run has completed")
	}
}

// --- Concurrent runs ---

func TestExecutor_ConcurrentRuns_SyncMutex(t *testing.T) {
	// Run N pipelines concurrently and verify all complete without data races.
	const N = 10

	store := NewMemoryStore()
	pipeline := newPipelineWithSteps([]PipelineStep{
		{ID: "step1", Type: "test"},
	})
	_ = store.CreatePipeline(context.Background(), pipeline)

	exec := NewExecutor(store, MockStepExecutor)

	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		runID := fmt.Sprintf("run-concurrent-%d", i)
		run := newRun(runID, pipeline.ID)
		_ = store.CreateRun(context.Background(), run)

		wg.Add(1)
		go func(r *PipelineRun) {
			defer wg.Done()
			if err := exec.Execute(context.Background(), pipeline, r, nil); err != nil {
				t.Errorf("Execute error for run %s: %v", r.ID, err)
			}
		}(run)
	}

	wg.Wait()

	// Wait for all runs to complete
	for i := 0; i < N; i++ {
		runID := fmt.Sprintf("run-concurrent-%d", i)
		waitForRunStatus(t, store, runID, RunStatusCompleted, 3*time.Second)
	}
}

func TestExecutor_ConcurrentCancelAndIsRunning_NoRace(t *testing.T) {
	// Hammer IsRunning and Cancel concurrently to exercise sync.Mutex paths.
	blockCh := make(chan struct{})
	blockingExecutor := func(ctx context.Context, stepType string, input map[string]any) (map[string]any, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-blockCh:
			return map[string]any{"ok": true}, nil
		}
	}

	store := NewMemoryStore()
	pipeline := newPipelineWithSteps([]PipelineStep{
		{ID: "step1", Type: "blocking"},
	})
	_ = store.CreatePipeline(context.Background(), pipeline)

	run := newRun("run-1", pipeline.ID)
	_ = store.CreateRun(context.Background(), run)

	exec := NewExecutor(store, blockingExecutor)
	_ = exec.Execute(context.Background(), pipeline, run, nil)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = exec.IsRunning("run-1")
		}()
	}

	exec.Cancel("run-1")
	close(blockCh)
	wg.Wait()
}

// --- ValidateSteps ---

func TestValidateSteps(t *testing.T) {
	tests := []struct {
		name       string
		steps      []PipelineStep
		wantValid  bool
		wantIssues bool
	}{
		{
			name: "valid single step",
			steps: []PipelineStep{
				{ID: "step1", Type: "inference.chat"},
			},
			wantValid:  true,
			wantIssues: false,
		},
		{
			name: "valid multi-step with dependency",
			steps: []PipelineStep{
				{ID: "step1", Type: "inference.chat"},
				{ID: "step2", Type: "inference.embed", DependsOn: []string{"step1"}},
			},
			wantValid:  true,
			wantIssues: false,
		},
		{
			name: "empty step ID",
			steps: []PipelineStep{
				{ID: "", Type: "inference.chat"},
			},
			wantValid:  false,
			wantIssues: true,
		},
		{
			name: "empty step type",
			steps: []PipelineStep{
				{ID: "step1", Type: ""},
			},
			wantValid:  false,
			wantIssues: true,
		},
		{
			name: "duplicate step ID",
			steps: []PipelineStep{
				{ID: "step1", Type: "inference.chat"},
				{ID: "step1", Type: "inference.embed"},
			},
			wantValid:  false,
			wantIssues: true,
		},
		{
			name: "dependency on non-existent step",
			steps: []PipelineStep{
				{ID: "step1", Type: "inference.chat", DependsOn: []string{"nonexistent"}},
			},
			wantValid:  false,
			wantIssues: true,
		},
		{
			name:       "empty steps slice",
			steps:      []PipelineStep{},
			wantValid:  true,
			wantIssues: false,
		},
		{
			name: "multiple issues",
			steps: []PipelineStep{
				{ID: "", Type: ""},
				{ID: "step1", Type: "inference.chat", DependsOn: []string{"ghost"}},
			},
			wantValid:  false,
			wantIssues: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, issues := ValidateSteps(tt.steps)

			if valid != tt.wantValid {
				t.Errorf("expected valid=%v, got %v (issues: %v)", tt.wantValid, valid, issues)
			}

			hasIssues := len(issues) > 0
			if hasIssues != tt.wantIssues {
				t.Errorf("expected hasIssues=%v, got %v (issues: %v)", tt.wantIssues, hasIssues, issues)
			}
		})
	}
}

func TestValidateSteps_CircularDependency(t *testing.T) {
	// A depends on B, B depends on A → circular
	steps := []PipelineStep{
		{ID: "A", Type: "test", DependsOn: []string{"B"}},
		{ID: "B", Type: "test", DependsOn: []string{"A"}},
	}

	valid, issues := ValidateSteps(steps)
	if valid {
		t.Error("expected invalid for circular dependency")
	}
	if len(issues) == 0 {
		t.Error("expected at least one issue for circular dependency")
	}
}

// --- MockStepExecutor ---

func TestMockStepExecutor_ReturnsExpectedShape(t *testing.T) {
	output, err := MockStepExecutor(context.Background(), "inference.chat", map[string]any{"model": "gpt"})
	if err != nil {
		t.Fatalf("MockStepExecutor returned error: %v", err)
	}
	if output["type"] != "inference.chat" {
		t.Errorf("expected type=inference.chat, got %v", output["type"])
	}
	if output["result"] != "mock_success" {
		t.Errorf("expected result=mock_success, got %v", output["result"])
	}
	inputPassed, ok := output["input"].(map[string]any)
	if !ok {
		t.Fatal("expected input in mock output")
	}
	if inputPassed["model"] != "gpt" {
		t.Errorf("expected input to contain model=gpt, got %v", inputPassed["model"])
	}
}
