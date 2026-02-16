package resource

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestStatusResource_URI(t *testing.T) {
	r := NewStatusResource(nil, nil)
	if r.URI() != "asms://resource/status" {
		t.Errorf("expected URI 'asms://resource/status', got '%s'", r.URI())
	}
}

func TestStatusResource_Domain(t *testing.T) {
	r := NewStatusResource(nil, nil)
	if r.Domain() != "resource" {
		t.Errorf("expected domain 'resource', got '%s'", r.Domain())
	}
}

func TestStatusResource_Get(t *testing.T) {
	store := NewMemoryStore()
	provider := &MockProvider{}

	tests := []struct {
		name     string
		provider ResourceProvider
		store    ResourceStore
		wantErr  bool
	}{
		{
			name:     "successful get",
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
			r := NewStatusResource(tt.provider, tt.store)
			result, err := r.Get(context.Background())

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
		})
	}
}

func TestStatusResource_Watch(t *testing.T) {
	provider := &MockProvider{}
	r := NewStatusResource(provider, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ch, err := r.Watch(ctx)
	if err != nil {
		t.Errorf("unexpected error starting watch: %v", err)
		return
	}

	select {
	case update, ok := <-ch:
		if ok && update.Error != nil {
			t.Errorf("unexpected error in update: %v", update.Error)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("expected update within timeout")
	}
}

func TestBudgetResource_URI(t *testing.T) {
	r := NewBudgetResource(nil)
	if r.URI() != "asms://resource/budget" {
		t.Errorf("expected URI 'asms://resource/budget', got '%s'", r.URI())
	}
}

func TestBudgetResource_Domain(t *testing.T) {
	r := NewBudgetResource(nil)
	if r.Domain() != "resource" {
		t.Errorf("expected domain 'resource', got '%s'", r.Domain())
	}
}

func TestBudgetResource_Get(t *testing.T) {
	provider := &MockProvider{}

	tests := []struct {
		name     string
		provider ResourceProvider
		wantErr  bool
	}{
		{
			name:     "successful get",
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
			r := NewBudgetResource(tt.provider)
			result, err := r.Get(context.Background())

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
		})
	}
}

func TestBudgetResource_Watch(t *testing.T) {
	provider := &MockProvider{}
	r := NewBudgetResource(provider)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ch, err := r.Watch(ctx)
	if err != nil {
		t.Errorf("unexpected error starting watch: %v", err)
		return
	}

	select {
	case update, ok := <-ch:
		if ok && update.Error != nil {
			t.Errorf("unexpected error in update: %v", update.Error)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("expected update within timeout")
	}
}

func TestAllocationsResource_URI(t *testing.T) {
	r := NewAllocationsResource(nil)
	if r.URI() != "asms://resource/allocations" {
		t.Errorf("expected URI 'asms://resource/allocations', got '%s'", r.URI())
	}
}

func TestAllocationsResource_Domain(t *testing.T) {
	r := NewAllocationsResource(nil)
	if r.Domain() != "resource" {
		t.Errorf("expected domain 'resource', got '%s'", r.Domain())
	}
}

func TestAllocationsResource_Get(t *testing.T) {
	store := NewMemoryStore()
	slot := createTestSlot("slot-1", "test-slot", SlotTypeInferenceNative)
	_ = store.CreateSlot(context.Background(), slot)

	tests := []struct {
		name    string
		store   ResourceStore
		wantErr bool
	}{
		{
			name:    "successful get",
			store:   store,
			wantErr: false,
		},
		{
			name:    "nil store",
			store:   nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewAllocationsResource(tt.store)
			result, err := r.Get(context.Background())

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

			if _, ok := resultMap["allocations"]; !ok {
				t.Error("expected 'allocations' in result")
			}
		})
	}
}

func TestAllocationsResource_Watch(t *testing.T) {
	store := NewMemoryStore()
	r := NewAllocationsResource(store)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	ch, err := r.Watch(ctx)
	if err != nil {
		t.Errorf("unexpected error starting watch: %v", err)
		return
	}

	select {
	case update, ok := <-ch:
		if ok && update.Error != nil {
			t.Errorf("unexpected error in update: %v", update.Error)
		}
	case <-time.After(200 * time.Millisecond):
		t.Error("expected update within timeout")
	}
}

func TestParseResourceURI(t *testing.T) {
	tests := []struct {
		uri      string
		wantType string
		wantOK   bool
	}{
		{"asms://resource/status", "status", true},
		{"asms://resource/budget", "budget", true},
		{"asms://resource/allocations", "allocations", true},
		{"asms://device/gpu-0/info", "", false},
		{"asms://model/123", "", false},
		{"invalid", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.uri, func(t *testing.T) {
			resourceType, ok := ParseResourceURI(tt.uri)
			if ok != tt.wantOK {
				t.Errorf("expected ok=%v, got %v", tt.wantOK, ok)
			}
			if resourceType != tt.wantType {
				t.Errorf("expected type='%s', got '%s'", tt.wantType, resourceType)
			}
		})
	}
}

func TestResource_Schema(t *testing.T) {
	statusR := NewStatusResource(nil, nil)
	if statusR.Schema().Type != "object" {
		t.Error("expected object schema for StatusResource")
	}

	budgetR := NewBudgetResource(nil)
	if budgetR.Schema().Type != "object" {
		t.Error("expected object schema for BudgetResource")
	}

	allocationsR := NewAllocationsResource(nil)
	if allocationsR.Schema().Type != "object" {
		t.Error("expected object schema for AllocationsResource")
	}
}

func TestResourceImplementsInterface(t *testing.T) {
	var _ unit.Resource = NewStatusResource(nil, nil)
	var _ unit.Resource = NewBudgetResource(nil)
	var _ unit.Resource = NewAllocationsResource(nil)
}
