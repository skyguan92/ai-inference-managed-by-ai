package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/infra/eventbus"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/pipeline"
)

type PipelineService struct {
	registry *unit.Registry
	store    pipeline.PipelineStore
	executor *pipeline.Executor
	bus      *eventbus.InMemoryEventBus
	runs     map[string]*pendingRun
	mu       sync.Mutex
}

type pendingRun struct {
	result chan *RunWithResults
	ctx    context.Context
	cancel context.CancelFunc
}

func NewPipelineService(registry *unit.Registry, store pipeline.PipelineStore, executor *pipeline.Executor, bus *eventbus.InMemoryEventBus) *PipelineService {
	return &PipelineService{
		registry: registry,
		store:    store,
		executor: executor,
		bus:      bus,
		runs:     make(map[string]*pendingRun),
	}
}

type CreateWithValidationResult struct {
	PipelineID string
	Valid      bool
	Issues     []string
}

func (s *PipelineService) CreateWithValidation(ctx context.Context, name string, steps []pipeline.PipelineStep, config map[string]any) (*CreateWithValidationResult, error) {
	valid, issues := pipeline.ValidateSteps(steps)
	if !valid {
		return &CreateWithValidationResult{
			Valid:  false,
			Issues: issues,
		}, fmt.Errorf("pipeline validation failed: %v", issues)
	}

	createCmd := s.registry.GetCommand("pipeline.create")
	if createCmd == nil {
		return nil, fmt.Errorf("pipeline.create command not found")
	}

	stepsAny := make([]any, len(steps))
	for i, step := range steps {
		stepsAny[i] = map[string]any{
			"id":         step.ID,
			"name":       step.Name,
			"type":       step.Type,
			"input":      step.Input,
			"depends_on": step.DependsOn,
		}
	}

	input := map[string]any{
		"name":  name,
		"steps": stepsAny,
	}
	if config != nil {
		input["config"] = config
	}

	result, err := createCmd.Execute(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("create pipeline: %w", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected create result type")
	}

	pipelineID := getString(resultMap, "pipeline_id")

	s.publishEvent(ctx, "pipeline.created_with_validation", map[string]any{
		"pipeline_id": pipelineID,
		"name":        name,
		"step_count":  len(steps),
	})

	return &CreateWithValidationResult{
		PipelineID: pipelineID,
		Valid:      true,
		Issues:     []string{},
	}, nil
}

type RunResult struct {
	RunID  string
	Status pipeline.RunStatus
}

func (s *PipelineService) RunAsync(ctx context.Context, pipelineID string, input map[string]any) (*RunResult, error) {
	runCmd := s.registry.GetCommand("pipeline.run")
	if runCmd == nil {
		return nil, fmt.Errorf("pipeline.run command not found")
	}

	cmdInput := map[string]any{
		"pipeline_id": pipelineID,
		"async":       true,
	}
	if input != nil {
		cmdInput["input"] = input
	}

	result, err := runCmd.Execute(ctx, cmdInput)
	if err != nil {
		return nil, fmt.Errorf("run pipeline: %w", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected run result type")
	}

	runID := getString(resultMap, "run_id")
	status := pipeline.RunStatus(getString(resultMap, "status"))

	s.publishEvent(ctx, "pipeline.run_started", map[string]any{
		"pipeline_id": pipelineID,
		"run_id":      runID,
		"async":       true,
	})

	return &RunResult{
		RunID:  runID,
		Status: status,
	}, nil
}

func (s *PipelineService) RunSync(ctx context.Context, pipelineID string, input map[string]any) (*RunWithResults, error) {
	p, err := s.store.GetPipeline(ctx, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("get pipeline: %w", err)
	}

	run := &pipeline.PipelineRun{
		ID:          generatePipelineRunID(),
		PipelineID:  pipelineID,
		Status:      pipeline.RunStatusPending,
		Input:       input,
		StepResults: make(map[string]any),
		StartedAt:   time.Now(),
	}

	if err := s.store.CreateRun(ctx, run); err != nil {
		return nil, fmt.Errorf("create run: %w", err)
	}

	p.Status = pipeline.PipelineStatusRunning
	p.UpdatedAt = time.Now().Unix()
	if err := s.store.UpdatePipeline(ctx, p); err != nil {
		return nil, fmt.Errorf("update pipeline status: %w", err)
	}

	runCtx, cancel := context.WithCancel(ctx)
	pending := &pendingRun{
		result: make(chan *RunWithResults, 1),
		ctx:    runCtx,
		cancel: cancel,
	}

	s.mu.Lock()
	s.runs[run.ID] = pending
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.runs, run.ID)
		s.mu.Unlock()
	}()

	if s.executor != nil {
		execErr := s.executor.Execute(runCtx, p, run, input)
		if execErr != nil {
			run.Status = pipeline.RunStatusFailed
			run.Error = execErr.Error()
			now := time.Now()
			run.CompletedAt = &now
			s.store.UpdateRun(ctx, run)
		}
	}

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			updatedRun, err := s.store.GetRun(ctx, run.ID)
			if err != nil {
				return nil, fmt.Errorf("get run status: %w", err)
			}

			if updatedRun.Status == pipeline.RunStatusCompleted ||
				updatedRun.Status == pipeline.RunStatusFailed ||
				updatedRun.Status == pipeline.RunStatusCancelled {
				p.Status = pipeline.PipelineStatusIdle
				p.UpdatedAt = time.Now().Unix()
				s.store.UpdatePipeline(ctx, p)

				return &RunWithResults{
					RunID:       updatedRun.ID,
					PipelineID:  updatedRun.PipelineID,
					Status:      updatedRun.Status,
					StepResults: updatedRun.StepResults,
					Error:       updatedRun.Error,
					StartedAt:   updatedRun.StartedAt,
					CompletedAt: updatedRun.CompletedAt,
				}, nil
			}
		}
	}
}

