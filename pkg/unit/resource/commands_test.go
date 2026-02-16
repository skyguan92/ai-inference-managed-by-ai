package resource

import (
	"context"
	"errors"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestAllocateCommand_Name(t *testing.T) {
	cmd := NewAllocateCommand(nil, nil)
	if cmd.Name() != "resource.allocate" {
		t.Errorf("expected name 'resource.allocate', got '%s'", cmd.Name())
	}
}

func TestAllocateCommand_Domain(t *testing.T) {
	cmd := NewAllocateCommand(nil, nil)
	if cmd.Domain() != "resource" {
		t.Errorf("expected domain 'resource', got '%s'", cmd.Domain())
	}
}

func TestAllocateCommand_Schemas(t *testing.T) {
	cmd := NewAllocateCommand(nil, nil)

	inputSchema := cmd.InputSchema()
	if inputSchema.Type != "object" {
		t.Errorf("expected input schema type 'object', got '%s'", inputSchema.Type)
	}
	if len(inputSchema.Required) != 3 {
		t.Errorf("expected 3 required fields, got %d", len(inputSchema.Required))
	}

	outputSchema := cmd.OutputSchema()
	if outputSchema.Type != "object" {
		t.Errorf("expected output schema type 'object', got '%s'", outputSchema.Type)
	}
}

func TestAllocateCommand_Execute(t *testing.T) {
	store := NewMemoryStore()
	provider := &MockProvider{}

	tests := []struct {
		name       string
		store      ResourceStore
		provider   ResourceProvider
		input      any
		wantErr    bool
		wantSlotID bool
	}{
		{
			name:     "successful allocation",
			store:    store,
			provider: provider,
			input: map[string]any{
				"name":         "test-slot",
				"type":         "inference_native",
				"memory_bytes": uint64(8000000000),
				"gpu_fraction": 1.0,
				"priority":     10,
			},
			wantErr:    false,
			wantSlotID: true,
		},
		{
			name:     "nil store",
			store:    nil,
			provider: provider,
			input: map[string]any{
				"name":         "test-slot",
				"type":         "inference_native",
				"memory_bytes": uint64(8000000000),
			},
			wantErr: true,
		},
		{
			name:     "missing name",
			store:    store,
			provider: provider,
			input: map[string]any{
				"type":         "inference_native",
				"memory_bytes": uint64(8000000000),
			},
			wantErr: true,
		},
		{
			name:     "missing type",
			store:    store,
			provider: provider,
			input: map[string]any{
				"name":         "test-slot",
				"memory_bytes": uint64(8000000000),
			},
			wantErr: true,
		},
		{
			name:     "invalid type",
			store:    store,
			provider: provider,
			input: map[string]any{
				"name":         "test-slot",
				"type":         "invalid_type",
				"memory_bytes": uint64(8000000000),
			},
			wantErr: true,
		},
		{
			name:     "missing memory_bytes",
			store:    store,
			provider: provider,
			input: map[string]any{
				"name": "test-slot",
				"type": "inference_native",
			},
			wantErr: true,
		},
		{
			name:     "zero memory_bytes",
			store:    store,
			provider: provider,
			input: map[string]any{
				"name":         "test-slot",
				"type":         "inference_native",
				"memory_bytes": uint64(0),
			},
			wantErr: true,
		},
		{
			name:     "cannot allocate",
			store:    store,
			provider: &MockProvider{canAllocateRes: &CanAllocateResult{CanAllocate: false, Reason: "insufficient memory"}},
			input: map[string]any{
				"name":         "test-slot",
				"type":         "inference_native",
				"memory_bytes": uint64(8000000000),
			},
			wantErr: true,
		},
		{
			name:     "provider error on can allocate",
			store:    store,
			provider: &MockProvider{canAllocateErr: errors.New("provider error")},
			input: map[string]any{
				"name":         "test-slot",
				"type":         "inference_native",
				"memory_bytes": uint64(8000000000),
			},
			wantErr: true,
		},
		{
			name:     "invalid input type",
			store:    store,
			provider: provider,
			input:    "invalid",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewAllocateCommand(tt.store, tt.provider)
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

			if tt.wantSlotID {
				slotID, ok := resultMap["slot_id"].(string)
				if !ok || slotID == "" {
					t.Error("expected non-empty slot_id")
				}
			}
		})
	}
}

func TestReleaseCommand_Name(t *testing.T) {
	cmd := NewReleaseCommand(nil)
	if cmd.Name() != "resource.release" {
		t.Errorf("expected name 'resource.release', got '%s'", cmd.Name())
	}
}

func TestReleaseCommand_Domain(t *testing.T) {
	cmd := NewReleaseCommand(nil)
	if cmd.Domain() != "resource" {
		t.Errorf("expected domain 'resource', got '%s'", cmd.Domain())
	}
}

