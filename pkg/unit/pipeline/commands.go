package pipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/ptrs"
)

type CreateCommand struct {
	store    PipelineStore
	executor *Executor
	events   unit.EventPublisher
}

func NewCreateCommand(store PipelineStore, executor *Executor) *CreateCommand {
	return &CreateCommand{store: store, executor: executor}
}

func NewCreateCommandWithEvents(store PipelineStore, executor *Executor, events unit.EventPublisher) *CreateCommand {
	return &CreateCommand{store: store, executor: executor, events: events}
}

func (c *CreateCommand) Name() string {
	return "pipeline.create"
}

func (c *CreateCommand) Domain() string {
	return "pipeline"
}

func (c *CreateCommand) Description() string {
	return "Create a new pipeline with steps"
}

func (c *CreateCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"name": {
				Name: "name",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Pipeline name",
					MinLength:   ptrs.Int(1),
				},
			},
			"steps": {
				Name: "steps",
				Schema: unit.Schema{
					Type:        "array",
					Description: "Pipeline steps",
					Items: &unit.Schema{
						Type: "object",
						Properties: map[string]unit.Field{
							"id":         {Name: "id", Schema: unit.Schema{Type: "string"}},
							"name":       {Name: "name", Schema: unit.Schema{Type: "string"}},
							"type":       {Name: "type", Schema: unit.Schema{Type: "string"}},
							"input":      {Name: "input", Schema: unit.Schema{Type: "object"}},
							"depends_on": {Name: "depends_on", Schema: unit.Schema{Type: "array", Items: &unit.Schema{Type: "string"}}},
						},
					},
				},
			},
			"config": {
				Name: "config",
				Schema: unit.Schema{
					Type:        "object",
					Description: "Optional pipeline configuration",
				},
			},
		},
		Required: []string{"name", "steps"},
	}
}

func (c *CreateCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"pipeline_id": {
				Name:   "pipeline_id",
				Schema: unit.Schema{Type: "string"},
			},
		},
	}
}

func (c *CreateCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input: map[string]any{
				"name": "chat-pipeline",
				"steps": []map[string]any{
					{"id": "step1", "name": "Chat", "type": "inference.chat", "input": map[string]any{"model": "llama3"}},
				},
			},
			Output:      map[string]any{"pipeline_id": "pipe-abc123"},
			Description: "Create a simple chat pipeline",
		},
	}
}

func (c *CreateCommand) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(c.events, c.Domain(), c.Name())
	ec.PublishStarted(input)

	if c.store == nil {
		err := ErrStoreNotSet
		ec.PublishFailed(err)
		return nil, err
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		err := fmt.Errorf("invalid input type: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	name, _ := inputMap["name"].(string)
	if name == "" {
		err := fmt.Errorf("name is required: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	stepsRaw, ok := inputMap["steps"].([]any)
	if !ok || len(stepsRaw) == 0 {
		err := fmt.Errorf("steps is required and must be non-empty: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	steps := make([]PipelineStep, len(stepsRaw))
	for i, s := range stepsRaw {
		stepMap, ok := s.(map[string]any)
		if !ok {
			err := fmt.Errorf("invalid step at index %d: %w", i, ErrInvalidInput)
			ec.PublishFailed(err)
			return nil, err
		}

		step := PipelineStep{
			ID:    fmt.Sprintf("%v", stepMap["id"]),
			Name:  fmt.Sprintf("%v", stepMap["name"]),
			Type:  fmt.Sprintf("%v", stepMap["type"]),
			Input: make(map[string]any),
		}

		if input, ok := stepMap["input"].(map[string]any); ok {
			step.Input = input
		}

		if deps, ok := stepMap["depends_on"].([]any); ok {
			step.DependsOn = make([]string, len(deps))
			for j, d := range deps {
				step.DependsOn[j] = fmt.Sprintf("%v", d)
			}
		}

		steps[i] = step
	}

	if valid, issues := ValidateSteps(steps); !valid {
		err := fmt.Errorf("invalid steps: %v: %w", issues, ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	config, _ := inputMap["config"].(map[string]any)

	now := time.Now().Unix()
	pipeline := &Pipeline{
		ID:        generateID("pipe"),
		Name:      name,
		Steps:     steps,
		Status:    PipelineStatusIdle,
		Config:    config,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := c.store.CreatePipeline(ctx, pipeline); err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("create pipeline: %w", err)
	}

	output := map[string]any{"pipeline_id": pipeline.ID}
	ec.PublishCompleted(output)
	return output, nil
}

type DeleteCommand struct {
	store  PipelineStore
	events unit.EventPublisher
}

func NewDeleteCommand(store PipelineStore) *DeleteCommand {
	return &DeleteCommand{store: store}
}

func NewDeleteCommandWithEvents(store PipelineStore, events unit.EventPublisher) *DeleteCommand {
	return &DeleteCommand{store: store, events: events}
}

func (c *DeleteCommand) Name() string {
	return "pipeline.delete"
}

func (c *DeleteCommand) Domain() string {
	return "pipeline"
}

func (c *DeleteCommand) Description() string {
	return "Delete a pipeline"
}

func (c *DeleteCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"pipeline_id": {
				Name: "pipeline_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Pipeline ID to delete",
					MinLength:   ptrs.Int(1),
				},
			},
		},
		Required: []string{"pipeline_id"},
	}
}

func (c *DeleteCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"success": {
				Name:   "success",
				Schema: unit.Schema{Type: "boolean"},
			},
		},
	}
}

func (c *DeleteCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"pipeline_id": "pipe-abc123"},
			Output:      map[string]any{"success": true},
			Description: "Delete a pipeline",
		},
	}
}

