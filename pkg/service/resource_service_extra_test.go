package service

import (
	"context"
	"errors"
	"testing"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/resource"
)

func TestResourceService_AllocateWithCheck_QueryNotFound(t *testing.T) {
	registry := unit.NewRegistry()
	svc := NewResourceService(registry)

	_, err := svc.AllocateWithCheck(context.Background(), AllocateRequest{
		Name:        "test-app",
		Type:        "inference_native",
		MemoryBytes: 1024,
	})
	if err == nil {
		t.Error("expected error when resource.can_allocate query not found")
	}
}

func TestResourceService_AllocateWithCheck_QueryError(t *testing.T) {
	registry := unit.NewRegistry()
	canAllocQuery := &mockResourceQuery{
		name: "resource.can_allocate",
		execute: func(ctx context.Context, input any) (any, error) {
			return nil, errors.New("query execution error")
		},
	}
	_ = registry.RegisterQuery(canAllocQuery)

	svc := NewResourceService(registry)
	_, err := svc.AllocateWithCheck(context.Background(), AllocateRequest{
		Name:        "test-app",
		Type:        "inference_native",
		MemoryBytes: 1024,
	})
	if err == nil {
		t.Error("expected error when query returns error")
	}
}

func TestResourceService_AllocateWithCheck_InsufficientNoReason(t *testing.T) {
	registry := unit.NewRegistry()
	canAllocQuery := &mockResourceQuery{
		name: "resource.can_allocate",
		execute: func(ctx context.Context, input any) (any, error) {
			// Return can_allocate=false without a reason
			return map[string]any{"can_allocate": false}, nil
		},
	}
	_ = registry.RegisterQuery(canAllocQuery)

	svc := NewResourceService(registry)
	_, err := svc.AllocateWithCheck(context.Background(), AllocateRequest{
		Name:        "test-app",
		Type:        "inference_native",
		MemoryBytes: 1024,
	})
	if err == nil {
		t.Error("expected error for insufficient resources")
	}
	if !errors.Is(err, resource.ErrInsufficientMemory) {
		t.Errorf("expected ErrInsufficientMemory, got: %v", err)
	}
}

func TestResourceService_AllocateWithCheck_AllocCmdNotFound(t *testing.T) {
	registry := unit.NewRegistry()
	canAllocQuery := &mockResourceQuery{
		name: "resource.can_allocate",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{"can_allocate": true}, nil
		},
	}
	_ = registry.RegisterQuery(canAllocQuery)
	// No resource.allocate command registered

	svc := NewResourceService(registry)
	_, err := svc.AllocateWithCheck(context.Background(), AllocateRequest{
		Name:        "test-app",
		Type:        "inference_native",
		MemoryBytes: 1024,
	})
	if err == nil {
		t.Error("expected error when resource.allocate command not found")
	}
}

func TestResourceService_AllocateWithCheck_AllocCmdError(t *testing.T) {
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
			return nil, errors.New("alloc error")
		},
	}
	_ = registry.RegisterCommand(allocCmd)

	svc := NewResourceService(registry)
	_, err := svc.AllocateWithCheck(context.Background(), AllocateRequest{
		Name:        "test-app",
		Type:        "inference_native",
		MemoryBytes: 1024,
	})
	if err == nil {
		t.Error("expected error when allocate command returns error")
	}
}

