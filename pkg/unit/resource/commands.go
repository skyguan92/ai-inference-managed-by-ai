package resource

import (
	"context"
	"fmt"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type AllocateCommand struct {
	store    ResourceStore
	provider ResourceProvider
}

func NewAllocateCommand(store ResourceStore, provider ResourceProvider) *AllocateCommand {
	return &AllocateCommand{store: store, provider: provider}
}

func (c *AllocateCommand) Name() string {
	return "resource.allocate"
}

func (c *AllocateCommand) Domain() string {
	return "resource"
}

func (c *AllocateCommand) Description() string {
	return "Allocate a resource slot with specified memory and GPU resources"
}

func (c *AllocateCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"name": {
				Name: "name",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Name for the allocated slot",
				},
			},
			"type": {
				Name: "type",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Type of slot",
					Enum:        []any{string(SlotTypeInferenceNative), string(SlotTypeDockerContainer), string(SlotTypeSystemService)},
				},
			},
			"memory_bytes": {
				Name: "memory_bytes",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Memory to allocate in bytes",
					Min:         ptrFloat(0),
				},
			},
			"gpu_fraction": {
				Name: "gpu_fraction",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Fraction of GPU to allocate (0.0-1.0)",
					Min:         ptrFloat(0),
					Max:         ptrFloat(1),
				},
			},
			"priority": {
				Name: "priority",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Priority level (higher = more important)",
					Min:         ptrFloat(0),
					Max:         ptrFloat(100),
				},
			},
		},
		Required: []string{"name", "type", "memory_bytes"},
	}
}

func (c *AllocateCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"slot_id": {
				Name:   "slot_id",
				Schema: unit.Schema{Type: "string"},
			},
		},
	}
}

func (c *AllocateCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"name": "llama-model", "type": "inference_native", "memory_bytes": 16000000000, "gpu_fraction": 1.0, "priority": 10},
			Output:      map[string]any{"slot_id": "slot-abc12345"},
			Description: "Allocate resources for a model inference slot",
		},
	}
}

func (c *AllocateCommand) Execute(ctx context.Context, input any) (any, error) {
	if c.store == nil {
		return nil, ErrProviderNotSet
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type: expected map[string]any")
	}

	name, _ := inputMap["name"].(string)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	slotTypeStr, _ := inputMap["type"].(string)
	slotType := SlotType(slotTypeStr)
	if !IsValidSlotType(slotType) {
		return nil, ErrInvalidSlotType
	}

	memoryBytes, ok := toUint64(inputMap["memory_bytes"])
	if !ok || memoryBytes == 0 {
		return nil, ErrInvalidMemoryValue
	}

	priority := 5
	if p, ok := toInt(inputMap["priority"]); ok {
		priority = p
	}

	gpuFraction := 0.0
	if g, ok := toFloat64(inputMap["gpu_fraction"]); ok {
		gpuFraction = g
	}

	if c.provider != nil {
		canAlloc, err := c.provider.CanAllocate(ctx, memoryBytes, priority)
		if err != nil {
			return nil, fmt.Errorf("check allocation: %w", err)
		}
		if !canAlloc.CanAllocate {
			reason := canAlloc.Reason
			if reason == "" {
				reason = "insufficient resources"
			}
			return nil, fmt.Errorf("%s: %w", reason, ErrInsufficientMemory)
		}
	}

	now := time.Now().Unix()
	slot := &ResourceSlot{
		ID:          generateSlotID(),
		Name:        name,
		Type:        slotType,
		MemoryLimit: memoryBytes,
		GPUFraction: gpuFraction,
		Priority:    priority,
		Status:      SlotStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := c.store.CreateSlot(ctx, slot); err != nil {
		return nil, fmt.Errorf("create slot: %w", err)
	}

	return map[string]any{"slot_id": slot.ID}, nil
}

type ReleaseCommand struct {
	store ResourceStore
}

func NewReleaseCommand(store ResourceStore) *ReleaseCommand {
	return &ReleaseCommand{store: store}
}

func (c *ReleaseCommand) Name() string {
	return "resource.release"
}

func (c *ReleaseCommand) Domain() string {
	return "resource"
}

func (c *ReleaseCommand) Description() string {
	return "Release an allocated resource slot"
}

func (c *ReleaseCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"slot_id": {
				Name: "slot_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "ID of the slot to release",
				},
			},
		},
		Required: []string{"slot_id"},
	}
}

