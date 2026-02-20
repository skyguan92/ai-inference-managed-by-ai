package pipeline

import (
	"context"
	"sync"
	"time"
)

type StepExecutor func(ctx context.Context, stepType string, input map[string]any) (map[string]any, error)

type Executor struct {
	store        PipelineStore
	stepExecutor StepExecutor
	runningRuns  map[string]context.CancelFunc
	mu           sync.Mutex
}

func NewExecutor(store PipelineStore, stepExecutor StepExecutor) *Executor {
	return &Executor{
		store:        store,
		stepExecutor: stepExecutor,
		runningRuns:  make(map[string]context.CancelFunc),
	}
}

func (e *Executor) Execute(ctx context.Context, pipeline *Pipeline, run *PipelineRun, input map[string]any) error {
	run.Status = RunStatusRunning
	run.StartedAt = time.Now()
	run.StepResults = make(map[string]any)

	if err := e.store.UpdateRun(ctx, run); err != nil {
		return err
	}

	runCtx, cancel := context.WithCancel(context.Background())
	e.mu.Lock()
	e.runningRuns[run.ID] = cancel
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		delete(e.runningRuns, run.ID)
		e.mu.Unlock()
	}()

	go func() {
		defer cancel()

		executedSteps := make(map[string]map[string]any)

		for _, step := range pipeline.Steps {
			select {
			case <-runCtx.Done():
				run.Status = RunStatusCancelled
				now := time.Now()
				run.CompletedAt = &now
				_ = e.store.UpdateRun(context.Background(), run)
				return
			default:
			}

			for _, dep := range step.DependsOn {
				if _, ok := executedSteps[dep]; !ok {
					run.Status = RunStatusFailed
					run.Error = "dependency not satisfied: " + dep
					now := time.Now()
					run.CompletedAt = &now
					_ = e.store.UpdateRun(context.Background(), run)
					return
				}
			}

			stepInput := make(map[string]any)
			for k, v := range step.Input {
				stepInput[k] = v
			}
			for k, v := range input {
				stepInput[k] = v
			}

			var output map[string]any
			var err error

			if e.stepExecutor != nil {
				output, err = e.stepExecutor(runCtx, step.Type, stepInput)
			} else {
				output = map[string]any{"mock": true, "step": step.ID}
			}

			if err != nil {
				run.Status = RunStatusFailed
				run.Error = "step " + step.ID + " failed: " + err.Error()
				now := time.Now()
				run.CompletedAt = &now
				_ = e.store.UpdateRun(context.Background(), run)
				return
			}

			executedSteps[step.ID] = output
			run.StepResults[step.ID] = output
		}

		run.Status = RunStatusCompleted
		now := time.Now()
		run.CompletedAt = &now
		_ = e.store.UpdateRun(context.Background(), run)
	}()

	return nil
}

func (e *Executor) Cancel(runID string) bool {
	e.mu.Lock()
	cancel, exists := e.runningRuns[runID]
	e.mu.Unlock()

	if !exists {
		return false
	}

	cancel()
	return true
}

func (e *Executor) IsRunning(runID string) bool {
	e.mu.Lock()
	_, exists := e.runningRuns[runID]
	e.mu.Unlock()
	return exists
}

func ValidateSteps(steps []PipelineStep) (bool, []string) {
	var issues []string
	stepIDs := make(map[string]bool)

	for _, step := range steps {
		if step.ID == "" {
			issues = append(issues, "step has empty ID")
		}
		if step.Type == "" {
			issues = append(issues, "step "+step.ID+" has empty type")
		}
		if stepIDs[step.ID] {
			issues = append(issues, "duplicate step ID: "+step.ID)
		}
		stepIDs[step.ID] = true
	}

	for _, step := range steps {
		for _, dep := range step.DependsOn {
			if !stepIDs[dep] {
				issues = append(issues, "step "+step.ID+" depends on non-existent step: "+dep)
			}
		}
	}

	visited := make(map[string]bool)
	var visit func(string) bool
	visit = func(stepID string) bool {
		if visited[stepID] {
			return true
		}
		visited[stepID] = true
		for _, step := range steps {
			if step.ID == stepID {
				for _, dep := range step.DependsOn {
					if visit(dep) {
						issues = append(issues, "circular dependency detected involving step: "+stepID)
						return true
					}
				}
			}
		}
		visited[stepID] = false
		return false
	}

	for _, step := range steps {
		visited = make(map[string]bool)
		visit(step.ID)
	}

	return len(issues) == 0, issues
}

func MockStepExecutor(ctx context.Context, stepType string, input map[string]any) (map[string]any, error) {
	return map[string]any{
		"type":   stepType,
		"input":  input,
		"result": "mock_success",
	}, nil
}
