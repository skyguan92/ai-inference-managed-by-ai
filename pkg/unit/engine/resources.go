package engine

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type EngineResource struct {
	name  string
	store EngineStore
}

func NewEngineResource(name string, store EngineStore) *EngineResource {
	return &EngineResource{
		name:  name,
		store: store,
	}
}

func (r *EngineResource) URI() string {
	return fmt.Sprintf("asms://engine/%s", r.name)
}

func (r *EngineResource) Domain() string {
	return "engine"
}

func (r *EngineResource) Schema() unit.Schema {
	return unit.Schema{
		Type:        "object",
		Description: "Engine information resource",
		Properties: map[string]unit.Field{
			"id":           {Name: "id", Schema: unit.Schema{Type: "string"}},
			"name":         {Name: "name", Schema: unit.Schema{Type: "string"}},
			"type":         {Name: "type", Schema: unit.Schema{Type: "string"}},
			"status":       {Name: "status", Schema: unit.Schema{Type: "string"}},
			"version":      {Name: "version", Schema: unit.Schema{Type: "string"}},
			"path":         {Name: "path", Schema: unit.Schema{Type: "string"}},
			"process_id":   {Name: "process_id", Schema: unit.Schema{Type: "string"}},
			"models":       {Name: "models", Schema: unit.Schema{Type: "array", Items: &unit.Schema{Type: "string"}}},
			"capabilities": {Name: "capabilities", Schema: unit.Schema{Type: "array", Items: &unit.Schema{Type: "string"}}},
			"config":       {Name: "config", Schema: unit.Schema{Type: "object"}},
			"created_at":   {Name: "created_at", Schema: unit.Schema{Type: "number"}},
			"updated_at":   {Name: "updated_at", Schema: unit.Schema{Type: "number"}},
		},
	}
}

func (r *EngineResource) Get(ctx context.Context) (any, error) {
	if r.store == nil {
		return nil, ErrProviderNotSet
	}

	engine, err := r.store.Get(ctx, r.name)
	if err != nil {
		return nil, fmt.Errorf("get engine %s: %w", r.name, err)
	}

	return map[string]any{
		"id":           engine.ID,
		"name":         engine.Name,
		"type":         string(engine.Type),
		"status":       string(engine.Status),
		"version":      engine.Version,
		"path":         engine.Path,
		"process_id":   engine.ProcessID,
		"models":       engine.Models,
		"capabilities": engine.Capabilities,
		"config":       engine.Config,
		"created_at":   engine.CreatedAt,
		"updated_at":   engine.UpdatedAt,
	}, nil
}

func (r *EngineResource) Watch(ctx context.Context) (<-chan unit.ResourceUpdate, error) {
	ch := make(chan unit.ResourceUpdate, 10)

	go func() {
		defer close(ch)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		var lastStatus EngineStatus
		var mu sync.Mutex

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

				mu.Lock()
				dataMap, ok := data.(map[string]any)
				if ok {
					newStatus := EngineStatus("")
					if s, ok := dataMap["status"].(string); ok {
						newStatus = EngineStatus(s)
					}

					if lastStatus != "" && newStatus != lastStatus {
						ch <- unit.ResourceUpdate{
							URI:       r.URI(),
							Timestamp: time.Now(),
							Operation: "status_changed",
							Data:      data,
						}
					} else {
						ch <- unit.ResourceUpdate{
							URI:       r.URI(),
							Timestamp: time.Now(),
							Operation: "refresh",
							Data:      data,
						}
					}
					lastStatus = newStatus
				}
				mu.Unlock()
			}
		}
	}()

	return ch, nil
}

func ParseEngineResourceURI(uri string) (name string, ok bool) {
	if !strings.HasPrefix(uri, "asms://engine/") {
		return "", false
	}

	name = strings.TrimPrefix(uri, "asms://engine/")
	if name == "" {
		return "", false
	}

	return name, true
}
