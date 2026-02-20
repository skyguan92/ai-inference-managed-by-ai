package model

import (
	"context"
	"errors"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestGetQuery_Name(t *testing.T) {
	q := NewGetQuery(nil)
	if q.Name() != "model.get" {
		t.Errorf("expected name 'model.get', got '%s'", q.Name())
	}
}

func TestGetQuery_Domain(t *testing.T) {
	q := NewGetQuery(nil)
	if q.Domain() != "model" {
		t.Errorf("expected domain 'model', got '%s'", q.Domain())
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
		name       string
		store      ModelStore
		input      any
		wantErr    bool
		checkField string
		checkValue any
	}{
		{
			name: "successful get",
			store: func() ModelStore {
				s := NewMemoryStore()
				_ = s.Create(context.Background(), createTestModel("model-123", "llama3"))
				return s
			}(),
			input:      map[string]any{"model_id": "model-123"},
			wantErr:    false,
			checkField: "name",
			checkValue: "llama3",
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

			if tt.checkValue != nil {
				if val, exists := resultMap[tt.checkField]; exists {
					if val != tt.checkValue {
						t.Errorf("expected %s=%v, got %v", tt.checkField, tt.checkValue, val)
					}
				} else {
					t.Errorf("expected field '%s' not found", tt.checkField)
				}
			}
		})
	}
}

func TestListQuery_Name(t *testing.T) {
	q := NewListQuery(nil)
	if q.Name() != "model.list" {
		t.Errorf("expected name 'model.list', got '%s'", q.Name())
	}
}

func TestListQuery_Execute(t *testing.T) {
	tests := []struct {
		name      string
		store     ModelStore
		input     any
		wantErr   bool
		wantCount int
		wantTotal int
	}{
		{
			name: "list all models",
			store: func() ModelStore {
				s := NewMemoryStore()
				_ = s.Create(context.Background(), createTestModel("model-1", "llama3"))
				_ = s.Create(context.Background(), createTestModel("model-2", "mistral"))
				return s
			}(),
			input:     map[string]any{},
			wantErr:   false,
			wantCount: 2,
			wantTotal: 2,
		},
		{
			name: "list with type filter",
			store: func() ModelStore {
				s := NewMemoryStore()
				_ = s.Create(context.Background(), createTestModel("model-1", "llama3"))
				m := createTestModel("model-2", "whisper")
				m.Type = ModelTypeASR
				_ = s.Create(context.Background(), m)
				return s
			}(),
			input:     map[string]any{"type": "llm"},
			wantErr:   false,
			wantCount: 1,
			wantTotal: 1,
		},
		{
			name: "list with pagination",
			store: func() ModelStore {
				s := NewMemoryStore()
				_ = s.Create(context.Background(), createTestModel("model-1", "llama3"))
				_ = s.Create(context.Background(), createTestModel("model-2", "mistral"))
				_ = s.Create(context.Background(), createTestModel("model-3", "codellama"))
				return s
			}(),
			input:     map[string]any{"limit": 2, "offset": 1},
			wantErr:   false,
			wantCount: 2,
			wantTotal: 3,
		},
		{
			name:    "nil store",
			store:   nil,
			input:   map[string]any{},
			wantErr: true,
		},
		{
			name:      "empty store",
			store:     NewMemoryStore(),
			input:     map[string]any{},
			wantErr:   false,
			wantCount: 0,
			wantTotal: 0,
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

			items, ok := resultMap["items"].([]map[string]any)
			if !ok {
				t.Error("expected 'items' to be []map[string]any")
				return
			}

			if len(items) != tt.wantCount {
				t.Errorf("expected %d items, got %d", tt.wantCount, len(items))
			}

			if total, ok := resultMap["total"].(int); ok {
				if total != tt.wantTotal {
					t.Errorf("expected total=%d, got %d", tt.wantTotal, total)
				}
			}
		})
	}
}

func TestSearchQuery_Name(t *testing.T) {
	q := NewSearchQuery(nil)
	if q.Name() != "model.search" {
		t.Errorf("expected name 'model.search', got '%s'", q.Name())
	}
}

