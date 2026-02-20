package engine

import (
	"context"
	"errors"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestGetQuery_Name(t *testing.T) {
	q := NewGetQuery(nil)
	if q.Name() != "engine.get" {
		t.Errorf("expected name 'engine.get', got '%s'", q.Name())
	}
}

func TestGetQuery_Domain(t *testing.T) {
	q := NewGetQuery(nil)
	if q.Domain() != "engine" {
		t.Errorf("expected domain 'engine', got '%s'", q.Domain())
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
		store      EngineStore
		input      any
		wantErr    bool
		checkField string
		checkValue any
	}{
		{
			name: "successful get",
			store: func() EngineStore {
				s := NewMemoryStore()
				_ = s.Create(context.Background(), createTestEngine("ollama", EngineTypeOllama))
				return s
			}(),
			input:      map[string]any{"name": "ollama"},
			wantErr:    false,
			checkField: "type",
			checkValue: "ollama",
		},
		{
			name:    "nil store",
			store:   nil,
			input:   map[string]any{"name": "ollama"},
			wantErr: true,
		},
		{
			name:    "missing name",
			store:   NewMemoryStore(),
			input:   map[string]any{},
			wantErr: true,
		},
		{
			name:    "engine not found",
			store:   NewMemoryStore(),
			input:   map[string]any{"name": "nonexistent"},
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
	if q.Name() != "engine.list" {
		t.Errorf("expected name 'engine.list', got '%s'", q.Name())
	}
}

func TestListQuery_Execute(t *testing.T) {
	tests := []struct {
		name      string
		store     EngineStore
		input     any
		wantErr   bool
		wantCount int
		wantTotal int
	}{
		{
			name: "list all engines",
			store: func() EngineStore {
				s := NewMemoryStore()
				_ = s.Create(context.Background(), createTestEngine("ollama", EngineTypeOllama))
				_ = s.Create(context.Background(), createTestEngine("vllm", EngineTypeVLLM))
				return s
			}(),
			input:     map[string]any{},
			wantErr:   false,
			wantCount: 2,
			wantTotal: 2,
		},
		{
			name: "list with type filter",
			store: func() EngineStore {
				s := NewMemoryStore()
				s.Create(context.Background(), createTestEngine("ollama", EngineTypeOllama))
				s.Create(context.Background(), createTestEngine("whisper", EngineTypeWhisper))
				return s
			}(),
			input:     map[string]any{"type": "ollama"},
			wantErr:   false,
			wantCount: 1,
			wantTotal: 1,
		},
		{
			name: "list with status filter",
			store: func() EngineStore {
				s := NewMemoryStore()
				e1 := createTestEngine("ollama", EngineTypeOllama)
				e1.Status = EngineStatusRunning
				s.Create(context.Background(), e1)
				e2 := createTestEngine("vllm", EngineTypeVLLM)
				e2.Status = EngineStatusStopped
				s.Create(context.Background(), e2)
				return s
			}(),
			input:     map[string]any{"status": "running"},
			wantErr:   false,
			wantCount: 1,
			wantTotal: 1,
		},
		{
			name: "list with pagination",
			store: func() EngineStore {
				s := NewMemoryStore()
				s.Create(context.Background(), createTestEngine("ollama", EngineTypeOllama))
				s.Create(context.Background(), createTestEngine("vllm", EngineTypeVLLM))
				s.Create(context.Background(), createTestEngine("sglang", EngineTypeSGLang))
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

func TestFeaturesQuery_Name(t *testing.T) {
	q := NewFeaturesQuery(nil, nil)
	if q.Name() != "engine.features" {
		t.Errorf("expected name 'engine.features', got '%s'", q.Name())
	}
}

func TestFeaturesQuery_Execute(t *testing.T) {
	tests := []struct {
		name               string
		store              EngineStore
		provider           EngineProvider
		input              any
		wantErr            bool
		wantMaxConcurrent  int
		wantSupportsStream bool
	}{
		{
			name: "successful get features",
			store: func() EngineStore {
				s := NewMemoryStore()
				s.Create(context.Background(), createTestEngine("ollama", EngineTypeOllama))
				return s
			}(),
			provider:           &MockProvider{},
			input:              map[string]any{"name": "ollama"},
			wantErr:            false,
			wantMaxConcurrent:  10,
			wantSupportsStream: true,
		},
		{
			name: "get custom features",
			store: func() EngineStore {
				s := NewMemoryStore()
				s.Create(context.Background(), createTestEngine("vllm", EngineTypeVLLM))
				return s
			}(),
			provider: &MockProvider{
				features: &EngineFeatures{
					SupportsStreaming:    false,
					SupportsBatch:        true,
					MaxConcurrent:        20,
					MaxContextLength:     32768,
					SupportsGPULayers:    true,
					SupportsQuantization: false,
				},
			},
			input:              map[string]any{"name": "vllm"},
			wantErr:            false,
			wantMaxConcurrent:  20,
			wantSupportsStream: false,
		},
		{
			name:     "nil store",
			store:    nil,
			provider: &MockProvider{},
			input:    map[string]any{"name": "ollama"},
			wantErr:  true,
		},
		{
			name:     "nil provider",
			store:    NewMemoryStore(),
			provider: nil,
			input:    map[string]any{"name": "ollama"},
			wantErr:  true,
		},
		{
			name:     "missing name",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    map[string]any{},
			wantErr:  true,
		},
		{
			name:     "engine not found",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    map[string]any{"name": "nonexistent"},
			wantErr:  true,
		},
		{
			name: "provider error",
			store: func() EngineStore {
				s := NewMemoryStore()
				s.Create(context.Background(), createTestEngine("ollama", EngineTypeOllama))
				return s
			}(),
			provider: &MockProvider{featuresErr: errors.New("features error")},
			input:    map[string]any{"name": "ollama"},
			wantErr:  true,
		},
		{
			name:     "invalid input type",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    "invalid",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewFeaturesQuery(tt.store, tt.provider)
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

			if maxConcurrent, ok := resultMap["max_concurrent"].(int); ok {
				if maxConcurrent != tt.wantMaxConcurrent {
					t.Errorf("expected max_concurrent=%d, got %d", tt.wantMaxConcurrent, maxConcurrent)
				}
			}

			if supportsStream, ok := resultMap["supports_streaming"].(bool); ok {
				if supportsStream != tt.wantSupportsStream {
					t.Errorf("expected supports_streaming=%v, got %v", tt.wantSupportsStream, supportsStream)
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
	if NewFeaturesQuery(nil, nil).Description() == "" {
		t.Error("expected non-empty description for FeaturesQuery")
	}
}

func TestQuery_Examples(t *testing.T) {
	if len(NewGetQuery(nil).Examples()) == 0 {
		t.Error("expected at least one example for GetQuery")
	}
	if len(NewListQuery(nil).Examples()) == 0 {
		t.Error("expected at least one example for ListQuery")
	}
	if len(NewFeaturesQuery(nil, nil).Examples()) == 0 {
		t.Error("expected at least one example for FeaturesQuery")
	}
}

func TestQueryImplementsInterface(t *testing.T) {
	var _ unit.Query = NewGetQuery(nil)
	var _ unit.Query = NewListQuery(nil)
	var _ unit.Query = NewFeaturesQuery(nil, nil)
}
