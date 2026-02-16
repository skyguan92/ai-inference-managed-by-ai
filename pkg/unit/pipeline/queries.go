package pipeline

import (
	"context"
	"fmt"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type GetQuery struct {
	store PipelineStore
}

func NewGetQuery(store PipelineStore) *GetQuery {
	return &GetQuery{store: store}
}

func (q *GetQuery) Name() string {
	return "pipeline.get"
}

func (q *GetQuery) Domain() string {
	return "pipeline"
}

func (q *GetQuery) Description() string {
	return "Get detailed pipeline information"
}

func (q *GetQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"pipeline_id": {
				Name: "pipeline_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Pipeline ID",
				},
			},
		},
		Required: []string{"pipeline_id"},
	}
}

func (q *GetQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"id":   {Name: "id", Schema: unit.Schema{Type: "string"}},
			"name": {Name: "name", Schema: unit.Schema{Type: "string"}},
			"steps": {
				Name: "steps",
				Schema: unit.Schema{
					Type: "array",
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
			"status": {Name: "status", Schema: unit.Schema{Type: "string"}},
			"config": {Name: "config", Schema: unit.Schema{Type: "object"}},
		},
	}
}

func (q *GetQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"pipeline_id": "pipe-abc123"},
			Output:      map[string]any{"id": "pipe-abc123", "name": "chat-pipeline", "status": "idle"},
			Description: "Get pipeline details",
		},
	}
}

func (q *GetQuery) Execute(ctx context.Context, input any) (any, error) {
	if q.store == nil {
		return nil, ErrStoreNotSet
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type: %w", ErrInvalidInput)
	}

	pipelineID, _ := inputMap["pipeline_id"].(string)
	if pipelineID == "" {
		return nil, fmt.Errorf("pipeline_id is required: %w", ErrInvalidInput)
	}

	pipeline, err := q.store.GetPipeline(ctx, pipelineID)
	if err != nil {
		return nil, fmt.Errorf("get pipeline %s: %w", pipelineID, err)
	}

	return map[string]any{
		"id":     pipeline.ID,
		"name":   pipeline.Name,
		"steps":  pipeline.Steps,
		"status": string(pipeline.Status),
		"config": pipeline.Config,
	}, nil
}

type ListQuery struct {
	store PipelineStore
}

func NewListQuery(store PipelineStore) *ListQuery {
	return &ListQuery{store: store}
}

func (q *ListQuery) Name() string {
	return "pipeline.list"
}

func (q *ListQuery) Domain() string {
	return "pipeline"
}

func (q *ListQuery) Description() string {
	return "List all pipelines"
}

func (q *ListQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"status": {
				Name: "status",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Filter by status",
					Enum:        []any{string(PipelineStatusIdle), string(PipelineStatusRunning), string(PipelineStatusPaused), string(PipelineStatusError)},
				},
			},
			"limit": {
				Name: "limit",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Maximum number of results",
					Min:         ptrFloat(1),
					Max:         ptrFloat(100),
				},
			},
			"offset": {
				Name: "offset",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Offset for pagination",
					Min:         ptrFloat(0),
				},
			},
		},
	}
}

func (q *ListQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"pipelines": {
				Name: "pipelines",
				Schema: unit.Schema{
					Type: "array",
					Items: &unit.Schema{
						Type: "object",
						Properties: map[string]unit.Field{
							"id":     {Name: "id", Schema: unit.Schema{Type: "string"}},
							"name":   {Name: "name", Schema: unit.Schema{Type: "string"}},
							"status": {Name: "status", Schema: unit.Schema{Type: "string"}},
						},
					},
				},
			},
			"total": {
				Name:   "total",
				Schema: unit.Schema{Type: "number"},
			},
		},
	}
}

func (q *ListQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{},
			Output:      map[string]any{"pipelines": []map[string]any{{"id": "pipe-abc123", "name": "chat-pipeline", "status": "idle"}}, "total": 1},
			Description: "List all pipelines",
		},
	}
}

func (q *ListQuery) Execute(ctx context.Context, input any) (any, error) {
	if q.store == nil {
		return nil, ErrStoreNotSet
	}

	inputMap, _ := input.(map[string]any)

	filter := PipelineFilter{
		Limit:  100,
		Offset: 0,
	}

	if s, ok := inputMap["status"].(string); ok && s != "" {
		filter.Status = PipelineStatus(s)
	}
	if limit, ok := toInt(inputMap["limit"]); ok && limit > 0 {
		filter.Limit = limit
	}
	if offset, ok := toInt(inputMap["offset"]); ok && offset >= 0 {
		filter.Offset = offset
	}

	pipelines, total, err := q.store.ListPipelines(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list pipelines: %w", err)
	}

	items := make([]map[string]any, len(pipelines))
	for i, p := range pipelines {
		items[i] = map[string]any{
			"id":     p.ID,
			"name":   p.Name,
			"status": string(p.Status),
		}
	}

	return map[string]any{
		"pipelines": items,
		"total":     total,
	}, nil
}

