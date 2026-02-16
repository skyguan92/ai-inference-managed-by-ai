package workflow

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type UnitExecutor func(ctx context.Context, unitType string, input map[string]any) (map[string]any, error)

type WorkflowEngine struct {
	registry    *unit.Registry
	store       WorkflowStore
	validator   *DAGValidator
	resolver    *VariableResolver
	executor    UnitExecutor
	runningRuns map[string]context.CancelFunc
	mu          sync.Mutex
}

func NewWorkflowEngine(registry *unit.Registry, store WorkflowStore, executor UnitExecutor) *WorkflowEngine {
	return &WorkflowEngine{
		registry:    registry,
		store:       store,
		validator:   NewDAGValidator(),
		resolver:    NewVariableResolver(),
		executor:    executor,
		runningRuns: make(map[string]context.CancelFunc),
	}
}

func NewWorkflowEngineWithDefaults(store WorkflowStore) *WorkflowEngine {
	return NewWorkflowEngine(unit.NewRegistry(), store, nil)
}

func (e *WorkflowEngine) Execute(ctx context.Context, def *WorkflowDef, input map[string]any) (*ExecutionResult, error) {
	validation := e.validator.Validate(def)
	if !validation.Valid {
		return nil, fmt.Errorf("workflow validation failed: %v", validation.Errors)
	}

	sortedSteps, err := TopologicalSort(def)
	if err != nil {
		return nil, fmt.Errorf("failed to sort steps: %w", err)
	}

	runID := generateRunID()
	startTime := time.Now()

	result := &ExecutionResult{
		WorkflowID:  def.Name,
		RunID:       runID,
		Status:      ExecutionStatusRunning,
		StepResults: make(map[string]StepResult),
		StartedAt:   startTime,
	}

	execCtx := &ExecutionContext{
		Input:  input,
		Config: def.Config,
		Steps:  make(map[string]map[string]any),
	}

	for _, step := range sortedSteps {
		stepResult := e.executeStep(ctx, step, execCtx)
		result.StepResults[step.ID] = stepResult

		if stepResult.Status == ExecutionStatusFailed {
			result.Status = ExecutionStatusFailed
			result.Error = stepResult.Error
			now := time.Now()
			result.CompletedAt = &now
			result.Duration = time.Since(startTime)
			_ = e.store.SaveExecution(ctx, result)
			return result, fmt.Errorf("step '%s' failed: %s", step.ID, stepResult.Error)
		}

		execCtx.Steps[step.ID] = stepResult.Output
	}

	output, err := e.resolver.ResolveOutput(def.Output, execCtx)
	if err != nil {
		result.Status = ExecutionStatusFailed
		result.Error = fmt.Sprintf("failed to resolve output: %v", err)
		now := time.Now()
		result.CompletedAt = &now
		result.Duration = time.Since(startTime)
		_ = e.store.SaveExecution(ctx, result)
		return result, fmt.Errorf("failed to resolve output: %w", err)
	}

	result.Output = output
	result.Status = ExecutionStatusCompleted
	now := time.Now()
	result.CompletedAt = &now
	result.Duration = time.Since(startTime)

	_ = e.store.SaveExecution(ctx, result)

	return result, nil
}

func (e *WorkflowEngine) executeStep(ctx context.Context, step *WorkflowStep, execCtx *ExecutionContext) StepResult {
	startTime := time.Now()
	result := StepResult{
		StepID:    step.ID,
		Status:    ExecutionStatusRunning,
		StartedAt: startTime,
	}

	resolvedInput, err := e.resolver.ResolveStepInput(step, execCtx)
	if err != nil {
		result.Status = ExecutionStatusFailed
		result.Error = err.Error()
		now := time.Now()
		result.EndedAt = &now
		result.Duration = time.Since(startTime)
		return result
	}

	maxAttempts := 1
	delaySeconds := 0
	if step.Retry != nil {
		maxAttempts = step.Retry.MaxAttempts
		delaySeconds = step.Retry.DelaySeconds
	}

	var output map[string]any
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		output, lastErr = e.executeUnit(ctx, step.Type, resolvedInput)

		if lastErr == nil {
			result.Status = ExecutionStatusCompleted
			result.Output = output
			now := time.Now()
			result.EndedAt = &now
			result.Duration = time.Since(startTime)
			return result
		}

		if attempt < maxAttempts {
			time.Sleep(time.Duration(delaySeconds) * time.Second)
		}
	}

	if step.OnFailure == "continue" {
		result.Status = ExecutionStatusCompleted
		result.Output = map[string]any{"error": lastErr.Error()}
	} else {
		result.Status = ExecutionStatusFailed
		result.Error = lastErr.Error()
	}

	now := time.Now()
	result.EndedAt = &now
	result.Duration = time.Since(startTime)
	return result
}