func TestResourceService_AllocateWithCheck_DefaultPriority(t *testing.T) {
	registry := unit.NewRegistry()

	var capturedInput map[string]any
	canAllocQuery := &mockResourceQuery{
		name: "resource.can_allocate",
		execute: func(ctx context.Context, input any) (any, error) {
			capturedInput = input.(map[string]any)
			return map[string]any{"can_allocate": true}, nil
		},
	}
	_ = registry.RegisterQuery(canAllocQuery)

	allocCmd := &mockResourceCommand{
		name: "resource.allocate",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{"slot_id": "slot-default"}, nil
		},
	}
	_ = registry.RegisterCommand(allocCmd)

	svc := NewResourceService(registry)
	_, err := svc.AllocateWithCheck(context.Background(), AllocateRequest{
		Name:        "test-app",
		Type:        "inference_native",
		MemoryBytes: 1024,
		Priority:    0, // Should default to 5
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedInput["priority"] != 5 {
		t.Errorf("expected default priority 5, got %v", capturedInput["priority"])
	}
}

func TestResourceService_ReleaseWithCleanup_CommandNotFound(t *testing.T) {
	registry := unit.NewRegistry()
	// No resource.release command registered

	svc := NewResourceService(registry)
	err := svc.ReleaseWithCleanup(context.Background(), "slot-123")
	if err == nil {
		t.Error("expected error when resource.release command not found")
	}
}

func TestResourceService_ReleaseWithCleanup_SlotNotFound(t *testing.T) {
	registry := unit.NewRegistry()

	// allocations query returns empty list -> slot not found
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
	err := svc.ReleaseWithCleanup(context.Background(), "slot-missing")
	if err == nil {
		t.Error("expected error for missing slot")
	}
	if !errors.Is(err, resource.ErrSlotNotFound) {
		t.Errorf("expected ErrSlotNotFound, got: %v", err)
	}
}

func TestResourceService_ReleaseWithCleanup_CommandError(t *testing.T) {
	registry := unit.NewRegistry()

	releaseCmd := &mockResourceCommand{
		name: "resource.release",
		execute: func(ctx context.Context, input any) (any, error) {
			return nil, errors.New("release error")
		},
	}
	_ = registry.RegisterCommand(releaseCmd)

	svc := NewResourceService(registry)
	err := svc.ReleaseWithCleanup(context.Background(), "slot-123")
	if err == nil {
		t.Error("expected error when release command fails")
	}
}

func TestResourceService_GetStatus_QueryNotFound(t *testing.T) {
	registry := unit.NewRegistry()
	svc := NewResourceService(registry)

	_, err := svc.GetStatus(context.Background())
	if err == nil {
		t.Error("expected error when resource.status query not found")
	}
}

func TestResourceService_GetStatus_QueryError(t *testing.T) {
	registry := unit.NewRegistry()
	statusQuery := &mockResourceQuery{
		name: "resource.status",
		execute: func(ctx context.Context, input any) (any, error) {
			return nil, errors.New("status error")
		},
	}
	_ = registry.RegisterQuery(statusQuery)

	svc := NewResourceService(registry)
	_, err := svc.GetStatus(context.Background())
	if err == nil {
		t.Error("expected error when status query returns error")
	}
}

func TestResourceService_CanAllocate_QueryNotFound(t *testing.T) {
	registry := unit.NewRegistry()
	svc := NewResourceService(registry)

	_, err := svc.CanAllocate(context.Background(), 1024)
	if err == nil {
		t.Error("expected error when resource.can_allocate query not found")
	}
}

func TestResourceService_CanAllocate_WithReason(t *testing.T) {
	registry := unit.NewRegistry()
	canAllocQuery := &mockResourceQuery{
		name: "resource.can_allocate",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{
				"can_allocate": false,
				"reason":       "gpu memory full",
			}, nil
		},
	}
	_ = registry.RegisterQuery(canAllocQuery)

	svc := NewResourceService(registry)
	result, err := svc.CanAllocate(context.Background(), 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.CanAllocate {
		t.Error("expected can_allocate=false")
	}
	if result.Reason != "gpu memory full" {
		t.Errorf("expected reason 'gpu memory full', got %q", result.Reason)
	}
}

func TestResourceService_GetBudgetInfo_QueryNotFound(t *testing.T) {
	registry := unit.NewRegistry()
	svc := NewResourceService(registry)

	_, err := svc.GetBudgetInfo(context.Background())
	if err == nil {
		t.Error("expected error when resource.budget query not found")
	}
}

func TestResourceService_GetBudgetInfo_QueryError(t *testing.T) {
	registry := unit.NewRegistry()
	budgetQuery := &mockResourceQuery{
		name: "resource.budget",
		execute: func(ctx context.Context, input any) (any, error) {
			return nil, errors.New("budget error")
		},
	}
	_ = registry.RegisterQuery(budgetQuery)

	svc := NewResourceService(registry)
	_, err := svc.GetBudgetInfo(context.Background())
	if err == nil {
		t.Error("expected error when budget query returns error")
	}
}

func TestResourceService_UpdateSlotStatus_CommandNotFound(t *testing.T) {
	registry := unit.NewRegistry()
	svc := NewResourceService(registry)

	err := svc.UpdateSlotStatus(context.Background(), "slot-123", "active")
	if err == nil {
		t.Error("expected error when resource.update_slot command not found")
	}
}

func TestResourceService_UpdateSlotStatus_CommandError(t *testing.T) {
	registry := unit.NewRegistry()
	updateCmd := &mockResourceCommand{
		name: "resource.update_slot",
		execute: func(ctx context.Context, input any) (any, error) {
			return nil, errors.New("update error")
		},
	}
	_ = registry.RegisterCommand(updateCmd)

	svc := NewResourceService(registry)
	err := svc.UpdateSlotStatus(context.Background(), "slot-123", "active")
	if err == nil {
		t.Error("expected error when update command returns error")
	}
}

func TestResourceService_ListAllocations_QueryNotFound(t *testing.T) {
	registry := unit.NewRegistry()
	svc := NewResourceService(registry)

	_, err := svc.ListAllocations(context.Background(), "")
	if err == nil {
		t.Error("expected error when resource.allocations query not found")
	}
}

func TestResourceService_ListAllocations_InvalidType(t *testing.T) {
	registry := unit.NewRegistry()
	svc := NewResourceService(registry)

	_, err := svc.ListAllocations(context.Background(), "invalid_type")
	if err == nil {
		t.Error("expected error for invalid slot type")
	}
}

func TestResourceService_ListAllocations_QueryError(t *testing.T) {
	registry := unit.NewRegistry()
	allocQuery := &mockResourceQuery{
		name: "resource.allocations",
		execute: func(ctx context.Context, input any) (any, error) {
			return nil, errors.New("alloc list error")
		},
	}
	_ = registry.RegisterQuery(allocQuery)

	svc := NewResourceService(registry)
	_, err := svc.ListAllocations(context.Background(), "")
	if err == nil {
		t.Error("expected error when query returns error")
	}
}

func TestResourceService_ListAllocations_EmptyResult(t *testing.T) {
	registry := unit.NewRegistry()
	allocQuery := &mockResourceQuery{
		name: "resource.allocations",
		execute: func(ctx context.Context, input any) (any, error) {
			// Return a result without "allocations" key
			return map[string]any{"count": 0}, nil
		},
	}
	_ = registry.RegisterQuery(allocQuery)

	svc := NewResourceService(registry)
	result, err := svc.ListAllocations(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d items", len(result))
	}
}

func TestResourceService_GetSlot_EmptySlotID(t *testing.T) {
	registry := unit.NewRegistry()
	svc := NewResourceService(registry)

	_, err := svc.GetSlot(context.Background(), "")
	if err == nil {
		t.Error("expected error for empty slot_id")
	}
}

func TestResourceService_GetSlot_QueryNotFound(t *testing.T) {
	registry := unit.NewRegistry()
	svc := NewResourceService(registry)

	_, err := svc.GetSlot(context.Background(), "slot-123")
	if err == nil {
		t.Error("expected error when resource.allocations query not found")
	}
}

func TestResourceService_UpdateSlotMemory_CommandNotFound(t *testing.T) {
	registry := unit.NewRegistry()
	svc := NewResourceService(registry)

	err := svc.UpdateSlotMemory(context.Background(), "slot-123", 1024)
	if err == nil {
		t.Error("expected error when resource.update_slot command not found")
	}
}

func TestResourceService_UpdateSlotMemory_CommandError(t *testing.T) {
	registry := unit.NewRegistry()
	updateCmd := &mockResourceCommand{
		name: "resource.update_slot",
		execute: func(ctx context.Context, input any) (any, error) {
			return nil, errors.New("memory update error")
		},
	}
	_ = registry.RegisterCommand(updateCmd)

	svc := NewResourceService(registry)
	err := svc.UpdateSlotMemory(context.Background(), "slot-123", 2*1024*1024*1024)
	if err == nil {
		t.Error("expected error when update command returns error")
	}
}
