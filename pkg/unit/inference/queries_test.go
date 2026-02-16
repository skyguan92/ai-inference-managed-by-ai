package inference

import (
	"context"
	"errors"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestModelsQuery_Name(t *testing.T) {
	query := NewModelsQuery(nil)
	if query.Name() != "inference.models" {
		t.Errorf("expected name 'inference.models', got '%s'", query.Name())
	}
}

func TestModelsQuery_Domain(t *testing.T) {
	query := NewModelsQuery(nil)
	if query.Domain() != "inference" {
		t.Errorf("expected domain 'inference', got '%s'", query.Domain())
	}
}

func TestModelsQuery_Schemas(t *testing.T) {
	query := NewModelsQuery(nil)

	inputSchema := query.InputSchema()
	if inputSchema.Type != "object" {
		t.Errorf("expected input schema type 'object', got '%s'", inputSchema.Type)
	}

	outputSchema := query.OutputSchema()
	if outputSchema.Type != "object" {
		t.Errorf("expected output schema type 'object', got '%s'", outputSchema.Type)
	}
}

func TestModelsQuery_Execute(t *testing.T) {
	tests := []struct {
		name     string
		provider InferenceProvider
		input    any
		wantErr  bool
	}{
		{
			name:     "list all models",
			provider: NewMockProvider(),
			input:    map[string]any{},
			wantErr:  false,
		},
		{
			name:     "list by type",
			provider: NewMockProvider(),
			input:    map[string]any{"type": "llm"},
			wantErr:  false,
		},
		{
			name:     "nil provider",
			provider: nil,
			input:    map[string]any{},
			wantErr:  true,
		},
		{
			name:     "provider error",
			provider: &MockProvider{listModelsErr: errors.New("list failed")},
			input:    map[string]any{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := NewModelsQuery(tt.provider)
			result, err := query.Execute(context.Background(), tt.input)

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

			models, ok := resultMap["models"].([]map[string]any)
			if !ok {
				t.Error("expected 'models' field to be []map[string]any")
				return
			}

			if len(models) == 0 {
				t.Error("expected at least one model")
			}
		})
	}
}

func TestVoicesQuery_Name(t *testing.T) {
	query := NewVoicesQuery(nil)
	if query.Name() != "inference.voices" {
		t.Errorf("expected name 'inference.voices', got '%s'", query.Name())
	}
}

func TestVoicesQuery_Domain(t *testing.T) {
	query := NewVoicesQuery(nil)
	if query.Domain() != "inference" {
		t.Errorf("expected domain 'inference', got '%s'", query.Domain())
	}
}

func TestVoicesQuery_Execute(t *testing.T) {
	tests := []struct {
		name     string
		provider InferenceProvider
		input    any
		wantErr  bool
	}{
		{
			name:     "list all voices",
			provider: NewMockProvider(),
			input:    map[string]any{},
			wantErr:  false,
		},
		{
			name:     "list voices for model",
			provider: NewMockProvider(),
			input:    map[string]any{"model": "tts-1"},
			wantErr:  false,
		},
		{
			name:     "nil provider",
			provider: nil,
			input:    map[string]any{},
			wantErr:  true,
		},
		{
			name:     "provider error",
			provider: &MockProvider{listVoicesErr: errors.New("list failed")},
			input:    map[string]any{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := NewVoicesQuery(tt.provider)
			result, err := query.Execute(context.Background(), tt.input)

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

			voices, ok := resultMap["voices"].([]map[string]any)
			if !ok {
				t.Error("expected 'voices' field to be []map[string]any")
				return
			}

			if len(voices) == 0 {
				t.Error("expected at least one voice")
			}
		})
	}
}

func TestQuery_Description(t *testing.T) {
	if NewModelsQuery(nil).Description() == "" {
		t.Error("expected non-empty description for ModelsQuery")
	}
	if NewVoicesQuery(nil).Description() == "" {
		t.Error("expected non-empty description for VoicesQuery")
	}
}

func TestQuery_Examples(t *testing.T) {
	if len(NewModelsQuery(nil).Examples()) == 0 {
		t.Error("expected at least one example for ModelsQuery")
	}
	if len(NewVoicesQuery(nil).Examples()) == 0 {
		t.Error("expected at least one example for VoicesQuery")
	}
}

func TestQueryImplementsInterface(t *testing.T) {
	var _ unit.Query = NewModelsQuery(nil)
	var _ unit.Query = NewVoicesQuery(nil)
}
