package resource

import (
	"context"
	"fmt"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
\t"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/ptrs"
)

type StatusQuery struct {
	provider ResourceProvider
	store    ResourceStore
	events   unit.EventPublisher
}

func NewStatusQuery(provider ResourceProvider, store ResourceStore) *StatusQuery {
	return &StatusQuery{provider: provider, store: store}
}

func NewStatusQueryWithEvents(provider ResourceProvider, store ResourceStore, events unit.EventPublisher) *StatusQuery {
	return &StatusQuery{provider: provider, store: store, events: events}
}

func (q *StatusQuery) Name() string {
	return "resource.status"
}

func (q *StatusQuery) Domain() string {
	return "resource"
}

func (q *StatusQuery) Description() string {
	return "Get current resource status including memory, storage, and pressure level"
}

func (q *StatusQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type:       "object",
		Properties: map[string]unit.Field{},
	}
}

func (q *StatusQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"memory": {
				Name: "memory",
				Schema: unit.Schema{
					Type: "object",
					Properties: map[string]unit.Field{
						"total":     {Name: "total", Schema: unit.Schema{Type: "number"}},
						"used":      {Name: "used", Schema: unit.Schema{Type: "number"}},
						"available": {Name: "available", Schema: unit.Schema{Type: "number"}},
					},
				},
			},
			"storage": {
				Name: "storage",
				Schema: unit.Schema{
					Type: "object",
					Properties: map[string]unit.Field{
						"total":     {Name: "total", Schema: unit.Schema{Type: "number"}},
						"used":      {Name: "used", Schema: unit.Schema{Type: "number"}},
						"available": {Name: "available", Schema: unit.Schema{Type: "number"}},
					},
				},
			},
			"slots": {
				Name: "slots",
				Schema: unit.Schema{
					Type: "array",
					Items: &unit.Schema{
						Type: "object",
						Properties: map[string]unit.Field{
							"id":           {Name: "id", Schema: unit.Schema{Type: "string"}},
							"name":         {Name: "name", Schema: unit.Schema{Type: "string"}},
							"type":         {Name: "type", Schema: unit.Schema{Type: "string"}},
							"memory_limit": {Name: "memory_limit", Schema: unit.Schema{Type: "number"}},
							"status":       {Name: "status", Schema: unit.Schema{Type: "string"}},
						},
					},
				},
			},
			"pressure": {
				Name:   "pressure",
				Schema: unit.Schema{Type: "string", Enum: []any{string(PressureLevelLow), string(PressureLevelMedium), string(PressureLevelHigh), string(PressureLevelCritical)}},
			},
		},
	}
}

func (q *StatusQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input: map[string]any{},
			Output: map[string]any{
				"memory":   map[string]any{"total": 68719476736, "used": 34359738368, "available": 34359738368},
				"storage":  map[string]any{"total": 1099511627776, "used": 549755813888, "available": 549755813888},
				"slots":    []map[string]any{},
				"pressure": "low",
			},
			Description: "Get current resource status",
		},
	}
}

func (q *StatusQuery) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(q.events, q.Domain(), q.Name())
	ec.PublishStarted(input)

	if q.provider == nil {
		err := ErrProviderNotSet
		ec.PublishFailed(err)
		return nil, err
	}

	status, err := q.provider.GetStatus(ctx)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("get resource status: %w", err)
	}

	var slots []map[string]any
	if q.store != nil {
		storeSlots, _, err := q.store.ListSlots(ctx, SlotFilter{})
		if err != nil {
			ec.PublishFailed(err)
			return nil, fmt.Errorf("list slots: %w", err)
		}
		for _, s := range storeSlots {
			slots = append(slots, map[string]any{
				"id":           s.ID,
				"name":         s.Name,
				"type":         string(s.Type),
				"memory_limit": s.MemoryLimit,
				"gpu_fraction": s.GPUFraction,
				"priority":     s.Priority,
				"status":       string(s.Status),
			})
		}
	} else {
		for _, s := range status.Slots {
			slots = append(slots, map[string]any{
				"id":           s.ID,
				"name":         s.Name,
				"type":         string(s.Type),
				"memory_limit": s.MemoryLimit,
				"gpu_fraction": s.GPUFraction,
				"priority":     s.Priority,
				"status":       string(s.Status),
			})
		}
	}

	output := map[string]any{
		"memory": map[string]any{
			"total":     status.Memory.Total,
			"used":      status.Memory.Used,
			"available": status.Memory.Available,
		},
		"storage": map[string]any{
			"total":     status.Storage.Total,
			"used":      status.Storage.Used,
			"available": status.Storage.Available,
		},
		"slots":    slots,
		"pressure": string(status.Pressure),
	}
	ec.PublishCompleted(output)
	return output, nil
}