func (c *DeleteCommand) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(c.events, c.Domain(), c.Name())
	ec.PublishStarted(input)

	if c.store == nil {
		err := ErrStoreNotSet
		ec.PublishFailed(err)
		return nil, err
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		err := fmt.Errorf("invalid input type: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	pipelineID, _ := inputMap["pipeline_id"].(string)
	if pipelineID == "" {
		err := fmt.Errorf("pipeline_id is required: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	pipeline, err := c.store.GetPipeline(ctx, pipelineID)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("get pipeline %s: %w", pipelineID, err)
	}

	if pipeline.Status == PipelineStatusRunning {
		ec.PublishFailed(ErrPipelineRunning)
		return nil, ErrPipelineRunning
	}

	if err := c.store.DeletePipeline(ctx, pipelineID); err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("delete pipeline %s: %w", pipelineID, err)
	}

	output := map[string]any{"success": true}
	ec.PublishCompleted(output)
	return output, nil
}

type RunCommand struct {
	store    PipelineStore
	executor *Executor
	events   unit.EventPublisher
}

func NewRunCommand(store PipelineStore, executor *Executor) *RunCommand {
	return &RunCommand{store: store, executor: executor}
}

func NewRunCommandWithEvents(store PipelineStore, executor *Executor, events unit.EventPublisher) *RunCommand {
	return &RunCommand{store: store, executor: executor, events: events}
}

func (c *RunCommand) Name() string {
	return "pipeline.run"
}

func (c *RunCommand) Domain() string {
	return "pipeline"
}

func (c *RunCommand) Description() string {
	return "Run a pipeline"
}

func (c *RunCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"pipeline_id": {
				Name: "pipeline_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Pipeline ID to run",
					MinLength:   ptrs.Int(1),
				},
			},
			"input": {
				Name: "input",
				Schema: unit.Schema{
					Type:        "object",
					Description: "Input data for the pipeline",
				},
			},
			"async": {
				Name: "async",
				Schema: unit.Schema{
					Type:        "boolean",
					Description: "Run asynchronously",
					Default:     true,
				},
			},
		},
		Required: []string{"pipeline_id"},
	}
}

func (c *RunCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"run_id": {
				Name:   "run_id",
				Schema: unit.Schema{Type: "string"},
			},
			"status": {
				Name:   "status",
				Schema: unit.Schema{Type: "string"},
			},
		},
	}
}

func (c *RunCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"pipeline_id": "pipe-abc123"},
			Output:      map[string]any{"run_id": "run-xyz789", "status": "pending"},
			Description: "Run a pipeline",
		},
		{
			Input:       map[string]any{"pipeline_id": "pipe-abc123", "input": map[string]any{"query": "hello"}},
			Output:      map[string]any{"run_id": "run-xyz789", "status": "pending"},
			Description: "Run a pipeline with input",
		},
	}
}

