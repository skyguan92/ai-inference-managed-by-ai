package model

import (
	"context"
	"errors"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestCreateCommand_Name(t *testing.T) {
	cmd := NewCreateCommand(nil)
	if cmd.Name() != "model.create" {
		t.Errorf("expected name 'model.create', got '%s'", cmd.Name())
	}
}

func TestCreateCommand_Domain(t *testing.T) {
	cmd := NewCreateCommand(nil)
	if cmd.Domain() != "model" {
		t.Errorf("expected domain 'model', got '%s'", cmd.Domain())
	}
}

func TestCreateCommand_Schemas(t *testing.T) {
	cmd := NewCreateCommand(nil)

	inputSchema := cmd.InputSchema()
	if inputSchema.Type != "object" {
		t.Errorf("expected input schema type 'object', got '%s'", inputSchema.Type)
	}
	if len(inputSchema.Required) != 1 {
		t.Errorf("expected 1 required field, got %d", len(inputSchema.Required))
	}

	outputSchema := cmd.OutputSchema()
	if outputSchema.Type != "object" {
		t.Errorf("expected output schema type 'object', got '%s'", outputSchema.Type)
	}
}

func TestCreateCommand_Execute(t *testing.T) {
	tests := []struct {
		name       string
		store      ModelStore
		input      any
		wantErr    bool
		checkField string
	}{
		{
			name:  "successful create",
			store: NewMemoryStore(),
			input: map[string]any{
				"name":   "llama3",
				"type":   "llm",
				"format": "gguf",
			},
			wantErr:    false,
			checkField: "model_id",
		},
		{
			name:    "nil store",
			store:   nil,
			input:   map[string]any{"name": "llama3"},
			wantErr: true,
		},
		{
			name:    "missing name",
			store:   NewMemoryStore(),
			input:   map[string]any{},
			wantErr: true,
		},
		{
			name:    "invalid input type",
			store:   NewMemoryStore(),
			input:   "invalid",
			wantErr: true,
		},
		{
			name:  "with optional fields",
			store: NewMemoryStore(),
			input: map[string]any{
				"name":   "test-model",
				"type":   "vlm",
				"format": "safetensors",
				"source": "huggingface",
				"path":   "/models/test",
			},
			wantErr:    false,
			checkField: "model_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewCreateCommand(tt.store)
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
	if cmd.Name() != "model.delete" {
		t.Errorf("expected name 'model.delete', got '%s'", cmd.Name())
	}
}

func TestDeleteCommand_Execute(t *testing.T) {
	tests := []struct {
		name    string
		store   ModelStore
		input   any
		wantErr bool
	}{
		{
			name: "successful delete",
			store: func() ModelStore {
				s := NewMemoryStore()
				s.Create(context.Background(), createTestModel("model-123", "llama3"))
				return s
			}(),
			input:   map[string]any{"model_id": "model-123"},
			wantErr: false,
		},
		{
			name:    "nil store",
			store:   nil,
			input:   map[string]any{"model_id": "model-123"},
			wantErr: true,
		},
		{
			name:    "missing model_id",
			store:   NewMemoryStore(),
			input:   map[string]any{},
			wantErr: true,
		},
		{
			name:    "model not found",
			store:   NewMemoryStore(),
			input:   map[string]any{"model_id": "nonexistent"},
			wantErr: true,
		},
		{
			name: "delete with force",
			store: func() ModelStore {
				s := NewMemoryStore()
				s.Create(context.Background(), createTestModel("model-123", "llama3"))
				return s
			}(),
			input:   map[string]any{"model_id": "model-123", "force": true},
			wantErr: false,
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

func TestPullCommand_Name(t *testing.T) {
	cmd := NewPullCommand(nil, nil)
	if cmd.Name() != "model.pull" {
		t.Errorf("expected name 'model.pull', got '%s'", cmd.Name())
	}
}

func TestPullCommand_Execute(t *testing.T) {
	tests := []struct {
		name     string
		store    ModelStore
		provider ModelProvider
		input    any
		wantErr  bool
	}{
		{
			name:     "successful pull from ollama",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input: map[string]any{
				"source": "ollama",
				"repo":   "llama3",
			},
			wantErr: false,
		},
		{
			name:     "successful pull with tag",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input: map[string]any{
				"source": "huggingface",
				"repo":   "meta-llama/Llama-3-8B",
				"tag":    "main",
			},
			wantErr: false,
		},
		{
			name:     "nil store",
			store:    nil,
			provider: &MockProvider{},
			input:    map[string]any{"source": "ollama", "repo": "llama3"},
			wantErr:  true,
		},
		{
			name:     "nil provider",
			store:    NewMemoryStore(),
			provider: nil,
			input:    map[string]any{"source": "ollama", "repo": "llama3"},
			wantErr:  true,
		},
		{
			name:     "missing source",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    map[string]any{"repo": "llama3"},
			wantErr:  true,
		},
		{
			name:     "missing repo",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    map[string]any{"source": "ollama"},
			wantErr:  true,
		},
		{
			name:     "provider error",
			store:    NewMemoryStore(),
			provider: &MockProvider{pullErr: errors.New("pull failed")},
			input:    map[string]any{"source": "ollama", "repo": "llama3"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewPullCommand(tt.store, tt.provider)
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

			if _, ok := resultMap["model_id"]; !ok {
				t.Error("expected 'model_id' field")
			}
			if _, ok := resultMap["status"]; !ok {
				t.Error("expected 'status' field")
			}
		})
	}
}

func TestImportCommand_Name(t *testing.T) {
	cmd := NewImportCommand(nil, nil)
	if cmd.Name() != "model.import" {
		t.Errorf("expected name 'model.import', got '%s'", cmd.Name())
	}
}

func TestImportCommand_Execute(t *testing.T) {
	tests := []struct {
		name     string
		store    ModelStore
		provider ModelProvider
		input    any
		wantErr  bool
	}{
		{
			name:     "successful import with auto detect",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input: map[string]any{
				"path":        "/models/llama3",
				"auto_detect": true,
			},
			wantErr: false,
		},
		{
			name:     "successful import with explicit settings",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input: map[string]any{
				"path": "/models/custom",
				"name": "my-model",
				"type": "vlm",
			},
			wantErr: false,
		},
		{
			name:     "nil store",
			store:    nil,
			provider: &MockProvider{},
			input:    map[string]any{"path": "/models/test"},
			wantErr:  true,
		},
		{
			name:     "missing path",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    map[string]any{},
			wantErr:  true,
		},
		{
			name:     "provider error",
			store:    NewMemoryStore(),
			provider: &MockProvider{importErr: errors.New("import failed")},
			input:    map[string]any{"path": "/models/test"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewImportCommand(tt.store, tt.provider)
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

			if _, ok := resultMap["model_id"]; !ok {
				t.Error("expected 'model_id' field")
			}
		})
	}
}

func TestVerifyCommand_Name(t *testing.T) {
	cmd := NewVerifyCommand(nil, nil)
	if cmd.Name() != "model.verify" {
		t.Errorf("expected name 'model.verify', got '%s'", cmd.Name())
	}
}

func TestVerifyCommand_Execute(t *testing.T) {
	tests := []struct {
		name      string
		store     ModelStore
		provider  ModelProvider
		input     any
		wantErr   bool
		wantValid bool
	}{
		{
			name: "successful verification",
			store: func() ModelStore {
				s := NewMemoryStore()
				s.Create(context.Background(), createTestModel("model-123", "llama3"))
				return s
			}(),
			provider:  &MockProvider{},
			input:     map[string]any{"model_id": "model-123"},
			wantErr:   false,
			wantValid: true,
		},
		{
			name: "verification with checksum",
			store: func() ModelStore {
				s := NewMemoryStore()
				s.Create(context.Background(), createTestModel("model-123", "llama3"))
				return s
			}(),
			provider:  &MockProvider{},
			input:     map[string]any{"model_id": "model-123", "checksum": "sha256:abc123"},
			wantErr:   false,
			wantValid: true,
		},
		{
			name: "verification failed",
			store: func() ModelStore {
				s := NewMemoryStore()
				s.Create(context.Background(), createTestModel("model-123", "llama3"))
				return s
			}(),
			provider:  &MockProvider{verifyRes: &VerificationResult{Valid: false, Issues: []string{"checksum mismatch"}}},
			input:     map[string]any{"model_id": "model-123"},
			wantErr:   false,
			wantValid: false,
		},
		{
			name:     "nil store",
			store:    nil,
			provider: &MockProvider{},
			input:    map[string]any{"model_id": "model-123"},
			wantErr:  true,
		},
		{
			name:     "missing model_id",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    map[string]any{},
			wantErr:  true,
		},
		{
			name:     "model not found",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    map[string]any{"model_id": "nonexistent"},
			wantErr:  true,
		},
		{
			name: "provider error",
			store: func() ModelStore {
				s := NewMemoryStore()
				s.Create(context.Background(), createTestModel("model-123", "llama3"))
				return s
			}(),
			provider: &MockProvider{verifyErr: errors.New("verify failed")},
			input:    map[string]any{"model_id": "model-123"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewVerifyCommand(tt.store, tt.provider)
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

			if valid, ok := resultMap["valid"].(bool); ok {
				if valid != tt.wantValid {
					t.Errorf("expected valid=%v, got %v", tt.wantValid, valid)
				}
			} else {
				t.Error("expected 'valid' field to be bool")
			}
		})
	}
}

func TestCommand_Description(t *testing.T) {
	if NewCreateCommand(nil).Description() == "" {
		t.Error("expected non-empty description for CreateCommand")
	}
	if NewDeleteCommand(nil).Description() == "" {
		t.Error("expected non-empty description for DeleteCommand")
	}
	if NewPullCommand(nil, nil).Description() == "" {
		t.Error("expected non-empty description for PullCommand")
	}
	if NewImportCommand(nil, nil).Description() == "" {
		t.Error("expected non-empty description for ImportCommand")
	}
	if NewVerifyCommand(nil, nil).Description() == "" {
		t.Error("expected non-empty description for VerifyCommand")
	}
}

func TestCommand_Examples(t *testing.T) {
	if len(NewCreateCommand(nil).Examples()) == 0 {
		t.Error("expected at least one example for CreateCommand")
	}
	if len(NewDeleteCommand(nil).Examples()) == 0 {
		t.Error("expected at least one example for DeleteCommand")
	}
	if len(NewPullCommand(nil, nil).Examples()) == 0 {
		t.Error("expected at least one example for PullCommand")
	}
	if len(NewImportCommand(nil, nil).Examples()) == 0 {
		t.Error("expected at least one example for ImportCommand")
	}
	if len(NewVerifyCommand(nil, nil).Examples()) == 0 {
		t.Error("expected at least one example for VerifyCommand")
	}
}

func TestCommandImplementsInterface(t *testing.T) {
	var _ unit.Command = NewCreateCommand(nil)
	var _ unit.Command = NewDeleteCommand(nil)
	var _ unit.Command = NewPullCommand(nil, nil)
	var _ unit.Command = NewImportCommand(nil, nil)
	var _ unit.Command = NewVerifyCommand(nil, nil)
}
