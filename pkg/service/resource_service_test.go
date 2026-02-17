package service

import (
	"context"
	"errors"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/resource"
)

type mockResourceCommand struct {
	name    string
	execute func(ctx context.Context, input any) (any, error)
}

func (m *mockResourceCommand) Name() string              { return m.name }
func (m *mockResourceCommand) Domain() string            { return "resource" }
func (m *mockResourceCommand) InputSchema() unit.Schema  { return unit.Schema{} }
func (m *mockResourceCommand) OutputSchema() unit.Schema { return unit.Schema{} }
func (m *mockResourceCommand) Description() string       { return "" }
func (m *mockResourceCommand) Examples() []unit.Example  { return nil }
func (m *mockResourceCommand) Execute(ctx context.Context, input any) (any, error) {
	return m.execute(ctx, input)
}

type mockResourceQuery struct {
	name    string
	execute func(ctx context.Context, input any) (any, error)
}

func (m *mockResourceQuery) Name() string              { return m.name }
func (m *mockResourceQuery) Domain() string            { return "resource" }
func (m *mockResourceQuery) InputSchema() unit.Schema  { return unit.Schema{} }
func (m *mockResourceQuery) OutputSchema() unit.Schema { return unit.Schema{} }
func (m *mockResourceQuery) Description() string       { return "" }
func (m *mockResourceQuery) Examples() []unit.Example  { return nil }
func (m *mockResourceQuery) Execute(ctx context.Context, input any) (any, error) {
	return m.execute(ctx, input)
}

func TestResourceService_AllocateWithCheck_Success(t *testing.T) {
	registry := unit.NewRegistry()

	canAllocQuery := &mockResourceQuery{
		name: "resource.can_allocate",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{"can_allocate": true}, nil
		},
	}
	_ = registry.RegisterQuery(canAllocQuery)

	allocCmd := &mockResourceCommand{
		name: "resource.allocate",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{"slot_id": "slot-123"}, nil
		},
	}
	_ = registry.RegisterCommand(allocCmd)

	svc := NewResourceService(registry)
	result, err := svc.AllocateWithCheck(context.Background(), AllocateRequest{
		Name:        "test-app",
		Type:        "inference_native",
		MemoryBytes: 1024 * 1024 * 1024,
		GPUFraction: 0.5,
		Priority:    5,
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result.SlotID != "slot-123" {
		t.Errorf("expected slot_id 'slot-123', got: %s", result.SlotID)
	}
}

func TestResourceService_AllocateWithCheck_InvalidInput(t *testing.T) {
	registry := unit.NewRegistry()
	svc := NewResourceService(registry)

	_, err := svc.AllocateWithCheck(context.Background(), AllocateRequest{
		Name:        "",
		Type:        "inference_native",
		MemoryBytes: 1024,
	})
	if err == nil {
		t.Error("expected error for empty name")
	}

	_, err = svc.AllocateWithCheck(context.Background(), AllocateRequest{
		Name:        "test",
		Type:        "",
		MemoryBytes: 1024,
	})
	if err == nil {
		t.Error("expected error for empty type")
	}

	_, err = svc.AllocateWithCheck(context.Background(), AllocateRequest{
		Name:        "test",
		Type:        "inference_native",
		MemoryBytes: 0,
	})
	if err == nil {
		t.Error("expected error for zero memory")
	}

	_, err = svc.AllocateWithCheck(context.Background(), AllocateRequest{
		Name:        "test",
		Type:        "invalid_type",
		MemoryBytes: 1024,
	})
	if err == nil {
		t.Error("expected error for invalid type")
	}
}

func TestResourceService_AllocateWithCheck_InsufficientResources(t *testing.T) {
	registry := unit.NewRegistry()

	canAllocQuery := &mockResourceQuery{
		name: "resource.can_allocate",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"can_allocate": false,
				"reason":       "not enough memory",
			}, nil
		},
	}
	_ = registry.RegisterQuery(canAllocQuery)

	svc := NewResourceService(registry)
	_, err := svc.AllocateWithCheck(context.Background(), AllocateRequest{
		Name:        "test-app",
		Type:        "inference_native",
		MemoryBytes: 1024 * 1024 * 1024,
	})

	if err == nil {
		t.Fatal("expected error for insufficient resources")
	}
	if !errors.Is(err, resource.ErrInsufficientMemory) {
		t.Errorf("expected ErrInsufficientMemory, got: %v", err)
	}
}