func TestSearchQuery_Execute(t *testing.T) {
	tests := []struct {
		name      string
		provider  ModelProvider
		input     any
		wantErr   bool
		wantCount int
	}{
		{
			name: "successful search",
			provider: &MockProvider{
				searchRes: []ModelSearchResult{
					{ID: "llama3", Name: "Llama 3", Type: ModelTypeLLM, Source: "ollama"},
					{ID: "llama2", Name: "Llama 2", Type: ModelTypeLLM, Source: "ollama"},
				},
			},
			input:     map[string]any{"query": "llama", "source": "ollama"},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:      "empty results",
			provider:  &MockProvider{searchRes: []ModelSearchResult{}},
			input:     map[string]any{"query": "nonexistent"},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name:     "nil provider",
			provider: nil,
			input:    map[string]any{"query": "llama"},
			wantErr:  true,
		},
		{
			name:     "missing query",
			provider: &MockProvider{},
			input:    map[string]any{},
			wantErr:  true,
		},
		{
			name:     "invalid input type",
			provider: &MockProvider{},
			input:    "invalid",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewSearchQuery(tt.provider)
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

			results, ok := resultMap["results"].([]map[string]any)
			if !ok {
				t.Error("expected 'results' to be []map[string]any")
				return
			}

			if len(results) != tt.wantCount {
				t.Errorf("expected %d results, got %d", tt.wantCount, len(results))
			}
		})
	}
}

func TestEstimateResourcesQuery_Name(t *testing.T) {
	q := NewEstimateResourcesQuery(nil, nil)
	if q.Name() != "model.estimate_resources" {
		t.Errorf("expected name 'model.estimate_resources', got '%s'", q.Name())
	}
}

func TestEstimateResourcesQuery_Execute(t *testing.T) {
	tests := []struct {
		name          string
		store         ModelStore
		provider      ModelProvider
		input         any
		wantErr       bool
		wantMemoryMin int64
		wantGPUType   string
	}{
		{
			name: "estimate from model requirements",
			store: func() ModelStore {
				s := NewMemoryStore()
				_ = s.Create(context.Background(), createTestModel("model-123", "llama3"))
				return s
			}(),
			provider:      &MockProvider{},
			input:         map[string]any{"model_id": "model-123"},
			wantErr:       false,
			wantMemoryMin: 8000000000,
			wantGPUType:   "NVIDIA RTX 4090",
		},
		{
			name: "estimate from provider",
			store: func() ModelStore {
				s := NewMemoryStore()
				m := createTestModel("model-123", "llama3")
				m.Requirements = nil
				_ = s.Create(context.Background(), m)
				return s
			}(),
			provider: &MockProvider{
				estimate: &ModelRequirements{
					MemoryMin:         12000000000,
					MemoryRecommended: 24000000000,
					GPUType:           "NVIDIA A100",
				},
			},
			input:         map[string]any{"model_id": "model-123"},
			wantErr:       false,
			wantMemoryMin: 12000000000,
			wantGPUType:   "NVIDIA A100",
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
				m := createTestModel("model-123", "llama3")
				m.Requirements = nil
				_ = s.Create(context.Background(), m)
				return s
			}(),
			provider: &MockProvider{estimateErr: errors.New("estimate failed")},
			input:    map[string]any{"model_id": "model-123"},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewEstimateResourcesQuery(tt.store, tt.provider)
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

			if memMin, ok := resultMap["memory_min"].(int64); ok {
				if memMin != tt.wantMemoryMin {
					t.Errorf("expected memory_min=%d, got %d", tt.wantMemoryMin, memMin)
				}
			}

			if gpuType, ok := resultMap["gpu_type"].(string); ok {
				if gpuType != tt.wantGPUType {
					t.Errorf("expected gpu_type=%s, got %s", tt.wantGPUType, gpuType)
				}
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
	if NewSearchQuery(nil).Description() == "" {
		t.Error("expected non-empty description for SearchQuery")
	}
	if NewEstimateResourcesQuery(nil, nil).Description() == "" {
		t.Error("expected non-empty description for EstimateResourcesQuery")
	}
}

func TestQuery_Examples(t *testing.T) {
	if len(NewGetQuery(nil).Examples()) == 0 {
		t.Error("expected at least one example for GetQuery")
	}
	if len(NewListQuery(nil).Examples()) == 0 {
		t.Error("expected at least one example for ListQuery")
	}
	if len(NewSearchQuery(nil).Examples()) == 0 {
		t.Error("expected at least one example for SearchQuery")
	}
	if len(NewEstimateResourcesQuery(nil, nil).Examples()) == 0 {
		t.Error("expected at least one example for EstimateResourcesQuery")
	}
}

func TestQueryImplementsInterface(t *testing.T) {
	var _ unit.Query = NewGetQuery(nil)
	var _ unit.Query = NewListQuery(nil)
	var _ unit.Query = NewSearchQuery(nil)
	var _ unit.Query = NewEstimateResourcesQuery(nil, nil)
}
