package app

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type AppResource struct {
	id       string
	store    AppStore
	provider AppProvider
}

func NewAppResource(id string, store AppStore, provider AppProvider) *AppResource {
	return &AppResource{
		id:       id,
		store:    store,
		provider: provider,
	}
}

// AppResourceFactory creates app Resource instances dynamically based on URI patterns.
type AppResourceFactory struct {
	store    AppStore
	provider AppProvider
}

func NewAppResourceFactory(store AppStore, provider AppProvider) *AppResourceFactory {
	return &AppResourceFactory{store: store, provider: provider}
}

func (f *AppResourceFactory) CanCreate(uri string) bool {
	return strings.HasPrefix(uri, "asms://app/") || uri == "asms://apps/templates"
}

func (f *AppResourceFactory) Create(uri string) (unit.Resource, error) {
	if uri == "asms://apps/templates" {
		return NewTemplatesResource(f.provider), nil
	}
	appID := strings.TrimPrefix(uri, "asms://app/")
	if appID == "" {
		return nil, fmt.Errorf("invalid app URI: %s", uri)
	}
	return NewAppResource(appID, f.store, f.provider), nil
}

func (f *AppResourceFactory) Pattern() string {
	return "asms://app/*"
}

func (r *AppResource) URI() string {
	return fmt.Sprintf("asms://app/%s", r.id)
}

func (r *AppResource) Domain() string {
	return "app"
}

func (r *AppResource) Schema() unit.Schema {
	return unit.Schema{
		Type:        "object",
		Description: "Application information resource",
		Properties: map[string]unit.Field{
			"id":         {Name: "id", Schema: unit.Schema{Type: "string"}},
			"name":       {Name: "name", Schema: unit.Schema{Type: "string"}},
			"template":   {Name: "template", Schema: unit.Schema{Type: "string"}},
			"status":     {Name: "status", Schema: unit.Schema{Type: "string"}},
			"ports":      {Name: "ports", Schema: unit.Schema{Type: "array", Items: &unit.Schema{Type: "number"}}},
			"volumes":    {Name: "volumes", Schema: unit.Schema{Type: "array", Items: &unit.Schema{Type: "string"}}},
			"config":     {Name: "config", Schema: unit.Schema{Type: "object"}},
			"metrics":    {Name: "metrics", Schema: unit.Schema{Type: "object"}},
			"created_at": {Name: "created_at", Schema: unit.Schema{Type: "number"}},
			"updated_at": {Name: "updated_at", Schema: unit.Schema{Type: "number"}},
		},
	}
}

func (r *AppResource) Get(ctx context.Context) (any, error) {
	if r.store == nil {
		return nil, ErrProviderNotSet
	}

	app, err := r.store.Get(ctx, r.id)
	if err != nil {
		return nil, fmt.Errorf("get app %s: %w", r.id, err)
	}

	result := map[string]any{
		"id":         app.ID,
		"name":       app.Name,
		"template":   app.Template,
		"status":     string(app.Status),
		"ports":      app.Ports,
		"volumes":    app.Volumes,
		"config":     app.Config,
		"created_at": app.CreatedAt,
		"updated_at": app.UpdatedAt,
	}

	if r.provider != nil && app.Status == AppStatusRunning {
		metrics, err := r.provider.GetMetrics(ctx, r.id)
		if err == nil {
			result["metrics"] = map[string]any{
				"cpu_usage":    metrics.CPUUsage,
				"memory_usage": metrics.MemoryUsage,
				"uptime":       metrics.Uptime,
			}
		}
	}

	return result, nil
}

func (r *AppResource) Watch(ctx context.Context) (<-chan unit.ResourceUpdate, error) {
	ch := make(chan unit.ResourceUpdate, 10)

	go func() {
		defer close(ch)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		var lastStatus AppStatus
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
					newStatus := AppStatus("")
					if s, ok := dataMap["status"].(string); ok {
						newStatus = AppStatus(s)
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

type TemplatesResource struct {
	provider AppProvider
}

func NewTemplatesResource(provider AppProvider) *TemplatesResource {
	return &TemplatesResource{provider: provider}
}

func (r *TemplatesResource) URI() string {
	return "asms://apps/templates"
}

func (r *TemplatesResource) Domain() string {
	return "app"
}

func (r *TemplatesResource) Schema() unit.Schema {
	return unit.Schema{
		Type:        "object",
		Description: "Application templates resource",
		Properties: map[string]unit.Field{
			"templates": {
				Name: "templates",
				Schema: unit.Schema{
					Type: "array",
					Items: &unit.Schema{
						Type: "object",
						Properties: map[string]unit.Field{
							"id":            {Name: "id", Schema: unit.Schema{Type: "string"}},
							"name":          {Name: "name", Schema: unit.Schema{Type: "string"}},
							"category":      {Name: "category", Schema: unit.Schema{Type: "string"}},
							"description":   {Name: "description", Schema: unit.Schema{Type: "string"}},
							"image":         {Name: "image", Schema: unit.Schema{Type: "string"}},
							"default_ports": {Name: "default_ports", Schema: unit.Schema{Type: "array", Items: &unit.Schema{Type: "number"}}},
						},
					},
				},
			},
		},
	}
}

func (r *TemplatesResource) Get(ctx context.Context) (any, error) {
	if r.provider == nil {
		return nil, ErrProviderNotSet
	}

	templates, err := r.provider.GetTemplates(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("get templates: %w", err)
	}

	items := make([]map[string]any, len(templates))
	for i, t := range templates {
		items[i] = map[string]any{
			"id":            t.ID,
			"name":          t.Name,
			"category":      string(t.Category),
			"description":   t.Description,
			"image":         t.Image,
			"default_ports": t.DefaultPorts,
		}
	}

	return map[string]any{"templates": items}, nil
}

func (r *TemplatesResource) Watch(ctx context.Context) (<-chan unit.ResourceUpdate, error) {
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

func ParseAppResourceURI(uri string) (id string, ok bool) {
	if !strings.HasPrefix(uri, "asms://app/") {
		return "", false
	}

	id = strings.TrimPrefix(uri, "asms://app/")
	if id == "" {
		return "", false
	}

	return id, true
}
