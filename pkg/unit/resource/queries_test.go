package resource

import (
	"context"
	"errors"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestStatusQuery_Name(t *testing.T) {
	q := NewStatusQuery(nil, nil)
	if q.Name() != "resource.status" {
		t.Errorf("expected name 'resource.status', got '%s'", q.Name())
	}
}

func TestStatusQuery_Domain(t *testing.T) {
	q := NewStatusQuery(nil, nil)
	if q.Domain() != "resource" {
		t.Errorf("expected domain 'resource', got '%s'", q.Domain())
	}
}

func TestStatusQuery_Schemas(t *testing.T) {
	q := NewStatusQuery(nil, nil)

	inputSchema := q.InputSchema()
	if inputSchema.Type != "object" {
		t.Errorf("expected input schema type 'object', got '%s'", inputSchema.Type)
	}

	outputSchema := q.OutputSchema()
	if outputSchema.Type != "object" {
		t.Errorf("expected output schema type 'object', got '%s'", outputSchema.Type)
	}
}

func TestStatusQuery_Execute(t *testing.T) {
	store := NewMemoryStore()
	provider := &MockProvider{}

	tests := []struct {
		name     string
		provider ResourceProvider
		store    ResourceStore
		wantErr  bool
	}{
		{
			name:     "successful status",
			provider: provider,
			store:    store,
			wantErr:  false,
		},
		{
			name:     "nil provider",
			provider: nil,
			store:    store,
			wantErr:  true,
		},
		{
			name:     "provider error",
			provider: &MockProvider{statusErr: errors.New("provider error")},
			store:    store,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewStatusQuery(tt.provider, tt.store)
			result, err := q.Execute(context.Background(), map[string]any{})

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

			if _, ok := resultMap["memory"]; !ok {
				t.Error("expected 'memory' in result")
			}
			if _, ok := resultMap["storage"]; !ok {
				t.Error("expected 'storage' in result")
			}
			if _, ok := resultMap["pressure"]; !ok {
				t.Error("expected 'pressure' in result")
			}
		})
	}
}

func TestBudgetQuery_Name(t *testing.T) {
	q := NewBudgetQuery(nil)
	if q.Name() != "resource.budget" {
		t.Errorf("expected name 'resource.budget', got '%s'", q.Name())
	}
}

func TestBudgetQuery_Domain(t *testing.T) {
	q := NewBudgetQuery(nil)
	if q.Domain() != "resource" {
		t.Errorf("expected domain 'resource', got '%s'", q.Domain())
	}
}

func TestBudgetQuery_Execute(t *testing.T) {
	provider := &MockProvider{}

	tests := []struct {
		name     string
		provider ResourceProvider
		wantErr  bool
	}{
		{
			name:     "successful budget",
			provider: provider,
			wantErr:  false,
		},
		{
			name:     "nil provider",
			provider: nil,
			wantErr:  true,
		},
		{
			name:     "provider error",
			provider: &MockProvider{budgetErr: errors.New("provider error")},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewBudgetQuery(tt.provider)
			result, err := q.Execute(context.Background(), map[string]any{})

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

			if _, ok := resultMap["total"]; !ok {
				t.Error("expected 'total' in result")
			}
			if _, ok := resultMap["reserved"]; !ok {
				t.Error("expected 'reserved' in result")
			}
			if _, ok := resultMap["pools"]; !ok {
				t.Error("expected 'pools' in result")
			}
		})
	}
}

func TestAllocationsQuery_Name(t *testing.T) {
	q := NewAllocationsQuery(nil)
	if q.Name() != "resource.allocations" {
		t.Errorf("expected name 'resource.allocations', got '%s'", q.Name())
	}
}

func TestAllocationsQuery_Domain(t *testing.T) {
	q := NewAllocationsQuery(nil)
	if q.Domain() != "resource" {
		t.Errorf("expected domain 'resource', got '%s'", q.Domain())
	}
}

func TestAllocationsQuery_Execute(t *testing.T) {
	store := NewMemoryStore()
	slot1 := createTestSlot("slot-1", "test-slot-1", SlotTypeInferenceNative)
	slot2 := createTestSlot("slot-2", "test-slot-2", SlotTypeDockerContainer)
	_ = store.CreateSlot(context.Background(), slot1)
	_ = store.CreateSlot(context.Background(), slot2)

	tests := []struct {
		name            string
		store           ResourceStore
		input           any
		wantErr         bool
		wantAllocations int
	}{
		{
			name:            "list all allocations",
			store:           store,
			input:           map[string]any{},
			wantErr:         false,
			wantAllocations: 2,
		},
		{
			name:            "filter by slot_id",
			store:           store,
			input:           map[string]any{"slot_id": "slot-1"},
			wantErr:         false,
			wantAllocations: 1,
		},
		{
			name:            "filter by type",
			store:           store,
			input:           map[string]any{"type": "inference_native"},
			wantErr:         false,
			wantAllocations: 1,
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
			q := NewAllocationsQuery(tt.store)
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

			allocations, ok := resultMap["allocations"].([]map[string]any)
			if !ok {
				t.Error("expected 'allocations' to be []map[string]any")
				return
			}

			if len(allocations) != tt.wantAllocations {
				t.Errorf("expected %d allocations, got %d", tt.wantAllocations, len(allocations))
			}
		})
	}
}

