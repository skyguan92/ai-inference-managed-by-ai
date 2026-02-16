package pipeline

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type PipelineResource struct {
	id    string
	store PipelineStore
}

func NewPipelineResource(id string, store PipelineStore) *PipelineResource {
	return &PipelineResource{
		id:    id,
		store: store,
	}
}

func (r *PipelineResource) URI() string {
	return fmt.Sprintf("asms://pipeline/%s", r.id)
}

func (r *PipelineResource) Domain() string {
	return "pipeline"
}

func (r *PipelineResource) Schema() unit.Schema {
	return unit.Schema{
		Type:        "object",
		Description: "Pipeline resource",
		Properties: map[string]unit.Field{
			"id":         {Name: "id", Schema: unit.Schema{Type: "string"}},
			"name":       {Name: "name", Schema: unit.Schema{Type: "string"}},
			"steps":      {Name: "steps", Schema: unit.Schema{Type: "array"}},
			"status":     {Name: "status", Schema: unit.Schema{Type: "string"}},
			"config":     {Name: "config", Schema: unit.Schema{Type: "object"}},
			"created_at": {Name: "created_at", Schema: unit.Schema{Type: "number"}},
			"updated_at": {Name: "updated_at", Schema: unit.Schema{Type: "number"}},
		},
	}
}

func (r *PipelineResource) Get(ctx context.Context) (any, error) {
	if r.store == nil {
		return nil, ErrStoreNotSet
	}

	pipeline, err := r.store.GetPipeline(ctx, r.id)
	if err != nil {
		return nil, fmt.Errorf("get pipeline %s: %w", r.id, err)
	}

	return map[string]any{
		"id":         pipeline.ID,
		"name":       pipeline.Name,
		"steps":      pipeline.Steps,
		"status":     string(pipeline.Status),
		"config":     pipeline.Config,
		"created_at": pipeline.CreatedAt,
		"updated_at": pipeline.UpdatedAt,
	}, nil
}

func (r *PipelineResource) Watch(ctx context.Context) (<-chan unit.ResourceUpdate, error) {
	ch := make(chan unit.ResourceUpdate, 10)

	go func() {
		defer close(ch)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		var lastStatus PipelineStatus
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
					newStatus := PipelineStatus("")
					if s, ok := dataMap["status"].(string); ok {
						newStatus = PipelineStatus(s)
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

type PipelinesResource struct {
	store PipelineStore
}

func NewPipelinesResource(store PipelineStore) *PipelinesResource {
	return &PipelinesResource{store: store}
}

func (r *PipelinesResource) URI() string {
	return "asms://pipelines"
}

func (r *PipelinesResource) Domain() string {
	return "pipeline"
}

func (r *PipelinesResource) Schema() unit.Schema {
	return unit.Schema{
		Type:        "object",
		Description: "Pipelines list resource",
		Properties: map[string]unit.Field{
			"pipelines": {
				Name: "pipelines",
				Schema: unit.Schema{
					Type: "array",
					Items: &unit.Schema{
						Type: "object",
						Properties: map[string]unit.Field{
							"id":     {Name: "id", Schema: unit.Schema{Type: "string"}},
							"name":   {Name: "name", Schema: unit.Schema{Type: "string"}},
							"status": {Name: "status", Schema: unit.Schema{Type: "string"}},
						},
					},
				},
			},
			"total": {Name: "total", Schema: unit.Schema{Type: "number"}},
		},
	}
}

func (r *PipelinesResource) Get(ctx context.Context) (any, error) {
	if r.store == nil {
		return nil, ErrStoreNotSet
	}

	pipelines, total, err := r.store.ListPipelines(ctx, PipelineFilter{Limit: 1000})
	if err != nil {
		return nil, fmt.Errorf("list pipelines: %w", err)
	}

	items := make([]map[string]any, len(pipelines))
	for i, p := range pipelines {
		items[i] = map[string]any{
			"id":     p.ID,
			"name":   p.Name,
			"status": string(p.Status),
		}
	}

	return map[string]any{
		"pipelines": items,
		"total":     total,
	}, nil
}

func (r *PipelinesResource) Watch(ctx context.Context) (<-chan unit.ResourceUpdate, error) {
	ch := make(chan unit.ResourceUpdate, 10)

	go func() {
		defer close(ch)
		ticker := time.NewTicker(60 * time.Second)
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

func ParsePipelineResourceURI(uri string) (id string, ok bool) {
	if !strings.HasPrefix(uri, "asms://pipeline/") {
		return "", false
	}

	id = strings.TrimPrefix(uri, "asms://pipeline/")
	if id == "" {
		return "", false
	}

	return id, true
}