func (e *WorkflowEngine) executeUnit(ctx context.Context, unitType string, input map[string]any) (map[string]any, error) {
	if e.executor != nil {
		return e.executor(ctx, unitType, input)
	}

	if e.registry == nil {
		return nil, fmt.Errorf("no executor or registry configured")
	}

	cmd := e.registry.GetCommand(unitType)
	if cmd != nil {
		output, err := cmd.Execute(ctx, input)
		if err != nil {
			return nil, err
		}
		if m, ok := output.(map[string]any); ok {
			return m, nil
		}
		return map[string]any{"result": output}, nil
	}

	query := e.registry.GetQuery(unitType)
	if query != nil {
		output, err := query.Execute(ctx, input)
		if err != nil {
			return nil, err
		}
		if m, ok := output.(map[string]any); ok {
			return m, nil
		}
		return map[string]any{"result": output}, nil
	}

	return nil, fmt.Errorf("unit '%s' not found in registry", unitType)
}

func (e *WorkflowEngine) ExecuteAsync(ctx context.Context, def *WorkflowDef, input map[string]any) (*ExecutionResult, error) {
	validation := e.validator.Validate(def)
	if !validation.Valid {
		return nil, fmt.Errorf("workflow validation failed: %v", validation.Errors)
	}

	runID := generateRunID()
	startTime := time.Now()

	result := &ExecutionResult{
		WorkflowID:  def.Name,
		RunID:       runID,
		Status:      ExecutionStatusRunning,
		StepResults: make(map[string]StepResult),
		StartedAt:   startTime,
	}

	asyncCtx, cancel := context.WithCancel(context.Background())
	e.mu.Lock()
	e.runningRuns[runID] = cancel
	e.mu.Unlock()

	go func() {
		defer func() {
			e.mu.Lock()
			delete(e.runningRuns, runID)
			e.mu.Unlock()
			cancel()
		}()

		execResult, err := e.Execute(asyncCtx, def, input)
		if err != nil {
			result.Status = ExecutionStatusFailed
			result.Error = err.Error()
		} else {
			result.Status = execResult.Status
			result.Output = execResult.Output
			result.StepResults = execResult.StepResults
		}

		now := time.Now()
		result.CompletedAt = &now
		result.Duration = time.Since(startTime)
		_ = e.store.SaveExecution(context.Background(), result)
	}()

	return result, nil
}

func (e *WorkflowEngine) Cancel(runID string) bool {
	e.mu.Lock()
	cancel, exists := e.runningRuns[runID]
	e.mu.Unlock()

	if !exists {
		return false
	}

	cancel()
	return true
}

func (e *WorkflowEngine) IsRunning(runID string) bool {
	e.mu.Lock()
	_, exists := e.runningRuns[runID]
	e.mu.Unlock()
	return exists
}

func (e *WorkflowEngine) GetExecution(ctx context.Context, runID string) (*ExecutionResult, error) {
	return e.store.GetExecution(ctx, runID)
}

func (e *WorkflowEngine) ListExecutions(ctx context.Context, workflowName string, limit int) ([]*ExecutionResult, error) {
	return e.store.ListExecutions(ctx, workflowName, limit)
}

func (e *WorkflowEngine) RegisterWorkflow(ctx context.Context, def *WorkflowDef) error {
	validation := e.validator.Validate(def)
	if !validation.Valid {
		return fmt.Errorf("workflow validation failed: %v", validation.Errors)
	}
	return e.store.SaveWorkflow(ctx, def)
}

func (e *WorkflowEngine) GetWorkflow(ctx context.Context, name string) (*WorkflowDef, error) {
	return e.store.GetWorkflow(ctx, name)
}

func (e *WorkflowEngine) ListWorkflows(ctx context.Context) ([]*WorkflowDef, error) {
	return e.store.ListWorkflows(ctx)
}

func (e *WorkflowEngine) DeleteWorkflow(ctx context.Context, name string) error {
	return e.store.DeleteWorkflow(ctx, name)
}

func generateRunID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return "run_" + hex.EncodeToString(b)
}
