package service

import (
	"context"
	"fmt"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/resource"
)

type AllocateRequest struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"`
	MemoryBytes int64   `json:"memory_bytes"`
	GPUFraction float64 `json:"gpu_fraction"`
	Priority    int     `json:"priority"`
}

type AllocateResult struct {
	SlotID string `json:"slot_id"`
}

type ResourceStatusResult struct {
	Memory   map[string]any   `json:"memory"`
	Storage  map[string]any   `json:"storage"`
	Slots    []map[string]any `json:"slots"`
	Pressure string           `json:"pressure"`
}

type BudgetInfoResult struct {
	Total    uint64         `json:"total"`
	Reserved uint64         `json:"reserved"`
	Pools    map[string]any `json:"pools"`
}

type CanAllocateResult struct {
	CanAllocate bool   `json:"can_allocate"`
	Reason      string `json:"reason,omitempty"`
}

type ResourceService struct {
	registry *unit.Registry
}

func NewResourceService(registry *unit.Registry) *ResourceService {
	return &ResourceService{registry: registry}
}

func (s *ResourceService) AllocateWithCheck(ctx context.Context, req AllocateRequest) (*AllocateResult, error) {
	if req.MemoryBytes <= 0 {
		return nil, fmt.Errorf("invalid memory bytes: must be positive")
	}
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if req.Type == "" {
		return nil, fmt.Errorf("type is required")
	}
	if !resource.IsValidSlotType(resource.SlotType(req.Type)) {
		return nil, fmt.Errorf("invalid slot type: %s", req.Type)
	}

	canAllocQuery := s.registry.GetQuery("resource.can_allocate")
	if canAllocQuery == nil {
		return nil, fmt.Errorf("resource.can_allocate query not found")
	}

	priority := req.Priority
	if priority == 0 {
		priority = 5
	}

	canResult, err := canAllocQuery.Execute(ctx, map[string]any{
		"memory_bytes": req.MemoryBytes,
		"priority":     priority,
	})
	if err != nil {
		return nil, fmt.Errorf("check allocation: %w", err)
	}

	canMap, ok := canResult.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid can_allocate result type")
	}

	if canAlloc, _ := canMap["can_allocate"].(bool); !canAlloc {
		reason, _ := canMap["reason"].(string)
		if reason == "" {
			reason = "insufficient resources"
		}
		return nil, fmt.Errorf("%s: %w", reason, resource.ErrInsufficientMemory)
	}

	allocCmd := s.registry.GetCommand("resource.allocate")
	if allocCmd == nil {
		return nil, fmt.Errorf("resource.allocate command not found")
	}

	allocResult, err := allocCmd.Execute(ctx, map[string]any{
		"name":         req.Name,
		"type":         req.Type,
		"memory_bytes": req.MemoryBytes,
		"gpu_fraction": req.GPUFraction,
		"priority":     priority,
	})
	if err != nil {
		return nil, fmt.Errorf("allocate resource: %w", err)
	}

	allocMap, ok := allocResult.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid allocate result type")
	}

	slotID, _ := allocMap["slot_id"].(string)
	return &AllocateResult{SlotID: slotID}, nil
}

func (s *ResourceService) ReleaseWithCleanup(ctx context.Context, slotID string) error {
	if slotID == "" {
		return fmt.Errorf("slot_id is required")
	}

	allocQuery := s.registry.GetQuery("resource.allocations")
	if allocQuery != nil {
		allocResult, err := allocQuery.Execute(ctx, map[string]any{"slot_id": slotID})
		if err == nil {
			if allocMap, ok := allocResult.(map[string]any); ok {
				if allocs, ok := allocMap["allocations"].([]map[string]any); ok && len(allocs) == 0 {
					return resource.ErrSlotNotFound
				}
			}
		}
	}

	releaseCmd := s.registry.GetCommand("resource.release")
	if releaseCmd == nil {
		return fmt.Errorf("resource.release command not found")
	}

	_, err := releaseCmd.Execute(ctx, map[string]any{"slot_id": slotID})
	if err != nil {
		return fmt.Errorf("release slot %s: %w", slotID, err)
	}

	return nil
}