type BudgetQuery struct {
	provider ResourceProvider
	events   unit.EventPublisher
}

func NewBudgetQuery(provider ResourceProvider) *BudgetQuery {
	return &BudgetQuery{provider: provider}
}

func NewBudgetQueryWithEvents(provider ResourceProvider, events unit.EventPublisher) *BudgetQuery {
	return &BudgetQuery{provider: provider, events: events}
}

func (q *BudgetQuery) Name() string {
	return "resource.budget"
}

func (q *BudgetQuery) Domain() string {
	return "resource"
}

func (q *BudgetQuery) Description() string {
	return "Get resource budget information including total, reserved, and pool allocations"
}

func (q *BudgetQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type:       "object",
		Properties: map[string]unit.Field{},
	}
}

func (q *BudgetQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"total": {
				Name:   "total",
				Schema: unit.Schema{Type: "number", Description: "Total memory available"},
			},
			"reserved": {
				Name:   "reserved",
				Schema: unit.Schema{Type: "number", Description: "Memory reserved for system use"},
			},
			"pools": {
				Name: "pools",
				Schema: unit.Schema{
					Type:        "object",
					Description: "Resource pools",
				},
			},
		},
	}
}

func (q *BudgetQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input: map[string]any{},
			Output: map[string]any{
				"total":    68719476736,
				"reserved": 17179869184,
				"pools": map[string]any{
					"inference": map[string]any{"name": "inference", "total": 51539607552, "reserved": 8589934592, "available": 42949672960},
				},
			},
			Description: "Get resource budget",
		},
	}
}

func (q *BudgetQuery) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(q.events, q.Domain(), q.Name())
	ec.PublishStarted(input)

	if q.provider == nil {
		err := ErrProviderNotSet
		ec.PublishFailed(err)
		return nil, err
	}

	budget, err := q.provider.GetBudget(ctx)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("get resource budget: %w", err)
	}

	pools := make(map[string]any)
	for name, pool := range budget.Pools {
		pools[name] = map[string]any{
			"name":      pool.Name,
			"total":     pool.Total,
			"reserved":  pool.Reserved,
			"available": pool.Available,
		}
	}

	output := map[string]any{
		"total":    budget.Total,
		"reserved": budget.Reserved,
		"pools":    pools,
	}
	ec.PublishCompleted(output)
	return output, nil
}

type AllocationsQuery struct {
	store  ResourceStore
	events unit.EventPublisher
}

func NewAllocationsQuery(store ResourceStore) *AllocationsQuery {
	return &AllocationsQuery{store: store}
}

func NewAllocationsQueryWithEvents(store ResourceStore, events unit.EventPublisher) *AllocationsQuery {
	return &AllocationsQuery{store: store, events: events}
}

func (q *AllocationsQuery) Name() string {
	return "resource.allocations"
}

func (q *AllocationsQuery) Domain() string {
	return "resource"
}

func (q *AllocationsQuery) Description() string {
	return "List resource allocations, optionally filtered by slot_id or type"
}

func (q *AllocationsQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"slot_id": {
				Name: "slot_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Filter by slot ID",
				},
			},
			"type": {
				Name: "type",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Filter by slot type",
					Enum:        []any{string(SlotTypeInferenceNative), string(SlotTypeDockerContainer), string(SlotTypeSystemService)},
				},
			},
		},
		Optional: []string{"slot_id", "type"},
	}
}

func (q *AllocationsQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"allocations": {
				Name: "allocations",
				Schema: unit.Schema{
					Type: "array",
					Items: &unit.Schema{
						Type: "object",
						Properties: map[string]unit.Field{
							"slot_id":      {Name: "slot_id", Schema: unit.Schema{Type: "string"}},
							"name":         {Name: "name", Schema: unit.Schema{Type: "string"}},
							"type":         {Name: "type", Schema: unit.Schema{Type: "string"}},
							"memory_used":  {Name: "memory_used", Schema: unit.Schema{Type: "number"}},
							"gpu_fraction": {Name: "gpu_fraction", Schema: unit.Schema{Type: "number"}},
							"priority":     {Name: "priority", Schema: unit.Schema{Type: "number"}},
							"status":       {Name: "status", Schema: unit.Schema{Type: "string"}},
						},
					},
				},
			},
		},
	}
}

