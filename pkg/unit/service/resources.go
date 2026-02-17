package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type ServiceResource struct {
	id       string
	store    ServiceStore
	provider ServiceProvider
}

func NewServiceResource(id string, store ServiceStore, provider ServiceProvider) *ServiceResource {
	return &ServiceResource{
		id:       id,
		store:    store,
		provider: provider,
	}
}

// ServiceResourceFactory creates service Resource instances dynamically based on URI patterns.
type ServiceResourceFactory struct {
	store    ServiceStore
	provider ServiceProvider
}

func NewServiceResourceFactory(store ServiceStore, provider ServiceProvider) *ServiceResourceFactory {
	return &ServiceResourceFactory{store: store, provider: provider}
}

func (f *ServiceResourceFactory) CanCreate(uri string) bool {
	return strings.HasPrefix(uri, "asms://service/") || uri == "asms://services"
}

func (f *ServiceResourceFactory) Create(uri string) (unit.Resource, error) {
	if uri == "asms://services" {
		return NewServicesResource(f.store), nil
	}
	serviceID := strings.TrimPrefix(uri, "asms://service/")
	if serviceID == "" {
		return nil, fmt.Errorf("invalid service URI: %s", uri)
	}
	return NewServiceResource(serviceID, f.store, f.provider), nil
}

func (f *ServiceResourceFactory) Pattern() string {
	return "asms://service/*"
}

func (r *ServiceResource) URI() string {
	return fmt.Sprintf("asms://service/%s", r.id)
}

func (r *ServiceResource) Domain() string {
	return "service"
}

func (r *ServiceResource) Schema() unit.Schema {
	return unit.Schema{
		Type:        "object",
		Description: "Service information resource",
		Properties: map[string]unit.Field{
			"id":              {Name: "id", Schema: unit.Schema{Type: "string"}},
			"name":            {Name: "name", Schema: unit.Schema{Type: "string"}},
			"model_id":        {Name: "model_id", Schema: unit.Schema{Type: "string"}},
			"status":          {Name: "status", Schema: unit.Schema{Type: "string"}},
			"replicas":        {Name: "replicas", Schema: unit.Schema{Type: "number"}},
			"active_replicas": {Name: "active_replicas", Schema: unit.Schema{Type: "number"}},
			"resource_class":  {Name: "resource_class", Schema: unit.Schema{Type: "string"}},
			"endpoints":       {Name: "endpoints", Schema: unit.Schema{Type: "array", Items: &unit.Schema{Type: "string"}}},
			"config":          {Name: "config", Schema: unit.Schema{Type: "object"}},
			"created_at":      {Name: "created_at", Schema: unit.Schema{Type: "number"}},
			"updated_at":      {Name: "updated_at", Schema: unit.Schema{Type: "number"}},
		},
	}
}

func (r *ServiceResource) Get(ctx context.Context) (any, error) {
	if r.store == nil {
		return nil, ErrProviderNotSet
	}

	service, err := r.store.Get(ctx, r.id)
	if err != nil {
		return nil, fmt.Errorf("get service %s: %w", r.id, err)
	}

	result := map[string]any{
		"id":              service.ID,
		"name":            service.Name,
		"model_id":        service.ModelID,
		"status":          string(service.Status),
		"replicas":        service.Replicas,
		"active_replicas": service.ActiveReplicas,
		"resource_class":  string(service.ResourceClass),
		"endpoints":       service.Endpoints,
		"config":          service.Config,
		"created_at":      service.CreatedAt,
		"updated_at":      service.UpdatedAt,
	}

	if r.provider != nil && service.Status == ServiceStatusRunning {
		metrics, err := r.provider.GetMetrics(ctx, r.id)
		if err == nil {
			result["metrics"] = map[string]any{
				"requests_per_second": metrics.RequestsPerSecond,
				"latency_p50":         metrics.LatencyP50,
				"latency_p99":         metrics.LatencyP99,
				"total_requests":      metrics.TotalRequests,
				"error_rate":          metrics.ErrorRate,
			}
		}
	}

	return result, nil
}

func (r *ServiceResource) Watch(ctx context.Context) (<-chan unit.ResourceUpdate, error) {
	ch := make(chan unit.ResourceUpdate, 10)

	go func() {
		defer close(ch)
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		var lastStatus ServiceStatus
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
					newStatus := ServiceStatus("")
					if s, ok := dataMap["status"].(string); ok {
						newStatus = ServiceStatus(s)
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

type ServicesResource struct {
	store ServiceStore
}

func NewServicesResource(store ServiceStore) *ServicesResource {
	return &ServicesResource{store: store}
}

func (r *ServicesResource) URI() string {
	return "asms://services"
}

func (r *ServicesResource) Domain() string {
	return "service"
}

func (r *ServicesResource) Schema() unit.Schema {
	return unit.Schema{
		Type:        "object",
		Description: "Services list resource",
		Properties: map[string]unit.Field{
			"services": {
				Name: "services",
				Schema: unit.Schema{
					Type: "array",
					Items: &unit.Schema{
						Type: "object",
						Properties: map[string]unit.Field{
							"id":        {Name: "id", Schema: unit.Schema{Type: "string"}},
							"model_id":  {Name: "model_id", Schema: unit.Schema{Type: "string"}},
							"status":    {Name: "status", Schema: unit.Schema{Type: "string"}},
							"replicas":  {Name: "replicas", Schema: unit.Schema{Type: "number"}},
							"endpoints": {Name: "endpoints", Schema: unit.Schema{Type: "array", Items: &unit.Schema{Type: "string"}}},
						},
					},
				},
			},
			"total": {Name: "total", Schema: unit.Schema{Type: "number"}},
		},
	}
}

func (r *ServicesResource) Get(ctx context.Context) (any, error) {
	if r.store == nil {
		return nil, ErrProviderNotSet
	}

	services, total, err := r.store.List(ctx, ServiceFilter{Limit: 1000})
	if err != nil {
		return nil, fmt.Errorf("list services: %w", err)
	}

	items := make([]map[string]any, len(services))
	for i, s := range services {
		items[i] = map[string]any{
			"id":        s.ID,
			"model_id":  s.ModelID,
			"status":    string(s.Status),
			"replicas":  s.Replicas,
			"endpoints": s.Endpoints,
		}
	}

	return map[string]any{
		"services": items,
		"total":    total,
	}, nil
}

func (r *ServicesResource) Watch(ctx context.Context) (<-chan unit.ResourceUpdate, error) {
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

func ParseServiceResourceURI(uri string) (id string, ok bool) {
	if !strings.HasPrefix(uri, "asms://service/") {
		return "", false
	}

	id = strings.TrimPrefix(uri, "asms://service/")
	if id == "" {
		return "", false
	}

	return id, true
}