func TestReleaseCommand_Execute(t *testing.T) {
	store := NewMemoryStore()
	slot := createTestSlot("slot-123", "test-slot", SlotTypeInferenceNative)
	_ = store.CreateSlot(context.Background(), slot)

	tests := []struct {
		name        string
		store       ResourceStore
		input       any
		wantErr     bool
		wantSuccess bool
	}{
		{
			name:        "successful release",
			store:       store,
			input:       map[string]any{"slot_id": "slot-123"},
			wantErr:     false,
			wantSuccess: true,
		},
		{
			name:    "nil store",
			store:   nil,
			input:   map[string]any{"slot_id": "slot-123"},
			wantErr: true,
		},
		{
			name:    "missing slot_id",
			store:   store,
			input:   map[string]any{},
			wantErr: true,
		},
		{
			name:    "slot not found",
			store:   store,
			input:   map[string]any{"slot_id": "nonexistent"},
			wantErr: true,
		},
		{
			name:    "invalid input type",
			store:   store,
			input:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewReleaseCommand(tt.store)
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

			success, ok := resultMap["success"].(bool)
			if !ok || success != tt.wantSuccess {
				t.Errorf("expected success=%v, got %v", tt.wantSuccess, success)
			}
		})
	}
}

func TestUpdateSlotCommand_Name(t *testing.T) {
	cmd := NewUpdateSlotCommand(nil)
	if cmd.Name() != "resource.update_slot" {
		t.Errorf("expected name 'resource.update_slot', got '%s'", cmd.Name())
	}
}

func TestUpdateSlotCommand_Domain(t *testing.T) {
	cmd := NewUpdateSlotCommand(nil)
	if cmd.Domain() != "resource" {
		t.Errorf("expected domain 'resource', got '%s'", cmd.Domain())
	}
}

func TestUpdateSlotCommand_Execute(t *testing.T) {
	store := NewMemoryStore()
	slot := createTestSlot("slot-456", "test-slot", SlotTypeInferenceNative)
	_ = store.CreateSlot(context.Background(), slot)

	tests := []struct {
		name        string
		store       ResourceStore
		input       any
		wantErr     bool
		wantSuccess bool
	}{
		{
			name:        "update memory limit",
			store:       store,
			input:       map[string]any{"slot_id": "slot-456", "memory_limit": uint64(16000000000)},
			wantErr:     false,
			wantSuccess: true,
		},
		{
			name:        "update status",
			store:       store,
			input:       map[string]any{"slot_id": "slot-456", "status": "idle"},
			wantErr:     false,
			wantSuccess: true,
		},
		{
			name:        "update both",
			store:       store,
			input:       map[string]any{"slot_id": "slot-456", "memory_limit": uint64(32000000000), "status": "active"},
			wantErr:     false,
			wantSuccess: true,
		},
		{
			name:    "nil store",
			store:   nil,
			input:   map[string]any{"slot_id": "slot-456", "memory_limit": uint64(16000000000)},
			wantErr: true,
		},
		{
			name:    "missing slot_id",
			store:   store,
			input:   map[string]any{"memory_limit": uint64(16000000000)},
			wantErr: true,
		},
		{
			name:    "invalid status",
			store:   store,
			input:   map[string]any{"slot_id": "slot-456", "status": "invalid"},
			wantErr: true,
		},
		{
			name:    "slot not found",
			store:   store,
			input:   map[string]any{"slot_id": "nonexistent", "memory_limit": uint64(16000000000)},
			wantErr: true,
		},
		{
			name:    "invalid input type",
			store:   store,
			input:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewUpdateSlotCommand(tt.store)
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

			success, ok := resultMap["success"].(bool)
			if !ok || success != tt.wantSuccess {
				t.Errorf("expected success=%v, got %v", tt.wantSuccess, success)
			}
		})
	}
}

func TestCommand_Description(t *testing.T) {
	allocCmd := NewAllocateCommand(nil, nil)
	if allocCmd.Description() == "" {
		t.Error("expected non-empty description for AllocateCommand")
	}

	releaseCmd := NewReleaseCommand(nil)
	if releaseCmd.Description() == "" {
		t.Error("expected non-empty description for ReleaseCommand")
	}

	updateCmd := NewUpdateSlotCommand(nil)
	if updateCmd.Description() == "" {
		t.Error("expected non-empty description for UpdateSlotCommand")
	}
}

func TestCommand_Examples(t *testing.T) {
	allocCmd := NewAllocateCommand(nil, nil)
	if len(allocCmd.Examples()) == 0 {
		t.Error("expected at least one example for AllocateCommand")
	}

	releaseCmd := NewReleaseCommand(nil)
	if len(releaseCmd.Examples()) == 0 {
		t.Error("expected at least one example for ReleaseCommand")
	}

	updateCmd := NewUpdateSlotCommand(nil)
	if len(updateCmd.Examples()) == 0 {
		t.Error("expected at least one example for UpdateSlotCommand")
	}
}

func TestCommandImplementsInterface(t *testing.T) {
	var _ unit.Command = NewAllocateCommand(nil, nil)
	var _ unit.Command = NewReleaseCommand(nil)
	var _ unit.Command = NewUpdateSlotCommand(nil)
}