func TestResourceService_ReleaseWithCleanup_Success(t *testing.T) {
	registry := unit.NewRegistry()

	allocQuery := &mockResourceQuery{
		name: "resource.allocations",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"allocations": []map[string]any{
					{"slot_id": "slot-123", "name": "test"},
				},
			}, nil
		},
	}
	_ = registry.RegisterQuery(allocQuery)

	releaseCmd := &mockResourceCommand{
		name: "resource.release",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{"success": true}, nil
		},
	}
	_ = registry.RegisterCommand(releaseCmd)

	svc := NewResourceService(registry)
	err := svc.ReleaseWithCleanup(context.Background(), "slot-123")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestResourceService_ReleaseWithCleanup_EmptySlotID(t *testing.T) {
	registry := unit.NewRegistry()
	svc := NewResourceService(registry)

	err := svc.ReleaseWithCleanup(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty slot_id")
	}
}

func TestResourceService_GetStatus_Success(t *testing.T) {
	registry := unit.NewRegistry()

	statusQuery := &mockResourceQuery{
		name: "resource.status",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"memory": map[string]any{
					"total":     16 * 1024 * 1024 * 1024,
					"used":      8 * 1024 * 1024 * 1024,
					"available": 8 * 1024 * 1024 * 1024,
				},
				"storage": map[string]any{
					"total":     500 * 1024 * 1024 * 1024,
					"used":      250 * 1024 * 1024 * 1024,
					"available": 250 * 1024 * 1024 * 1024,
				},
				"slots":    []map[string]any{},
				"pressure": "low",
			}, nil
		},
	}
	_ = registry.RegisterQuery(statusQuery)

	svc := NewResourceService(registry)
	status, err := svc.GetStatus(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if status.Pressure != "low" {
		t.Errorf("expected pressure 'low', got: %s", status.Pressure)
	}
	if status.Memory == nil {
		t.Error("expected memory info")
	}
}

func TestResourceService_CanAllocate_Success(t *testing.T) {
	registry := unit.NewRegistry()

	canAllocQuery := &mockResourceQuery{
		name: "resource.can_allocate",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"can_allocate": true,
			}, nil
		},
	}
	_ = registry.RegisterQuery(canAllocQuery)

	svc := NewResourceService(registry)
	result, err := svc.CanAllocate(context.Background(), 1024*1024*1024)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !result.CanAllocate {
		t.Error("expected can_allocate=true")
	}
}

func TestResourceService_CanAllocate_InvalidMemory(t *testing.T) {
	registry := unit.NewRegistry()
	svc := NewResourceService(registry)

	_, err := svc.CanAllocate(context.Background(), 0)
	if err == nil {
		t.Error("expected error for zero memory")
	}

	_, err = svc.CanAllocate(context.Background(), -100)
	if err == nil {
		t.Error("expected error for negative memory")
	}
}

func TestResourceService_GetBudgetInfo_Success(t *testing.T) {
	registry := unit.NewRegistry()

	budgetQuery := &mockResourceQuery{
		name: "resource.budget",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"total":    uint64(16 * 1024 * 1024 * 1024),
				"reserved": uint64(4 * 1024 * 1024 * 1024),
				"pools": map[string]any{
					"gpu": map[string]any{
						"total":     8 * 1024 * 1024 * 1024,
						"available": 4 * 1024 * 1024 * 1024,
					},
				},
			}, nil
		},
	}
	_ = registry.RegisterQuery(budgetQuery)

	svc := NewResourceService(registry)
	budget, err := svc.GetBudgetInfo(context.Background())

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if budget.Total != 16*1024*1024*1024 {
		t.Errorf("expected total 16GB, got: %d", budget.Total)
	}
}

