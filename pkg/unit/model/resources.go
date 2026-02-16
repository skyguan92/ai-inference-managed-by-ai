package model

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type ModelResource struct {
	modelID string
	store   ModelStore
}

func NewModelResource(modelID string, store ModelStore) *ModelResource {
	return &ModelResource{
		modelID: modelID,
		store:   store,
	}
}

func (r *ModelResource) URI() string {
	return fmt.Sprintf("asms://model/%s", r.modelID)
}

func (r *ModelResource) Domain() string {
	return "model"
}

func (r *ModelResource) Schema() unit.Schema {
	return unit.Schema{
		Type:        "object",
		Description: "Model information resource",
		Properties: map[string]unit.Field{
			"id":           {Name: "id", Schema: unit.Schema{Type: "string"}},
			"name":         {Name: "name", Schema: unit.Schema{Type: "string"}},
			"type":         {Name: "type", Schema: unit.Schema{Type: "string"}},
			"format":       {Name: "format", Schema: unit.Schema{Type: "string"}},
			"status":       {Name: "status", Schema: unit.Schema{Type: "string"}},
			"size":         {Name: "size", Schema: unit.Schema{Type: "number"}},
			"source":       {Name: "source", Schema: unit.Schema{Type: "string"}},
			"path":         {Name: "path", Schema: unit.Schema{Type: "string"}},
			"requirements": {Name: "requirements", Schema: unit.Schema{Type: "object"}},
			"tags":         {Name: "tags", Schema: unit.Schema{Type: "array", Items: &unit.Schema{Type: "string"}}},
			"created_at":   {Name: "created_at", Schema: unit.Schema{Type: "number"}},
			"updated_at":   {Name: "updated_at", Schema: unit.Schema{Type: "number"}},
		},
	}
}

func (r *ModelResource) Get(ctx context.Context) (any, error) {
	if r.store == nil {
		return nil, ErrProviderNotSet
	}

	model, err := r.store.Get(ctx, r.modelID)
	if err != nil {
		return nil, fmt.Errorf("get model %s: %w", r.modelID, err)
	}

	result := map[string]any{
		"id":         model.ID,
		"name":       model.Name,
		"type":       string(model.Type),
		"format":     string(model.Format),
		"status":     string(model.Status),
		"size":       model.Size,
		"source":     model.Source,
		"path":       model.Path,
		"tags":       model.Tags,
		"created_at": model.CreatedAt,
		"updated_at": model.UpdatedAt,
	}

	if model.Requirements != nil {
		result["requirements"] = map[string]any{
			"memory_min":         model.Requirements.MemoryMin,
			"memory_recommended": model.Requirements.MemoryRecommended,
			"gpu_type":           model.Requirements.GPUType,
			"gpu_memory":         model.Requirements.GPUMemory,
		}
	}

	return result, nil
}

func (r *ModelResource) Watch(ctx context.Context) (<-chan unit.ResourceUpdate, error) {
	ch := make(chan unit.ResourceUpdate, 10)

	go func() {
		defer close(ch)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		var lastStatus ModelStatus
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
					newStatus := ModelStatus("")
					if s, ok := dataMap["status"].(string); ok {
						newStatus = ModelStatus(s)
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

func ParseModelResourceURI(uri string) (modelID string, ok bool) {
	if !strings.HasPrefix(uri, "asms://model/") {
		return "", false
	}

	modelID = strings.TrimPrefix(uri, "asms://model/")
	if modelID == "" {
		return "", false
	}

	return modelID, true
}