func TestCanAllocateQuery_Name(t *testing.T) {
	q := NewCanAllocateQuery(nil)
	if q.Name() != "resource.can_allocate" {
		t.Errorf("expected name 'resource.can_allocate', got '%s'", q.Name())
	}
}

func TestCanAllocateQuery_Domain(t *testing.T) {
	q := NewCanAllocateQuery(nil)
	if q.Domain() != "resource" {
		t.Errorf("expected domain 'resource', got '%s'", q.Domain())
	}
}

func TestCanAllocateQuery_Execute(t *testing.T) {
	provider := &MockProvider{}

	tests := []struct {
		name         string
		provider     ResourceProvider
		input        any
		wantErr      bool
		wantCanAlloc bool
	}{
		{
			name:         "can allocate",
			provider:     provider,
			input:        map[string]any{"memory_bytes": uint64(8000000000), "priority": 10},
			wantErr:      false,
			wantCanAlloc: true,
		},
		{
			name:         "cannot allocate",
			provider:     &MockProvider{canAllocateRes: &CanAllocateResult{CanAllocate: false, Reason: "insufficient memory"}},
			input:        map[string]any{"memory_bytes": uint64(128000000000)},
			wantErr:      false,
			wantCanAlloc: false,
		},
		{
			name:     "nil provider",
			provider: nil,
			input:    map[string]any{"memory_bytes": uint64(8000000000)},
			wantErr:  true,
		},
		{
			name:     "missing memory_bytes",
			provider: provider,
			input:    map[string]any{"priority": 10},
			wantErr:  true,
		},
		{
			name:     "zero memory_bytes",
			provider: provider,
			input:    map[string]any{"memory_bytes": uint64(0)},
			wantErr:  true,
		},
		{
			name:     "provider error",
			provider: &MockProvider{canAllocateErr: errors.New("provider error")},
			input:    map[string]any{"memory_bytes": uint64(8000000000)},
			wantErr:  true,
		},
		{
			name:     "invalid input type",
			provider: provider,
			input:    "invalid",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := NewCanAllocateQuery(tt.provider)
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

			canAlloc, ok := resultMap["can_allocate"].(bool)
			if !ok {
				t.Error("expected 'can_allocate' to be bool")
				return
			}

			if canAlloc != tt.wantCanAlloc {
				t.Errorf("expected can_allocate=%v, got %v", tt.wantCanAlloc, canAlloc)
			}
		})
	}
}

func TestQuery_Description(t *testing.T) {
	statusQ := NewStatusQuery(nil, nil)
	if statusQ.Description() == "" {
		t.Error("expected non-empty description for StatusQuery")
	}

	budgetQ := NewBudgetQuery(nil)
	if budgetQ.Description() == "" {
		t.Error("expected non-empty description for BudgetQuery")
	}

	allocationsQ := NewAllocationsQuery(nil)
	if allocationsQ.Description() == "" {
		t.Error("expected non-empty description for AllocationsQuery")
	}

	canAllocQ := NewCanAllocateQuery(nil)
	if canAllocQ.Description() == "" {
		t.Error("expected non-empty description for CanAllocateQuery")
	}
}

func TestQuery_Examples(t *testing.T) {
	statusQ := NewStatusQuery(nil, nil)
	if len(statusQ.Examples()) == 0 {
		t.Error("expected at least one example for StatusQuery")
	}

	budgetQ := NewBudgetQuery(nil)
	if len(budgetQ.Examples()) == 0 {
		t.Error("expected at least one example for BudgetQuery")
	}

	allocationsQ := NewAllocationsQuery(nil)
	if len(allocationsQ.Examples()) == 0 {
		t.Error("expected at least one example for AllocationsQuery")
	}

	canAllocQ := NewCanAllocateQuery(nil)
	if len(canAllocQ.Examples()) == 0 {
		t.Error("expected at least one example for CanAllocateQuery")
	}
}

func TestQueryImplementsInterface(t *testing.T) {
	var _ unit.Query = NewStatusQuery(nil, nil)
	var _ unit.Query = NewBudgetQuery(nil)
	var _ unit.Query = NewAllocationsQuery(nil)
	var _ unit.Query = NewCanAllocateQuery(nil)
}