func (s *ResourceService) GetStatus(ctx context.Context) (*ResourceStatusResult, error) {
	statusQuery := s.registry.GetQuery("resource.status")
	if statusQuery == nil {
		return nil, fmt.Errorf("resource.status query not found")
	}

	result, err := statusQuery.Execute(ctx, map[string]any{})
	if err != nil {
		return nil, fmt.Errorf("get resource status: %w", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid status result type")
	}

	status := &ResourceStatusResult{}

	if mem, ok := resultMap["memory"].(map[string]any); ok {
		status.Memory = mem
	}
	if stor, ok := resultMap["storage"].(map[string]any); ok {
		status.Storage = stor
	}
	if slots, ok := resultMap["slots"].([]map[string]any); ok {
		status.Slots = slots
	}
	if pressure, ok := resultMap["pressure"].(string); ok {
		status.Pressure = pressure
	}

	return status, nil
}

func (s *ResourceService) CanAllocate(ctx context.Context, memoryBytes int64) (*CanAllocateResult, error) {
	if memoryBytes <= 0 {
		return nil, fmt.Errorf("memory bytes must be positive")
	}

	canAllocQuery := s.registry.GetQuery("resource.can_allocate")
	if canAllocQuery == nil {
		return nil, fmt.Errorf("resource.can_allocate query not found")
	}

	result, err := canAllocQuery.Execute(ctx, map[string]any{
		"memory_bytes": memoryBytes,
	})
	if err != nil {
		return nil, fmt.Errorf("check allocation: %w", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid can_allocate result type")
	}

	allocResult := &CanAllocateResult{}
	if canAlloc, ok := resultMap["can_allocate"].(bool); ok {
		allocResult.CanAllocate = canAlloc
	}
	if reason, ok := resultMap["reason"].(string); ok {
		allocResult.Reason = reason
	}

	return allocResult, nil
}

func (s *ResourceService) GetBudgetInfo(ctx context.Context) (*BudgetInfoResult, error) {
	budgetQuery := s.registry.GetQuery("resource.budget")
	if budgetQuery == nil {
		return nil, fmt.Errorf("resource.budget query not found")
	}

	result, err := budgetQuery.Execute(ctx, map[string]any{})
	if err != nil {
		return nil, fmt.Errorf("get resource budget: %w", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid budget result type")
	}

	budget := &BudgetInfoResult{
		Pools: make(map[string]any),
	}

	if total, ok := resultMap["total"].(uint64); ok {
		budget.Total = total
	}
	if reserved, ok := resultMap["reserved"].(uint64); ok {
		budget.Reserved = reserved
	}
	if pools, ok := resultMap["pools"].(map[string]any); ok {
		budget.Pools = pools
	}

	return budget, nil
}

func (s *ResourceService) UpdateSlotStatus(ctx context.Context, slotID string, status string) error {
	if slotID == "" {
		return fmt.Errorf("slot_id is required")
	}
	if status == "" {
		return fmt.Errorf("status is required")
	}
	if !resource.IsValidSlotStatus(resource.SlotStatus(status)) {
		return fmt.Errorf("invalid slot status: %s", status)
	}

	updateCmd := s.registry.GetCommand("resource.update_slot")
	if updateCmd == nil {
		return fmt.Errorf("resource.update_slot command not found")
	}

	_, err := updateCmd.Execute(ctx, map[string]any{
		"slot_id": slotID,
		"status":  status,
	})
	if err != nil {
		return fmt.Errorf("update slot %s status: %w", slotID, err)
	}

	return nil
}

func (s *ResourceService) ListAllocations(ctx context.Context, slotType string) ([]map[string]any, error) {
	allocQuery := s.registry.GetQuery("resource.allocations")
	if allocQuery == nil {
		return nil, fmt.Errorf("resource.allocations query not found")
	}

	input := map[string]any{}
	if slotType != "" {
		if !resource.IsValidSlotType(resource.SlotType(slotType)) {
			return nil, fmt.Errorf("invalid slot type: %s", slotType)
		}
		input["type"] = slotType
	}

	result, err := allocQuery.Execute(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("list allocations: %w", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid allocations result type")
	}

	if allocations, ok := resultMap["allocations"].([]map[string]any); ok {
		return allocations, nil
	}

	return []map[string]any{}, nil
}

func (s *ResourceService) GetSlot(ctx context.Context, slotID string) (map[string]any, error) {
	if slotID == "" {
		return nil, fmt.Errorf("slot_id is required")
	}

	allocQuery := s.registry.GetQuery("resource.allocations")
	if allocQuery == nil {
		return nil, fmt.Errorf("resource.allocations query not found")
	}

	result, err := allocQuery.Execute(ctx, map[string]any{"slot_id": slotID})
	if err != nil {
		return nil, fmt.Errorf("get slot: %w", err)
	}

	resultMap, ok := result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid allocations result type")
	}

	if allocations, ok := resultMap["allocations"].([]map[string]any); ok && len(allocations) > 0 {
		return allocations[0], nil
	}

	return nil, resource.ErrSlotNotFound
}

func (s *ResourceService) UpdateSlotMemory(ctx context.Context, slotID string, memoryLimit int64) error {
	if slotID == "" {
		return fmt.Errorf("slot_id is required")
	}
	if memoryLimit <= 0 {
		return fmt.Errorf("memory limit must be positive")
	}

	updateCmd := s.registry.GetCommand("resource.update_slot")
	if updateCmd == nil {
		return fmt.Errorf("resource.update_slot command not found")
	}

	_, err := updateCmd.Execute(ctx, map[string]any{
		"slot_id":      slotID,
		"memory_limit": memoryLimit,
	})
	if err != nil {
		return fmt.Errorf("update slot %s memory: %w", slotID, err)
	}

	return nil
}