type StatusQuery struct {
	store PipelineStore
}

func NewStatusQuery(store PipelineStore) *StatusQuery {
	return &StatusQuery{store: store}
}

func (q *StatusQuery) Name() string {
	return "pipeline.status"
}

func (q *StatusQuery) Domain() string {
	return "pipeline"
}

func (q *StatusQuery) Description() string {
	return "Get pipeline run status"
}

func (q *StatusQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"run_id": {
				Name: "run_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Run ID",
				},
			},
		},
		Required: []string{"run_id"},
	}
}

func (q *StatusQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"status":       {Name: "status", Schema: unit.Schema{Type: "string"}},
			"step_results": {Name: "step_results", Schema: unit.Schema{Type: "object"}},
			"error":        {Name: "error", Schema: unit.Schema{Type: "string"}},
		},
	}
}

func (q *StatusQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"run_id": "run-xyz789"},
			Output:      map[string]any{"status": "completed", "step_results": map[string]any{"step1": map[string]any{"result": "ok"}}},
			Description: "Get run status",
		},
	}
}

func (q *StatusQuery) Execute(ctx context.Context, input any) (any, error) {
	if q.store == nil {
		return nil, ErrStoreNotSet
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type: %w", ErrInvalidInput)
	}

	runID, _ := inputMap["run_id"].(string)
	if runID == "" {
		return nil, fmt.Errorf("run_id is required: %w", ErrInvalidInput)
	}

	run, err := q.store.GetRun(ctx, runID)
	if err != nil {
		return nil, fmt.Errorf("get run %s: %w", runID, err)
	}

	result := map[string]any{
		"status":       string(run.Status),
		"step_results": run.StepResults,
	}

	if run.Error != "" {
		result["error"] = run.Error
	}

	return result, nil
}

type ValidateQuery struct{}

func NewValidateQuery() *ValidateQuery {
	return &ValidateQuery{}
}

func (q *ValidateQuery) Name() string {
	return "pipeline.validate"
}

func (q *ValidateQuery) Domain() string {
	return "pipeline"
}

func (q *ValidateQuery) Description() string {
	return "Validate pipeline step definitions"
}

func (q *ValidateQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"steps": {
				Name: "steps",
				Schema: unit.Schema{
					Type:        "array",
					Description: "Pipeline steps to validate",
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
		},
		Required: []string{"steps"},
	}
}

func (q *ValidateQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"valid": {
				Name:   "valid",
				Schema: unit.Schema{Type: "boolean"},
			},
			"issues": {
				Name: "issues",
				Schema: unit.Schema{
					Type:  "array",
					Items: &unit.Schema{Type: "string"},
				},
			},
		},
	}
}

func (q *ValidateQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input: map[string]any{
				"steps": []map[string]any{
					{"id": "step1", "name": "Step 1", "type": "inference.chat"},
				},
			},
			Output:      map[string]any{"valid": true, "issues": []string{}},
			Description: "Validate valid steps",
		},
		{
			Input: map[string]any{
				"steps": []map[string]any{
					{"id": "", "name": "Step 1", "type": "inference.chat"},
				},
			},
			Output:      map[string]any{"valid": false, "issues": []string{"step has empty ID"}},
			Description: "Validate invalid steps",
		},
	}
}

func (q *ValidateQuery) Execute(ctx context.Context, input any) (any, error) {
	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type: %w", ErrInvalidInput)
	}

	stepsRaw, ok := inputMap["steps"].([]any)
	if !ok {
		return nil, fmt.Errorf("steps is required: %w", ErrInvalidInput)
	}

	steps := make([]PipelineStep, len(stepsRaw))
	for i, s := range stepsRaw {
		stepMap, ok := s.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid step at index %d: %w", i, ErrInvalidInput)
		}

		step := PipelineStep{
			ID:   fmt.Sprintf("%v", stepMap["id"]),
			Name: fmt.Sprintf("%v", stepMap["name"]),
			Type: fmt.Sprintf("%v", stepMap["type"]),
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

	valid, issues := ValidateSteps(steps)

	return map[string]any{
		"valid":  valid,
		"issues": issues,
	}, nil
}
