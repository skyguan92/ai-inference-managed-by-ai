package resource

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type StatusResource struct {
	provider ResourceProvider
	store    ResourceStore
	last     *ResourceStatus
	mu       sync.RWMutex
}

func NewStatusResource(provider ResourceProvider, store ResourceStore) *StatusResource {
	return &StatusResource{provider: provider, store: store}
}

// ResourceFactory creates resource Resource instances dynamically based on URI patterns.
type ResourceFactory struct {
	provider ResourceProvider
	store    ResourceStore
}

func NewResourceFactory(provider ResourceProvider, store ResourceStore) *ResourceFactory {
	return &ResourceFactory{provider: provider, store: store}
}

func (f *ResourceFactory) CanCreate(uri string) bool {
	return strings.HasPrefix(uri, "asms://resource/")
}

func (f *ResourceFactory) Create(uri string) (unit.Resource, error) {
	switch uri {
	case "asms://resource/status":
		return NewStatusResource(f.provider, f.store), nil
	case "asms://resource/budget":
		return NewBudgetResource(f.provider), nil
	case "asms://resource/allocations":
		return NewAllocationsResource(f.store), nil
	default:
		return nil, fmt.Errorf("unknown resource URI: %s", uri)
	}
}

func (f *ResourceFactory) Pattern() string {
	return "asms://resource/*"
}

func (r *StatusResource) URI() string {
	return "asms://resource/status"
}

func (r *StatusResource) Domain() string {
	return "resource"
}

func (r *StatusResource) Schema() unit.Schema {
	return unit.Schema{
		Type:        "object",
		Description: "Resource status including memory, storage, and pressure",
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
				Name:   "slots",
				Schema: unit.Schema{Type: "array"},
			},
			"pressure": {
				Name:   "pressure",
				Schema: unit.Schema{Type: "string"},
			},
		},
	}
}

func (r *StatusResource) Get(ctx context.Context) (any, error) {
	if r.provider == nil {
		return nil, ErrProviderNotSet
	}

	status, err := r.provider.GetStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("get resource status: %w", err)
	}

	r.mu.Lock()
	r.last = status
	r.mu.Unlock()

	var slots []map[string]any
	if r.store != nil {
		storeSlots, _, err := r.store.ListSlots(ctx, SlotFilter{})
		if err == nil {
			for _, s := range storeSlots {
				slots = append(slots, map[string]any{
					"id":           s.ID,
					"name":         s.Name,
					"type":         string(s.Type),
					"memory_limit": s.MemoryLimit,
					"status":       string(s.Status),
				})
			}
		}
	}

	return map[string]any{
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
	}, nil
}

func (r *StatusResource) Watch(ctx context.Context) (<-chan unit.ResourceUpdate, error) {
	ch := make(chan unit.ResourceUpdate, 10)

	go func() {
		defer close(ch)
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				data, err := r.Get(ctx)
				if err != nil {
					ch <- unit.ResourceUpdate{
						URI:       r.URI(),
						Timestamp: time.Now(),
						Operation: "error",
						Error:     err,
					}
					continue
				}

				ch <- unit.ResourceUpdate{
					URI:       r.URI(),
					Timestamp: time.Now(),
					Operation: "refresh",
					Data:      data,
				}
			}
		}
	}()

	return ch, nil
}

type BudgetResource struct {
	provider ResourceProvider
	last     *ResourceBudget
	mu       sync.RWMutex
}

func NewBudgetResource(provider ResourceProvider) *BudgetResource {
	return &BudgetResource{provider: provider}
}

func (r *BudgetResource) URI() string {
	return "asms://resource/budget"
}

func (r *BudgetResource) Domain() string {
	return "resource"
}

func (r *BudgetResource) Schema() unit.Schema {
	return unit.Schema{
		Type:        "object",
		Description: "Resource budget information",
		Properties: map[string]unit.Field{
			"total":    {Name: "total", Schema: unit.Schema{Type: "number"}},
			"reserved": {Name: "reserved", Schema: unit.Schema{Type: "number"}},
			"pools":    {Name: "pools", Schema: unit.Schema{Type: "object"}},
		},
	}
}

func (r *BudgetResource) Get(ctx context.Context) (any, error) {
	if r.provider == nil {
		return nil, ErrProviderNotSet
	}

	budget, err := r.provider.GetBudget(ctx)
	if err != nil {
		return nil, fmt.Errorf("get resource budget: %w", err)
	}

	r.mu.Lock()
	r.last = budget
	r.mu.Unlock()

	pools := make(map[string]any)
	for name, pool := range budget.Pools {
		pools[name] = map[string]any{
			"name":      pool.Name,
			"total":     pool.Total,
			"reserved":  pool.Reserved,
			"available": pool.Available,
		}
	}

	return map[string]any{
		"total":    budget.Total,
		"reserved": budget.Reserved,
		"pools":    pools,
	}, nil
}

func (r *BudgetResource) Watch(ctx context.Context) (<-chan unit.ResourceUpdate, error) {
	ch := make(chan unit.ResourceUpdate, 10)

	go func() {
		defer close(ch)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				data, err := r.Get(ctx)
				if err != nil {
					ch <- unit.ResourceUpdate{
						URI:       r.URI(),
						Timestamp: time.Now(),
						Operation: "error",
						Error:     err,
					}
					continue
				}

				ch <- unit.ResourceUpdate{
					URI:       r.URI(),
					Timestamp: time.Now(),
					Operation: "refresh",
					Data:      data,
				}
			}
		}
	}()

	return ch, nil
}

type AllocationsResource struct {
	store ResourceStore
}

func NewAllocationsResource(store ResourceStore) *AllocationsResource {
	return &AllocationsResource{store: store}
}

func (r *AllocationsResource) URI() string {
	return "asms://resource/allocations"
}

func (r *AllocationsResource) Domain() string {
	return "resource"
}

func (r *AllocationsResource) Schema() unit.Schema {
	return unit.Schema{
		Type:        "object",
		Description: "List of resource allocations",
		Properties: map[string]unit.Field{
			"allocations": {
				Name:   "allocations",
				Schema: unit.Schema{Type: "array"},
			},
		},
	}
}

func (r *AllocationsResource) Get(ctx context.Context) (any, error) {
	if r.store == nil {
		return nil, ErrProviderNotSet
	}

	slots, _, err := r.store.ListSlots(ctx, SlotFilter{})
	if err != nil {
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

	return map[string]any{"allocations": allocations}, nil
}

func (r *AllocationsResource) Watch(ctx context.Context) (<-chan unit.ResourceUpdate, error) {
	ch := make(chan unit.ResourceUpdate, 10)

	go func() {
		defer close(ch)
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				data, err := r.Get(ctx)
				if err != nil {
					ch <- unit.ResourceUpdate{
						URI:       r.URI(),
						Timestamp: time.Now(),
						Operation: "error",
						Error:     err,
					}
					continue
				}

				ch <- unit.ResourceUpdate{
					URI:       r.URI(),
					Timestamp: time.Now(),
					Operation: "refresh",
					Data:      data,
				}
			}
		}
	}()

	return ch, nil
}

func ParseResourceURI(uri string) (resourceType string, ok bool) {
	if !strings.HasPrefix(uri, "asms://resource/") {
		return "", false
	}

	parts := strings.Split(strings.TrimPrefix(uri, "asms://resource/"), "/")
	if len(parts) != 1 {
		return "", false
	}

	return parts[0], true
}