func TestResourceService_UpdateSlotStatus_Success(t *testing.T) {
	registry := unit.NewRegistry()

	updateCmd := &mockResourceCommand{
		name: "resource.update_slot",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{"success": true}, nil
		},
	}
	_ = registry.RegisterCommand(updateCmd)

	svc := NewResourceService(registry)
	err := svc.UpdateSlotStatus(context.Background(), "slot-123", "active")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestResourceService_UpdateSlotStatus_InvalidInput(t *testing.T) {
	registry := unit.NewRegistry()
	svc := NewResourceService(registry)

	err := svc.UpdateSlotStatus(context.Background(), "", "active")
	if err == nil {
		t.Error("expected error for empty slot_id")
	}

	err = svc.UpdateSlotStatus(context.Background(), "slot-123", "")
	if err == nil {
		t.Error("expected error for empty status")
	}

	err = svc.UpdateSlotStatus(context.Background(), "slot-123", "invalid_status")
	if err == nil {
		t.Error("expected error for invalid status")
	}
}

func TestResourceService_ListAllocations_Success(t *testing.T) {
	registry := unit.NewRegistry()

	allocQuery := &mockResourceQuery{
		name: "resource.allocations",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"allocations": []map[string]any{
					{"slot_id": "slot-1", "name": "app1"},
					{"slot_id": "slot-2", "name": "app2"},
				},
			}, nil
		},
	}
	_ = registry.RegisterQuery(allocQuery)

	svc := NewResourceService(registry)
	allocations, err := svc.ListAllocations(context.Background(), "")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(allocations) != 2 {
		t.Errorf("expected 2 allocations, got: %d", len(allocations))
	}
}

func TestResourceService_GetSlot_Success(t *testing.T) {
	registry := unit.NewRegistry()

	allocQuery := &mockResourceQuery{
		name: "resource.allocations",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"allocations": []map[string]any{
					{"slot_id": "slot-123", "name": "test-app", "type": "inference_native"},
				},
			}, nil
		},
	}
	_ = registry.RegisterQuery(allocQuery)

	svc := NewResourceService(registry)
	slot, err := svc.GetSlot(context.Background(), "slot-123")

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if slot["slot_id"] != "slot-123" {
		t.Errorf("expected slot_id 'slot-123', got: %v", slot["slot_id"])
	}
}

func TestResourceService_GetSlot_NotFound(t *testing.T) {
	registry := unit.NewRegistry()

	allocQuery := &mockResourceQuery{
		name: "resource.allocations",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"allocations": []map[string]any{},
			}, nil
		},
	}
	_ = registry.RegisterQuery(allocQuery)

	svc := NewResourceService(registry)
	_, err := svc.GetSlot(context.Background(), "slot-nonexistent")

	if err == nil {
		t.Fatal("expected error for nonexistent slot")
	}
	if !errors.Is(err, resource.ErrSlotNotFound) {
		t.Errorf("expected ErrSlotNotFound, got: %v", err)
	}
}

func TestResourceService_UpdateSlotMemory_Success(t *testing.T) {
	registry := unit.NewRegistry()

	updateCmd := &mockResourceCommand{
		name: "resource.update_slot",
		execute: func(ctx context.Context, input any) (any, error) {
			inputMap := input.(map[string]any)
			if inputMap["memory_limit"] == nil {
				t.Error("expected memory_limit in input")
			}
			return map[string]any{"success": true}, nil
		},
	}
	_ = registry.RegisterCommand(updateCmd)

	svc := NewResourceService(registry)
	err := svc.UpdateSlotMemory(context.Background(), "slot-123", 2*1024*1024*1024)

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestResourceService_UpdateSlotMemory_InvalidInput(t *testing.T) {
	registry := unit.NewRegistry()
	svc := NewResourceService(registry)

	err := svc.UpdateSlotMemory(context.Background(), "", 1024)
	if err == nil {
		t.Error("expected error for empty slot_id")
	}

	err = svc.UpdateSlotMemory(context.Background(), "slot-123", 0)
	if err == nil {
		t.Error("expected error for zero memory")
	}

	err = svc.UpdateSlotMemory(context.Background(), "slot-123", -100)
	if err == nil {
		t.Error("expected error for negative memory")
	}
}
