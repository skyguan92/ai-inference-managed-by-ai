package pipeline

import (
	"context"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestGetQuery_Name(t *testing.T) {
	q := NewGetQuery(nil)
	if q.Name() != "pipeline.get" {
		t.Errorf("expected name 'pipeline.get', got '%s'", q.Name())
	}
}

func TestGetQuery_Domain(t *testing.T) {
	q := NewGetQuery(nil)
	if q.Domain() != "pipeline" {
		t.Errorf("expected domain 'pipeline', got '%s'", q.Domain())
	}
}

func TestGetQuery_Schemas(t *testing.T) {
	q := NewGetQuery(nil)

	inputSchema := q.InputSchema()
	if inputSchema.Type != "object" {
		t.Errorf("expected input schema type 'object', got '%s'", inputSchema.Type)
	}
	if len(inputSchema.Required) != 1 {
		t.Errorf("expected 1 required field, got %d", len(inputSchema.Required))
	}

	outputSchema := q.OutputSchema()
	if outputSchema.Type != "object" {
		t.Errorf("expected output schema type 'object', got '%s'", outputSchema.Type)
	}
}

func TestGetQuery_Execute(t *testing.T) {
	tests := []struct {
		name    string
		store   PipelineStore
		input   any
		wantErr bool
	}{
		{
			name:    "successful get",
			store:   createStoreWithPipeline("pipe-123", PipelineStatusIdle),
			input:   map[string]any{"pipeline_id": "pipe-123"},
			wantErr: false,
		},
		{
			name:    "nil store",
			store:   nil,
			input:   map[string]any{"pipeline_id": "pipe-123"},
			wantErr: true,
		},
		{
			name:    "missing pipeline_id",
			store:   NewMemoryStore(),
			input:   map[string]any{},
			wantErr: true,
		},
		{
			name:    "pipeline not found",
			store:   NewMemoryStore(),
			input:   map[string]any{"pipeline_id": "nonexistent"},
			wantErr: true,
		},
		{
			name:    "invalid input type",
			store:   NewMemoryStore(),
			input:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewGetQuery(tt.store)
			result, err := q.Execute(context.Background(), tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			resultMap, ok := result.(map[string]any)
			if !ok {
				t.Error("expected result to be map[string]any")
				return
			}

			if _, exists := resultMap["id"]; !exists {
				t.Error("expected field 'id' not found")
			}
			if _, exists := resultMap["steps"]; !exists {
				t.Error("expected field 'steps' not found")
			}
		})
	}
}

func TestListQuery_Name(t *testing.T) {
	q := NewListQuery(nil)
	if q.Name() != "pipeline.list" {
		t.Errorf("expected name 'pipeline.list', got '%s'", q.Name())
	}
}

func TestListQuery_Execute(t *testing.T) {
	tests := []struct {
		name     string
		store    PipelineStore
		input    any
		wantErr  bool
		checkLen bool
	}{
		{
			name:     "list all pipelines",
			store:    createStoreWithMultiplePipelines(),
			input:    map[string]any{},
			wantErr:  false,
			checkLen: true,
		},
		{
			name:     "list with status filter",
			store:    createStoreWithMultiplePipelines(),
			input:    map[string]any{"status": "idle"},
			wantErr:  false,
			checkLen: true,
		},
		{
			name:     "list with pagination",
			store:    createStoreWithMultiplePipelines(),
			input:    map[string]any{"limit": 1, "offset": 0},
			wantErr:  false,
			checkLen: true,
		},
		{
			name:    "nil store",
			store:   nil,
			input:   map[string]any{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewListQuery(tt.store)
			result, err := q.Execute(context.Background(), tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			resultMap, ok := result.(map[string]any)
			if !ok {
				t.Error("expected result to be map[string]any")
				return
			}

			if _, exists := resultMap["pipelines"]; !exists {
				t.Error("expected field 'pipelines' not found")
			}
			if _, exists := resultMap["total"]; !exists {
				t.Error("expected field 'total' not found")
			}

			if tt.checkLen {
				pipelines, ok := resultMap["pipelines"].([]map[string]any)
				if !ok {
					t.Error("expected pipelines to be []map[string]any")
				}
				if len(pipelines) == 0 {
					t.Error("expected at least one pipeline")
				}
			}
		})
	}
}

func TestStatusQuery_Name(t *testing.T) {
	q := NewStatusQuery(nil)
	if q.Name() != "pipeline.status" {
		t.Errorf("expected name 'pipeline.status', got '%s'", q.Name())
	}
}

func TestStatusQuery_Execute(t *testing.T) {
	tests := []struct {
		name    string
		store   PipelineStore
		input   any
		wantErr bool
		setup   func(store PipelineStore)
	}{
		{
			name:    "get run status",
			store:   NewMemoryStore(),
			input:   map[string]any{"run_id": "run-123"},
			wantErr: false,
			setup: func(store PipelineStore) {
				store.CreatePipeline(context.Background(), createTestPipeline("pipe-123", "test", PipelineStatusIdle))
				run := createTestRun("run-123", "pipe-123", RunStatusRunning)
				store.CreateRun(context.Background(), run)
			},
		},
		{
			name:    "get completed run status",
			store:   NewMemoryStore(),
			input:   map[string]any{"run_id": "run-123"},
			wantErr: false,
			setup: func(store PipelineStore) {
				store.CreatePipeline(context.Background(), createTestPipeline("pipe-123", "test", PipelineStatusIdle))
				run := createTestRun("run-123", "pipe-123", RunStatusCompleted)
				run.StepResults = map[string]any{"step1": map[string]any{"result": "ok"}}
				store.CreateRun(context.Background(), run)
			},
		},
		{
			name:    "get failed run status",
			store:   NewMemoryStore(),
			input:   map[string]any{"run_id": "run-123"},
			wantErr: false,
			setup: func(store PipelineStore) {
				store.CreatePipeline(context.Background(), createTestPipeline("pipe-123", "test", PipelineStatusIdle))
				run := createTestRun("run-123", "pipe-123", RunStatusFailed)
				run.Error = "step1 failed"
				store.CreateRun(context.Background(), run)
			},
		},
		{
			name:    "nil store",
			store:   nil,
			input:   map[string]any{"run_id": "run-123"},
			wantErr: true,
		},
		{
			name:    "missing run_id",
			store:   NewMemoryStore(),
			input:   map[string]any{},
			wantErr: true,
		},
		{
			name:    "run not found",
			store:   NewMemoryStore(),
			input:   map[string]any{"run_id": "nonexistent"},
			wantErr: true,
		},
		{
			name:    "invalid input type",
			store:   NewMemoryStore(),
			input:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(tt.store)
			}

			q := NewStatusQuery(tt.store)
			result, err := q.Execute(context.Background(), tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			resultMap, ok := result.(map[string]any)
			if !ok {
				t.Error("expected result to be map[string]any")
				return
			}

			if _, exists := resultMap["status"]; !exists {
				t.Error("expected field 'status' not found")
			}
		})
	}
}

func TestValidateQuery_Name(t *testing.T) {
	q := NewValidateQuery()
	if q.Name() != "pipeline.validate" {
		t.Errorf("expected name 'pipeline.validate', got '%s'", q.Name())
	}
}

func TestValidateQuery_Execute(t *testing.T) {
	tests := []struct {
		name        string
		input       any
		wantErr     bool
		expectValid bool
	}{
		{
			name: "valid steps",
			input: map[string]any{
				"steps": []any{
					map[string]any{"id": "step1", "name": "Step 1", "type": "inference.chat"},
					map[string]any{"id": "step2", "name": "Step 2", "type": "inference.chat", "depends_on": []any{"step1"}},
				},
			},
			wantErr:     false,
			expectValid: true,
		},
		{
			name: "empty step id",
			input: map[string]any{
				"steps": []any{
					map[string]any{"id": "", "name": "Step 1", "type": "inference.chat"},
				},
			},
			wantErr:     false,
			expectValid: false,
		},
		{
			name: "duplicate step id",
			input: map[string]any{
				"steps": []any{
					map[string]any{"id": "step1", "name": "Step 1", "type": "inference.chat"},
					map[string]any{"id": "step1", "name": "Step 2", "type": "inference.chat"},
				},
			},
			wantErr:     false,
			expectValid: false,
		},
		{
			name: "invalid dependency",
			input: map[string]any{
				"steps": []any{
					map[string]any{"id": "step1", "name": "Step 1", "type": "inference.chat", "depends_on": []any{"nonexistent"}},
				},
			},
			wantErr:     false,
			expectValid: false,
		},
		{
			name: "missing steps",
			input: map[string]any{
				"steps": "invalid",
			},
			wantErr: true,
		},
		{
			name:    "invalid input type",
			input:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewValidateQuery()
			result, err := q.Execute(context.Background(), tt.input)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			resultMap, ok := result.(map[string]any)
			if !ok {
				t.Error("expected result to be map[string]any")
				return
			}

			valid, ok := resultMap["valid"].(bool)
			if !ok {
				t.Error("expected field 'valid' to be bool")
				return
			}

			if valid != tt.expectValid {
				t.Errorf("expected valid=%v, got %v", tt.expectValid, valid)
			}
		})
	}
}