func (q *AllocationsQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input: map[string]any{},
			Output: map[string]any{
				"allocations": []map[string]any{
					{"slot_id": "slot-abc12345", "name": "llama-model", "type": "inference_native", "memory_used": 16000000000, "gpu_fraction": 1.0, "priority": 10, "status": "active"},
				},
			},
			Description: "List all resource allocations",
		},
	}
}

func (q *AllocationsQuery) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(q.events, q.Domain(), q.Name())
	ec.PublishStarted(input)

	if q.store == nil {
		err := ErrProviderNotSet
		ec.PublishFailed(err)
		return nil, err
	}

	inputMap, _ := input.(map[string]any)

	filter := SlotFilter{}
	if slotID, ok := inputMap["slot_id"].(string); ok && slotID != "" {
		filter.SlotID = slotID
	}
	if typeStr, ok := inputMap["type"].(string); ok && typeStr != "" {
		filter.Type = SlotType(typeStr)
	}

	slots, _, err := q.store.ListSlots(ctx, filter)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("list allocations: %w", err)
	}

	allocations := make([]map[string]any, len(slots))
	for i, s := range slots {
		allocations[i] = map[string]any{
			"slot_id":      s.ID,
			"name":         s.Name,
			"type":         string(s.Type),
			"memory_used":  s.MemoryLimit,
			"gpu_fraction": s.GPUFraction,
			"priority":     s.Priority,
			"status":       string(s.Status),
		}
	}

	output := map[string]any{"allocations": allocations}
	ec.PublishCompleted(output)
	return output, nil
}

type CanAllocateQuery struct {
	provider ResourceProvider
	events   unit.EventPublisher
}

func NewCanAllocateQuery(provider ResourceProvider) *CanAllocateQuery {
	return &CanAllocateQuery{provider: provider}
}

func NewCanAllocateQueryWithEvents(provider ResourceProvider, events unit.EventPublisher) *CanAllocateQuery {
	return &CanAllocateQuery{provider: provider, events: events}
}

func (q *CanAllocateQuery) Name() string {
	return "resource.can_allocate"
}

func (q *CanAllocateQuery) Domain() string {
	return "resource"
}

func (q *CanAllocateQuery) Description() string {
	return "Check if the specified resources can be allocated"
}

func (q *CanAllocateQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"memory_bytes": {
				Name: "memory_bytes",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Memory bytes to allocate",
					Min:         ptrs.Float64(0),
				},
			},
			"priority": {
				Name: "priority",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Priority level (optional)",
					Min:         ptrs.Float64(0),
				},
			},
		},
		Required: []string{"memory_bytes"},
	}
}

func (q *CanAllocateQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"can_allocate": {
				Name:   "can_allocate",
				Schema: unit.Schema{Type: "boolean"},
			},
			"reason": {
				Name:   "reason",
				Schema: unit.Schema{Type: "string"},
			},
		},
	}
}

func (q *CanAllocateQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"memory_bytes": 16000000000, "priority": 10},
			Output:      map[string]any{"can_allocate": true},
			Description: "Check if 16GB can be allocated with high priority",
		},
		{
			Input:       map[string]any{"memory_bytes": 128000000000},
			Output:      map[string]any{"can_allocate": false, "reason": "insufficient memory"},
			Description: "Check if 128GB can be allocated (fails)",
		},
	}
}

func (q *CanAllocateQuery) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(q.events, q.Domain(), q.Name())
	ec.PublishStarted(input)

	if q.provider == nil {
		err := ErrProviderNotSet
		ec.PublishFailed(err)
		return nil, err
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		err := fmt.Errorf("invalid input type: expected map[string]any")
		ec.PublishFailed(err)
		return nil, err
	}

	memoryBytes, ok := toUint64(inputMap["memory_bytes"])
	if !ok || memoryBytes == 0 {
		ec.PublishFailed(ErrInvalidMemoryValue)
		return nil, ErrInvalidMemoryValue
	}

	priority := 5
	if p, ok := toInt(inputMap["priority"]); ok {
		priority = p
	}

	result, err := q.provider.CanAllocate(ctx, memoryBytes, priority)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("check allocation: %w", err)
	}

	output := map[string]any{
		"can_allocate": result.CanAllocate,
	}
	if result.Reason != "" {
		output["reason"] = result.Reason
	}

	ec.PublishCompleted(output)
	return output, nil
}
