package service

import (
	"context"
	"errors"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestGetQuery_Name(t *testing.T) {
	q := NewGetQuery(nil, nil)
	if q.Name() != "service.get" {
		t.Errorf("expected name 'service.get', got '%s'", q.Name())
	}
}

func TestGetQuery_Domain(t *testing.T) {
	q := NewGetQuery(nil, nil)
	if q.Domain() != "service" {
		t.Errorf("expected domain 'service', got '%s'", q.Domain())
	}
}

func TestGetQuery_Schemas(t *testing.T) {
	q := NewGetQuery(nil, nil)

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
		name     string
		store    ServiceStore
		provider ServiceProvider
		input    any
		wantErr  bool
	}{
		{
			name:     "successful get",
			store:    createStoreWithService("svc-123", "model-1", ServiceStatusRunning),
			provider: &MockProvider{},
			input:    map[string]any{"service_id": "svc-123"},
			wantErr:  false,
		},
		{
			name:     "get stopped service",
			store:    createStoreWithService("svc-123", "model-1", ServiceStatusStopped),
			provider: &MockProvider{},
			input:    map[string]any{"service_id": "svc-123"},
			wantErr:  false,
		},
		{
			name:     "nil store",
			store:    nil,
			provider: &MockProvider{},
			input:    map[string]any{"service_id": "svc-123"},
			wantErr:  true,
		},
		{
			name:     "missing service_id",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    map[string]any{},
			wantErr:  true,
		},
		{
			name:     "service not found",
			store:    NewMemoryStore(),
			provider: &MockProvider{},
			input:    map[string]any{"service_id": "nonexistent"},
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
			q := NewGetQuery(tt.store, tt.provider)
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
		})
	}
}

func TestListQuery_Name(t *testing.T) {
	q := NewListQuery(nil)
	if q.Name() != "service.list" {
		t.Errorf("expected name 'service.list', got '%s'", q.Name())
	}
}

func TestListQuery_Execute(t *testing.T) {
	tests := []struct {
		name     string
		store    ServiceStore
		input    any
		wantErr  bool
		checkLen bool
	}{
		{
			name:     "list all services",
			store:    createStoreWithMultipleServices(),
			input:    map[string]any{},
			wantErr:  false,
			checkLen: true,
		},
		{
			name:     "list with status filter",
			store:    createStoreWithMultipleServices(),
			input:    map[string]any{"status": "running"},
			wantErr:  false,
			checkLen: true,
		},
		{
			name:     "list with model_id filter",
			store:    createStoreWithMultipleServices(),
			input:    map[string]any{"model_id": "model-1"},
			wantErr:  false,
			checkLen: true,
		},
		{
			name:     "list with pagination",
			store:    createStoreWithMultipleServices(),
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

			if _, exists := resultMap["services"]; !exists {
				t.Error("expected field 'services' not found")
			}
			if _, exists := resultMap["total"]; !exists {
				t.Error("expected field 'total' not found")
			}

			if tt.checkLen {
				services, ok := resultMap["services"].([]map[string]any)
				if !ok {
					t.Error("expected services to be []map[string]any")
				}
				if len(services) == 0 {
					t.Error("expected at least one service")
				}
			}
		})
	}
}

func TestRecommendQuery_Name(t *testing.T) {
	q := NewRecommendQuery(nil)
	if q.Name() != "service.recommend" {
		t.Errorf("expected name 'service.recommend', got '%s'", q.Name())
	}
}

func TestRecommendQuery_Execute(t *testing.T) {
	tests := []struct {
		name     string
		provider ServiceProvider
		input    any
		wantErr  bool
	}{
		{
			name:     "successful recommendation",
			provider: &MockProvider{},
			input:    map[string]any{"model_id": "llama3-70b"},
			wantErr:  false,
		},
		{
			name:     "recommendation with hint",
			provider: &MockProvider{},
			input:    map[string]any{"model_id": "llama3-70b", "hint": "high-throughput"},
			wantErr:  false,
		},
		{
			name:     "nil provider",
			provider: nil,
			input:    map[string]any{"model_id": "llama3-70b"},
			wantErr:  true,
		},
		{
			name:     "missing model_id",
			provider: &MockProvider{},
			input:    map[string]any{},
			wantErr:  true,
		},
		{
			name:     "provider error",
			provider: &MockProvider{recommendErr: errors.New("recommendation failed")},
			input:    map[string]any{"model_id": "llama3-70b"},
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
			q := NewRecommendQuery(tt.provider)
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

			if _, exists := resultMap["resource_class"]; !exists {
				t.Error("expected field 'resource_class' not found")
			}
			if _, exists := resultMap["replicas"]; !exists {
				t.Error("expected field 'replicas' not found")
			}
			if _, exists := resultMap["expected_throughput"]; !exists {
				t.Error("expected field 'expected_throughput' not found")
			}
		})
	}
}

func TestQuery_Description(t *testing.T) {
	if NewGetQuery(nil, nil).Description() == "" {
		t.Error("expected non-empty description for GetQuery")
	}
	if NewListQuery(nil).Description() == "" {
		t.Error("expected non-empty description for ListQuery")
	}
	if NewRecommendQuery(nil).Description() == "" {
		t.Error("expected non-empty description for RecommendQuery")
	}
}

func TestQuery_Examples(t *testing.T) {
	if len(NewGetQuery(nil, nil).Examples()) == 0 {
		t.Error("expected at least one example for GetQuery")
	}
	if len(NewListQuery(nil).Examples()) == 0 {
		t.Error("expected at least one example for ListQuery")
	}
	if len(NewRecommendQuery(nil).Examples()) == 0 {
		t.Error("expected at least one example for RecommendQuery")
	}
}

func TestQueryImplementsInterface(t *testing.T) {
	var _ unit.Query = NewGetQuery(nil, nil)
	var _ unit.Query = NewListQuery(nil)
	var _ unit.Query = NewRecommendQuery(nil)
}

func createStoreWithMultipleServices() ServiceStore {
	store := NewMemoryStore()
	_ = store.Create(context.Background(), createTestService("svc-1", "model-1", ServiceStatusRunning))
	_ = store.Create(context.Background(), createTestService("svc-2", "model-1", ServiceStatusStopped))
	_ = store.Create(context.Background(), createTestService("svc-3", "model-2", ServiceStatusRunning))
	return store
}
