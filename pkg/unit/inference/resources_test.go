package inference

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestModelsResource_URI(t *testing.T) {
	resource := NewModelsResource(nil)
	if resource.URI() != "asms://inference/models" {
		t.Errorf("expected URI 'asms://inference/models', got '%s'", resource.URI())
	}
}

func TestModelsResource_Domain(t *testing.T) {
	resource := NewModelsResource(nil)
	if resource.Domain() != "inference" {
		t.Errorf("expected domain 'inference', got '%s'", resource.Domain())
	}
}

func TestModelsResource_Schema(t *testing.T) {
	resource := NewModelsResource(nil)
	schema := resource.Schema()

	if schema.Type != "object" {
		t.Errorf("expected schema type 'object', got '%s'", schema.Type)
	}

	if _, ok := schema.Properties["models"]; !ok {
		t.Error("expected 'models' property in schema")
	}
}

func TestModelsResource_Get(t *testing.T) {
	tests := []struct {
		name     string
		provider InferenceProvider
		wantErr  bool
	}{
		{
			name:     "successful get",
			provider: NewMockProvider(),
			wantErr:  false,
		},
		{
			name:     "nil provider",
			provider: nil,
			wantErr:  true,
		},
		{
			name:     "provider error",
			provider: &MockProvider{listModelsErr: errors.New("list failed")},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resource := NewModelsResource(tt.provider)
			result, err := resource.Get(context.Background())

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
				t.Error("expected 'models' field")
				return
			}

			if len(models) == 0 {
				t.Error("expected at least one model")
			}
		})
	}
}

func TestModelsResource_Watch(t *testing.T) {
	resource := NewModelsResource(NewMockProvider())
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ch, err := resource.Watch(ctx)
	if err != nil {
		t.Errorf("unexpected error starting watch: %v", err)
		return
	}

	if ch == nil {
		t.Error("expected non-nil channel")
		return
	}

	_, ok := <-ch
	if !ok {
		t.Log("channel closed as expected after context cancellation")
	}
}

func TestResourceImplementsInterface(t *testing.T) {
	var _ unit.Resource = NewModelsResource(nil)
}