func (c *RunCommand) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(c.events, c.Domain(), c.Name())
	ec.PublishStarted(input)

	if c.store == nil {
		err := ErrStoreNotSet
		ec.PublishFailed(err)
		return nil, err
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		err := fmt.Errorf("invalid input type: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	pipelineID, _ := inputMap["pipeline_id"].(string)
	if pipelineID == "" {
		err := fmt.Errorf("pipeline_id is required: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	pipeline, err := c.store.GetPipeline(ctx, pipelineID)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("get pipeline %s: %w", pipelineID, err)
	}

	runInput, _ := inputMap["input"].(map[string]any)

	run := &PipelineRun{
		ID:          generateID("run"),
		PipelineID:  pipelineID,
		Status:      RunStatusPending,
		Input:       runInput,
		StepResults: make(map[string]any),
		StartedAt:   time.Now(),
	}

	if err := c.store.CreateRun(ctx, run); err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("create run: %w", err)
	}

	pipeline.Status = PipelineStatusRunning
	pipeline.UpdatedAt = time.Now().Unix()
	if err := c.store.UpdatePipeline(ctx, pipeline); err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("update pipeline status: %w", err)
	}

	if c.executor != nil {
		if err := c.executor.Execute(ctx, pipeline, run, runInput); err != nil {
			run.Status = RunStatusFailed
			run.Error = err.Error()
			_ = c.store.UpdateRun(ctx, run)
			ec.PublishFailed(err)
			return nil, fmt.Errorf("execute pipeline: %w", err)
		}
	}

	output := map[string]any{"run_id": run.ID, "status": string(RunStatusRunning)}
	ec.PublishCompleted(output)
	return output, nil
}

type CancelCommand struct {
	store    PipelineStore
	executor *Executor
	events   unit.EventPublisher
}

func NewCancelCommand(store PipelineStore, executor *Executor) *CancelCommand {
	return &CancelCommand{store: store, executor: executor}
}

func NewCancelCommandWithEvents(store PipelineStore, executor *Executor, events unit.EventPublisher) *CancelCommand {
	return &CancelCommand{store: store, executor: executor, events: events}
}

func (c *CancelCommand) Name() string {
	return "pipeline.cancel"
}

func (c *CancelCommand) Domain() string {
	return "pipeline"
}

func (c *CancelCommand) Description() string {
	return "Cancel a running pipeline"
}

func (c *CancelCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"run_id": {
				Name: "run_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Run ID to cancel",
					MinLength:   ptrs.Int(1),
				},
			},
		},
		Required: []string{"run_id"},
	}
}

func (c *CancelCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"success": {
				Name:   "success",
				Schema: unit.Schema{Type: "boolean"},
			},
		},
	}
}

func (c *CancelCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"run_id": "run-xyz789"},
			Output:      map[string]any{"success": true},
			Description: "Cancel a running pipeline",
		},
	}
}

func (c *CancelCommand) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(c.events, c.Domain(), c.Name())
	ec.PublishStarted(input)

	if c.store == nil {
		err := ErrStoreNotSet
		ec.PublishFailed(err)
		return nil, err
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		err := fmt.Errorf("invalid input type: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	runID, _ := inputMap["run_id"].(string)
	if runID == "" {
		err := fmt.Errorf("run_id is required: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	run, err := c.store.GetRun(ctx, runID)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("get run %s: %w", runID, err)
	}

	switch run.Status {
	case RunStatusPending, RunStatusRunning:
		// cancellable
	case RunStatusCompleted, RunStatusCancelled, RunStatusFailed:
		// already in a terminal state — cancel is a no-op, return success (idempotent)
		output := map[string]any{"success": true}
		ec.PublishCompleted(output)
		return output, nil
	default:
		ec.PublishFailed(ErrRunNotCancellable)
		return nil, ErrRunNotCancellable
	}

	cancelled := false
	if c.executor != nil {
		cancelled = c.executor.Cancel(runID)
	}

	if !cancelled {
		run.Status = RunStatusCancelled
		now := time.Now()
		run.CompletedAt = &now
		if err := c.store.UpdateRun(ctx, run); err != nil {
			ec.PublishFailed(err)
			return nil, fmt.Errorf("update run: %w", err)
		}
	}

	pipeline, err := c.store.GetPipeline(ctx, run.PipelineID)
	if err == nil {
		pipeline.Status = PipelineStatusIdle
		pipeline.UpdatedAt = time.Now().Unix()
		_ = c.store.UpdatePipeline(ctx, pipeline)
	}

	output := map[string]any{"success": true}
	ec.PublishCompleted(output)
	return output, nil
}
