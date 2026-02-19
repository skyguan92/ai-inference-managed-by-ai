package inference

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type ModelsResource struct {
	provider InferenceProvider
}

func NewModelsResource(provider InferenceProvider) *ModelsResource {
	return &ModelsResource{provider: provider}
}

// InferenceResourceFactory creates inference Resource instances dynamically based on URI patterns.
type InferenceResourceFactory struct {
	provider InferenceProvider
}

func NewInferenceResourceFactory(provider InferenceProvider) *InferenceResourceFactory {
	return &InferenceResourceFactory{provider: provider}
}

func (f *InferenceResourceFactory) CanCreate(uri string) bool {
	return uri == "asms://inference/models"
}

func (f *InferenceResourceFactory) Create(uri string) (unit.Resource, error) {
	if uri == "asms://inference/models" {
		return NewModelsResource(f.provider), nil
	}
	return nil, fmt.Errorf("unknown inference resource URI: %s", uri)
}

func (f *InferenceResourceFactory) Pattern() string {
	return "asms://inference/models"
}

func (r *ModelsResource) URI() string {
	return "asms://inference/models"
}

func (r *ModelsResource) Domain() string {
	return "inference"
}

func (r *ModelsResource) Schema() unit.Schema {
	return unit.Schema{
		Type:        "object",
		Description: "Available inference models resource",
		Properties: map[string]unit.Field{
			"models": {
				Name: "models",
				Schema: unit.Schema{
					Type: "array",
					Items: &unit.Schema{
						Type: "object",
						Properties: map[string]unit.Field{
							"id":          {Name: "id", Schema: unit.Schema{Type: "string"}},
							"name":        {Name: "name", Schema: unit.Schema{Type: "string"}},
							"type":        {Name: "type", Schema: unit.Schema{Type: "string"}},
							"provider":    {Name: "provider", Schema: unit.Schema{Type: "string"}},
							"description": {Name: "description", Schema: unit.Schema{Type: "string"}},
							"max_tokens":  {Name: "max_tokens", Schema: unit.Schema{Type: "number"}},
							"modalities": {
								Name: "modalities",
								Schema: unit.Schema{
									Type:  "array",
									Items: &unit.Schema{Type: "string"},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *ModelsResource) Get(ctx context.Context) (any, error) {
	if r.provider == nil {
		return nil, ErrProviderNotSet
	}

	models, err := r.provider.ListModels(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("get models: %w", err)
	}

	items := make([]map[string]any, len(models))
	for i, m := range models {
		items[i] = map[string]any{
			"id":          m.ID,
			"name":        m.Name,
			"type":        m.Type,
			"provider":    m.Provider,
			"description": m.Description,
			"max_tokens":  m.MaxTokens,
			"modalities":  m.Modalities,
		}
	}

	return map[string]any{"models": items}, nil
}

func (r *ModelsResource) Watch(ctx context.Context) (<-chan unit.ResourceUpdate, error) {
	ch := make(chan unit.ResourceUpdate, 10)

	go func() {
		defer close(ch)
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		var lastCount int
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
					models, _ := dataMap["models"].([]map[string]any)
					newCount := len(models)

					if lastCount > 0 && newCount != lastCount {
						ch <- unit.ResourceUpdate{
							URI:       r.URI(),
							Timestamp: time.Now(),
							Operation: "models_changed",
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
					lastCount = newCount
				}
				mu.Unlock()
			}
		}
	}()

	return ch, nil
}

var _ unit.Resource = (*ModelsResource)(nil)