func TestQuery_Description(t *testing.T) {
	if NewGetQuery(nil).Description() == "" {
		t.Error("expected non-empty description for GetQuery")
	}
	if NewListQuery(nil).Description() == "" {
		t.Error("expected non-empty description for ListQuery")
	}
	if NewStatusQuery(nil).Description() == "" {
		t.Error("expected non-empty description for StatusQuery")
	}
	if NewValidateQuery().Description() == "" {
		t.Error("expected non-empty description for ValidateQuery")
	}
}

func TestQuery_Examples(t *testing.T) {
	if len(NewGetQuery(nil).Examples()) == 0 {
		t.Error("expected at least one example for GetQuery")
	}
	if len(NewListQuery(nil).Examples()) == 0 {
		t.Error("expected at least one example for ListQuery")
	}
	if len(NewStatusQuery(nil).Examples()) == 0 {
		t.Error("expected at least one example for StatusQuery")
	}
	if len(NewValidateQuery().Examples()) == 0 {
		t.Error("expected at least one example for ValidateQuery")
	}
}

func TestQueryImplementsInterface(t *testing.T) {
	var _ unit.Query = NewGetQuery(nil)
	var _ unit.Query = NewListQuery(nil)
	var _ unit.Query = NewStatusQuery(nil)
	var _ unit.Query = NewValidateQuery()
}

func createStoreWithMultiplePipelines() PipelineStore {
	store := NewMemoryStore()
	store.CreatePipeline(context.Background(), createTestPipeline("pipe-1", "pipeline-1", PipelineStatusIdle))
	store.CreatePipeline(context.Background(), createTestPipeline("pipe-2", "pipeline-2", PipelineStatusRunning))
	store.CreatePipeline(context.Background(), createTestPipeline("pipe-3", "pipeline-3", PipelineStatusIdle))
	return store
}