type RunWithResults struct {
	RunID       string
	PipelineID  string
	Status      pipeline.RunStatus
	StepResults map[string]any
	Error       string
	StartedAt   time.Time
	CompletedAt *time.Time
}

func (s *PipelineService) GetRunWithResults(ctx context.Context, runID string) (*RunWithResults, error) {
	statusQuery := s.registry.GetQuery("pipeline.status")
	if statusQuery == nil {
		return nil, fmt.Errorf("pipeline.status query not found")
	}

	result, err := statusQuery.Execute(ctx, map[string]any{
		"run_id": runID,
	})
	if err != nil {
		return nil, fmt.Errorf("get run status: %w", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected status result type")
	}

	run, err := s.store.GetRun(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("get run: %w", err)
	}

	return &RunWithResults{
		RunID:       runID,
		PipelineID:  run.PipelineID,
		Status:      pipeline.RunStatus(getString(resultMap, "status")),
		StepResults: getMapAny(resultMap, "step_results"),
		Error:       getString(resultMap, "error"),
		StartedAt:   run.StartedAt,
		CompletedAt: run.CompletedAt,
	}, nil
}

type CancelResult struct {
	Success bool
	Message string
}

func (s *PipelineService) CancelRun(ctx context.Context, runID string) (*CancelResult, error) {
	run, err := s.store.GetRun(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("get run: %w", err)
	}

	if run.Status != pipeline.RunStatusPending && run.Status != pipeline.RunStatusRunning {
		return &CancelResult{
			Success: false,
			Message: fmt.Sprintf("run is in status '%s' and cannot be cancelled", run.Status),
		}, nil
	}

	cancelCmd := s.registry.GetCommand("pipeline.cancel")
	if cancelCmd == nil {
		return nil, fmt.Errorf("pipeline.cancel command not found")
	}

	_, err = cancelCmd.Execute(ctx, map[string]any{
		"run_id": runID,
	})
	if err != nil {
		return nil, fmt.Errorf("cancel run: %w", err)
	}

	s.mu.Lock()
	if pending, exists := s.runs[runID]; exists {
		pending.cancel()
	}
	s.mu.Unlock()

	s.publishEvent(ctx, "pipeline.run_cancelled", map[string]any{
		"run_id":      runID,
		"pipeline_id": run.PipelineID,
	})

	return &CancelResult{
		Success: true,
		Message: "run cancelled successfully",
	}, nil
}

type PipelineSummary struct {
	ID        string
	Name      string
	Status    pipeline.PipelineStatus
	StepCount int
	CreatedAt int64
	UpdatedAt int64
}

func (s *PipelineService) ListByStatus(ctx context.Context, status pipeline.PipelineStatus, limit, offset int) ([]PipelineSummary, int, error) {
	listQuery := s.registry.GetQuery("pipeline.list")
	if listQuery == nil {
		return nil, 0, fmt.Errorf("pipeline.list query not found")
	}

	if limit <= 0 {
		limit = 100
	}

	input := map[string]any{
		"limit":  limit,
		"offset": offset,
	}
	if status != "" {
		input["status"] = string(status)
	}

	result, err := listQuery.Execute(ctx, input)
	if err != nil {
		return nil, 0, fmt.Errorf("list pipelines: %w", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, 0, fmt.Errorf("unexpected list result type")
	}

	total := getInt(resultMap, "total")

	var pipelines []map[string]any
	if items, ok := resultMap["pipelines"].([]map[string]any); ok {
		pipelines = items
	} else if items, ok := resultMap["pipelines"].([]any); ok {
		pipelines = make([]map[string]any, len(items))
		for i, item := range items {
			if m, ok := item.(map[string]any); ok {
				pipelines[i] = m
			}
		}
	}

	summaries := make([]PipelineSummary, len(pipelines))
	for i, p := range pipelines {
		summaries[i] = PipelineSummary{
			ID:        getString(p, "id"),
			Name:      getString(p, "name"),
			Status:    pipeline.PipelineStatus(getString(p, "status")),
			CreatedAt: getInt64(p, "created_at"),
			UpdatedAt: getInt64(p, "updated_at"),
		}
	}

	return summaries, total, nil
}

type DefinitionValidationResult struct {
	Valid          bool
	Issues         []string
	HasCycle       bool
	MissingDeps    []string
	DuplicateSteps []string
}

func (s *PipelineService) ValidateDefinition(ctx context.Context, steps []pipeline.PipelineStep) (*DefinitionValidationResult, error) {
	validateQuery := s.registry.GetQuery("pipeline.validate")
	if validateQuery == nil {
		return nil, fmt.Errorf("pipeline.validate query not found")
	}

	stepsAny := make([]any, len(steps))
	for i, step := range steps {
		stepsAny[i] = map[string]any{
			"id":         step.ID,
			"name":       step.Name,
			"type":       step.Type,
			"input":      step.Input,
			"depends_on": step.DependsOn,
		}
	}

	result, err := validateQuery.Execute(ctx, map[string]any{
		"steps": stepsAny,
	})
	if err != nil {
		return nil, fmt.Errorf("validate definition: %w", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected validation result type")
	}

	valid := getBool(resultMap, "valid")
	var issues []string
	if v, ok := resultMap["issues"].([]string); ok {
		issues = v
	} else if v, ok := resultMap["issues"].([]any); ok {
		issues = make([]string, len(v))
		for i, item := range v {
			if s, ok := item.(string); ok {
				issues[i] = s
			}
		}
	}

	hasCycle := false
	missingDeps := []string{}
	duplicateSteps := []string{}

	for _, issue := range issues {
		if containsAny(issue, "circular", "cycle") {
			hasCycle = true
		}
		if containsAny(issue, "non-existent", "not found") && containsAny(issue, "depend") {
			missingDeps = append(missingDeps, issue)
		}
		if containsAny(issue, "duplicate") {
			duplicateSteps = append(duplicateSteps, issue)
		}
	}

	return &DefinitionValidationResult{
		Valid:          valid,
		Issues:         issues,
		HasCycle:       hasCycle,
		MissingDeps:    missingDeps,
		DuplicateSteps: duplicateSteps,
	}, nil
}

type PipelineDeleteResult struct {
	Success    bool
	Cancelled  bool
	RunCount   int
	CleanedRun string
}

func (s *PipelineService) DeleteWithCleanup(ctx context.Context, pipelineID string, force bool) (*PipelineDeleteResult, error) {
	p, err := s.store.GetPipeline(ctx, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("get pipeline: %w", err)
	}

	runs, err := s.store.ListRuns(ctx, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("list runs: %w", err)
	}

	var activeRuns []string
	for _, run := range runs {
		if run.Status == pipeline.RunStatusRunning || run.Status == pipeline.RunStatusPending {
			activeRuns = append(activeRuns, run.ID)
		}
	}

	if len(activeRuns) > 0 && !force {
		return nil, fmt.Errorf("pipeline has %d active runs, use force=true to cancel and delete", len(activeRuns))
	}

	cancelled := false
	for _, runID := range activeRuns {
		_, cancelErr := s.CancelRun(ctx, runID)
		if cancelErr == nil {
			cancelled = true
		}
	}

	deleteCmd := s.registry.GetCommand("pipeline.delete")
	if deleteCmd == nil {
		return nil, fmt.Errorf("pipeline.delete command not found")
	}

	_, err = deleteCmd.Execute(ctx, map[string]any{
		"pipeline_id": pipelineID,
	})
	if err != nil {
		return nil, fmt.Errorf("delete pipeline: %w", err)
	}

	s.publishEvent(ctx, "pipeline.deleted_with_cleanup", map[string]any{
		"pipeline_id":   pipelineID,
		"run_count":     len(runs),
		"cancelled":     cancelled,
		"pipeline_name": p.Name,
	})

	return &PipelineDeleteResult{
		Success:   true,
		Cancelled: cancelled,
		RunCount:  len(runs),
	}, nil
}

func (s *PipelineService) GetPipeline(ctx context.Context, pipelineID string) (*pipeline.Pipeline, error) {
	return s.store.GetPipeline(ctx, pipelineID)
}

func (s *PipelineService) ListRuns(ctx context.Context, pipelineID string) ([]pipeline.PipelineRun, error) {
	return s.store.ListRuns(ctx, pipelineID)
}

func (s *PipelineService) publishEvent(ctx context.Context, eventType string, payload any) {
	if s.bus == nil {
		return
	}

	evt := &pipelineEvent{
		eventType: eventType,
		domain:    "pipeline",
		payload:   payload,
	}

	_ = s.bus.Publish(evt)
}

type pipelineEvent struct {
	eventType string
	domain    string
	payload   any
}

func (e *pipelineEvent) Type() string          { return e.eventType }
func (e *pipelineEvent) Domain() string        { return e.domain }
func (e *pipelineEvent) Payload() any          { return e.payload }
func (e *pipelineEvent) Timestamp() time.Time  { return time.Now() }
func (e *pipelineEvent) CorrelationID() string { return "" }

func generatePipelineRunID() string {
	return fmt.Sprintf("run-%d", time.Now().UnixNano())
}

func getMapAny(m map[string]any, key string) map[string]any {
	if v, ok := m[key]; ok {
		if m, ok := v.(map[string]any); ok {
			return m
		}
	}
	return nil
}

func containsAny(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}