func (c *ReleaseCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"success": {
				Name:   "success",
				Schema: unit.Schema{Type: "boolean"},
			},
		},
	}
}

func (c *ReleaseCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"slot_id": "slot-abc12345"},
			Output:      map[string]any{"success": true},
			Description: "Release an allocated slot",
		},
	}
}

func (c *ReleaseCommand) Execute(ctx context.Context, input any) (any, error) {
	if c.store == nil {
		return nil, ErrProviderNotSet
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type: expected map[string]any")
	}

	slotID, _ := inputMap["slot_id"].(string)
	if slotID == "" {
		return nil, ErrInvalidSlotID
	}

	if err := c.store.DeleteSlot(ctx, slotID); err != nil {
		return nil, fmt.Errorf("release slot %s: %w", slotID, err)
	}

	return map[string]any{"success": true}, nil
}

type UpdateSlotCommand struct {
	store ResourceStore
}

func NewUpdateSlotCommand(store ResourceStore) *UpdateSlotCommand {
	return &UpdateSlotCommand{store: store}
}

func (c *UpdateSlotCommand) Name() string {
	return "resource.update_slot"
}

func (c *UpdateSlotCommand) Domain() string {
	return "resource"
}

func (c *UpdateSlotCommand) Description() string {
	return "Update an existing resource slot configuration"
}

func (c *UpdateSlotCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"slot_id": {
				Name: "slot_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "ID of the slot to update",
				},
			},
			"memory_limit": {
				Name: "memory_limit",
				Schema: unit.Schema{
					Type:        "number",
					Description: "New memory limit in bytes",
					Min:         ptrFloat(0),
				},
			},
			"status": {
				Name: "status",
				Schema: unit.Schema{
					Type:        "string",
					Description: "New status for the slot",
					Enum:        []any{string(SlotStatusActive), string(SlotStatusIdle), string(SlotStatusPreempted)},
				},
			},
		},
		Required: []string{"slot_id"},
	}
}

func (c *UpdateSlotCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"success": {
				Name:   "success",
				Schema: unit.Schema{Type: "boolean"},
			},
		},
	}
}

func (c *UpdateSlotCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"slot_id": "slot-abc12345", "memory_limit": 32000000000, "status": "active"},
			Output:      map[string]any{"success": true},
			Description: "Update slot memory limit and status",
		},
	}
}

func (c *UpdateSlotCommand) Execute(ctx context.Context, input any) (any, error) {
	if c.store == nil {
		return nil, ErrProviderNotSet
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type: expected map[string]any")
	}

	slotID, _ := inputMap["slot_id"].(string)
	if slotID == "" {
		return nil, ErrInvalidSlotID
	}

	slot, err := c.store.GetSlot(ctx, slotID)
	if err != nil {
		return nil, fmt.Errorf("get slot %s: %w", slotID, err)
	}

	if memoryLimit, ok := toUint64(inputMap["memory_limit"]); ok {
		slot.MemoryLimit = memoryLimit
	}

	if statusStr, ok := inputMap["status"].(string); ok {
		status := SlotStatus(statusStr)
		if !IsValidSlotStatus(status) {
			return nil, ErrInvalidSlotStatus
		}
		slot.Status = status
	}

	slot.UpdatedAt = time.Now().Unix()

	if err := c.store.UpdateSlot(ctx, slot); err != nil {
		return nil, fmt.Errorf("update slot %s: %w", slotID, err)
	}

	return map[string]any{"success": true}, nil
}

func ptrFloat(v float64) *float64 {
	return &v
}

func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	case uint64:
		return float64(val), true
	default:
		return 0, false
	}
}

func toInt(v any) (int, bool) {
	switch val := v.(type) {
	case int:
		return val, true
	case int32:
		return int(val), true
	case int64:
		return int(val), true
	case float64:
		return int(val), true
	default:
		return 0, false
	}
}

func toUint64(v any) (uint64, bool) {
	switch val := v.(type) {
	case uint64:
		return val, true
	case uint:
		return uint64(val), true
	case uint32:
		return uint64(val), true
	case int:
		if val >= 0 {
			return uint64(val), true
		}
		return 0, false
	case int64:
		if val >= 0 {
			return uint64(val), true
		}
		return 0, false
	case float64:
		if val >= 0 {
			return uint64(val), true
		}
		return 0, false
	default:
		return 0, false
	}
}
