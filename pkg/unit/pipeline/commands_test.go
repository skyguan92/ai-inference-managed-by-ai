package pipeline

import (
	"context"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestCreateCommand_Name(t *testing.T) {
	cmd := NewCreateCommand(nil, nil)
	if cmd.Name() != "pipeline.create" {
		t.Errorf("expected name 'pipeline.create', got '%s'", cmd.Name())
	}
}

func TestCreateCommand_Domain(t *testing.T) {
	cmd := NewCreateCommand(nil, nil)
	if cmd.Domain() != "pipeline" {
		t.Errorf("expected domain 'pipeline', got '%s'", cmd.Domain())
	}
}

func TestCreateCommand_Schemas(t *testing.T) {
	cmd := NewCreateCommand(nil, nil)

	inputSchema := cmd.InputSchema()
	if inputSchema.Type != "object" {
		t.Errorf("expected input schema type 'object', got '%s'", inputSchema.Type)
	}
	if len(inputSchema.Required) != 2 {
		t.Errorf("expected 2 required fields, got %d", len(inputSchema.Required))
	}

	outputSchema := cmd.OutputSchema()
	if outputSchema.Type != "object" {
		t.Errorf("expected output schema type 'object', got '%s'", outputSchema.Type)
	}
}

func TestCreateCommand_Execute(t *testing.T) {
	tests := []struct {
		name       string
		store      PipelineStore
		input      any
		wantErr    bool
		checkField string
	}{
		{
			name:  "successful create",
			store: NewMemoryStore(),
			input: map[string]any{
				"name": "test-pipeline",
				"steps": []any{
					map[string]any{"id": "step1", "name": "Step 1", "type": "inference.chat", "input": map[string]any{"model": "test"}},
				},
			},
			wantErr:    false,
			checkField: "pipeline_id",
		},
		{
			name:  "create with config",
			store: NewMemoryStore(),
			input: map[string]any{
				"name": "test-pipeline",
				"steps": []any{
					map[string]any{"id": "step1", "name": "Step 1", "type": "inference.chat"},
				},
				"config": map[string]any{"timeout": 30},
			},
			wantErr:    false,
			checkField: "pipeline_id",
		},
		{
			name:    "nil store",
			store:   nil,
			input:   map[string]any{"name": "test", "steps": []any{map[string]any{"id": "s1", "type": "test"}}},
			wantErr: true,
		},
		{
			name:    "missing name",
			store:   NewMemoryStore(),
			input:   map[string]any{"steps": []map[string]any{{"id": "s1", "type": "test"}}},
			wantErr: true,
		},
		{
			name:    "missing steps",
			store:   NewMemoryStore(),
			input:   map[string]any{"name": "test"},
			wantErr: true,
		},
		{
			name:    "empty steps",
			store:   NewMemoryStore(),
			input:   map[string]any{"name": "test", "steps": []any{}},
			wantErr: true,
		},
		{
			name:    "invalid input type",
			store:   NewMemoryStore(),
			input:   "invalid",
			wantErr: true,
		},
		{
			name:  "invalid step - empty id",
			store: NewMemoryStore(),
			input: map[string]any{
				"name": "test-pipeline",
				"steps": []any{
					map[string]any{"id": "", "name": "Step 1", "type": "inference.chat"},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewCreateCommand(tt.store, nil)
			result, err := cmd.Execute(context.Background(), tt.input)

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

			if tt.checkField != "" {
				if _, exists := resultMap[tt.checkField]; !exists {
					t.Errorf("expected field '%s' not found", tt.checkField)
				}
			}
		})
	}
}

func TestDeleteCommand_Name(t *testing.T) {
	cmd := NewDeleteCommand(nil)
	if cmd.Name() != "pipeline.delete" {
		t.Errorf("expected name 'pipeline.delete', got '%s'", cmd.Name())
	}
}

func TestDeleteCommand_Execute(t *testing.T) {
	tests := []struct {
		name    string
		store   PipelineStore
		input   any
		wantErr bool
	}{
		{
			name:    "successful delete",
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
		{
			name:    "cannot delete running pipeline",
			store:   createStoreWithPipeline("pipe-123", PipelineStatusRunning),
			input:   map[string]any{"pipeline_id": "pipe-123"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewDeleteCommand(tt.store)
			result, err := cmd.Execute(context.Background(), tt.input)

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

			if success, ok := resultMap["success"].(bool); !ok || !success {
				t.Error("expected success=true")
			}
		})
	}
}

func TestRunCommand_Name(t *testing.T) {
	cmd := NewRunCommand(nil, nil)
	if cmd.Name() != "pipeline.run" {
		t.Errorf("expected name 'pipeline.run', got '%s'", cmd.Name())
	}
}

func TestRunCommand_Execute(t *testing.T) {
	tests := []struct {
		name    string
		store   PipelineStore
		input   any
		wantErr bool
	}{
		{
			name:    "successful run",
			store:   createStoreWithPipeline("pipe-123", PipelineStatusIdle),
			input:   map[string]any{"pipeline_id": "pipe-123"},
			wantErr: false,
		},
		{
			name:    "run with input",
			store:   createStoreWithPipeline("pipe-123", PipelineStatusIdle),
			input:   map[string]any{"pipeline_id": "pipe-123", "input": map[string]any{"query": "hello"}},
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
			executor := NewExecutor(tt.store, MockStepExecutor)
			cmd := NewRunCommand(tt.store, executor)
			result, err := cmd.Execute(context.Background(), tt.input)

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

			if _, exists := resultMap["run_id"]; !exists {
				t.Error("expected field 'run_id' not found")
			}
			if _, exists := resultMap["status"]; !exists {
				t.Error("expected field 'status' not found")
			}
		})
	}
}

func TestCancelCommand_Name(t *testing.T) {
	cmd := NewCancelCommand(nil, nil)
	if cmd.Name() != "pipeline.cancel" {
		t.Errorf("expected name 'pipeline.cancel', got '%s'", cmd.Name())
	}
}

func TestCancelCommand_Execute(t *testing.T) {
	tests := []struct {
		name     string
		store    PipelineStore
		executor *Executor
		input    any
		wantErr  bool
		setup    func(store PipelineStore) string
	}{
		{
			name:     "cancel running run",
			store:    NewMemoryStore(),
			executor: NewExecutor(nil, nil),
			input:    map[string]any{"run_id": "run-123"},
			wantErr:  false,
			setup: func(store PipelineStore) string {
				store.CreatePipeline(context.Background(), createTestPipeline("pipe-123", "test", PipelineStatusIdle))
				run := createTestRun("run-123", "pipe-123", RunStatusRunning)
				store.CreateRun(context.Background(), run)
				return ""
			},
		},
		{
			name:    "nil store",
			store:   nil,
			input:   map[string]any{"run_id": "run-123"},
			wantErr: true,
		},
		{
			name:     "missing run_id",
			store:    NewMemoryStore(),
			executor: NewExecutor(nil, nil),
			input:    map[string]any{},
			wantErr:  true,
		},
		{
			name:     "run not found",
			store:    NewMemoryStore(),
			executor: NewExecutor(nil, nil),
			input:    map[string]any{"run_id": "nonexistent"},
			wantErr:  true,
		},
		{
			name:     "invalid input type",
			store:    NewMemoryStore(),
			executor: NewExecutor(nil, nil),
			input:    "invalid",
			wantErr:  true,
		},
		{
			name:     "cannot cancel completed run",
			store:    NewMemoryStore(),
			executor: NewExecutor(nil, nil),
			input:    map[string]any{"run_id": "run-123"},
			wantErr:  true,
			setup: func(store PipelineStore) string {
				store.CreatePipeline(context.Background(), createTestPipeline("pipe-123", "test", PipelineStatusIdle))
				run := createTestRun("run-123", "pipe-123", RunStatusCompleted)
				store.CreateRun(context.Background(), run)
				return ""
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(tt.store)
			}

			cmd := NewCancelCommand(tt.store, tt.executor)
			result, err := cmd.Execute(context.Background(), tt.input)

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

			if success, ok := resultMap["success"].(bool); !ok || !success {
				t.Error("expected success=true")
			}
		})
	}
}

func TestCommand_Description(t *testing.T) {
	if NewCreateCommand(nil, nil).Description() == "" {
		t.Error("expected non-empty description for CreateCommand")
	}
	if NewDeleteCommand(nil).Description() == "" {
		t.Error("expected non-empty description for DeleteCommand")
	}
	if NewRunCommand(nil, nil).Description() == "" {
		t.Error("expected non-empty description for RunCommand")
	}
	if NewCancelCommand(nil, nil).Description() == "" {
		t.Error("expected non-empty description for CancelCommand")
	}
}

func TestCommand_Examples(t *testing.T) {
	if len(NewCreateCommand(nil, nil).Examples()) == 0 {
		t.Error("expected at least one example for CreateCommand")
	}
	if len(NewDeleteCommand(nil).Examples()) == 0 {
		t.Error("expected at least one example for DeleteCommand")
	}
	if len(NewRunCommand(nil, nil).Examples()) == 0 {
		t.Error("expected at least one example for RunCommand")
	}
	if len(NewCancelCommand(nil, nil).Examples()) == 0 {
		t.Error("expected at least one example for CancelCommand")
	}
}

func TestCommandImplementsInterface(t *testing.T) {
	var _ unit.Command = NewCreateCommand(nil, nil)
	var _ unit.Command = NewDeleteCommand(nil)
	var _ unit.Command = NewRunCommand(nil, nil)
	var _ unit.Command = NewCancelCommand(nil, nil)
}

func createStoreWithPipeline(id string, status PipelineStatus) PipelineStore {
	store := NewMemoryStore()
	pipeline := createTestPipeline(id, "test-pipeline", status)
	store.CreatePipeline(context.Background(), pipeline)
	return store
}
