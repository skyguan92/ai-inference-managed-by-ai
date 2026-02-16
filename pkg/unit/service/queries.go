package service

import (
	"context"
	"fmt"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type GetQuery struct {
	store    ServiceStore
	provider ServiceProvider
}

func NewGetQuery(store ServiceStore, provider ServiceProvider) *GetQuery {
	return &GetQuery{store: store, provider: provider}
}

func (q *GetQuery) Name() string {
	return "service.get"
}

func (q *GetQuery) Domain() string {
	return "service"
}

func (q *GetQuery) Description() string {
	return "Get detailed information about a service"
}

func (q *GetQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"service_id": {
				Name: "service_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Service ID",
				},
			},
		},
		Required: []string{"service_id"},
	}
}

func (q *GetQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"id":              {Name: "id", Schema: unit.Schema{Type: "string"}},
			"name":            {Name: "name", Schema: unit.Schema{Type: "string"}},
			"model_id":        {Name: "model_id", Schema: unit.Schema{Type: "string"}},
			"status":          {Name: "status", Schema: unit.Schema{Type: "string"}},
			"replicas":        {Name: "replicas", Schema: unit.Schema{Type: "number"}},
			"endpoints":       {Name: "endpoints", Schema: unit.Schema{Type: "array", Items: &unit.Schema{Type: "string"}}},
			"resource_class":  {Name: "resource_class", Schema: unit.Schema{Type: "string"}},
			"active_replicas": {Name: "active_replicas", Schema: unit.Schema{Type: "number"}},
			"metrics": {
				Name: "metrics",
				Schema: unit.Schema{
					Type: "object",
					Properties: map[string]unit.Field{
						"requests_per_second": {Name: "requests_per_second", Schema: unit.Schema{Type: "number"}},
						"latency_p50":         {Name: "latency_p50", Schema: unit.Schema{Type: "number"}},
						"latency_p99":         {Name: "latency_p99", Schema: unit.Schema{Type: "number"}},
						"total_requests":      {Name: "total_requests", Schema: unit.Schema{Type: "number"}},
						"error_rate":          {Name: "error_rate", Schema: unit.Schema{Type: "number"}},
					},
				},
			},
		},
	}
}

func (q *GetQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"service_id": "svc-abc123"},
			Output:      map[string]any{"id": "svc-abc123", "model_id": "llama3-70b", "status": "running", "replicas": 2, "endpoints": []string{"http://localhost:8080"}},
			Description: "Get service details",
		},
	}
}

func (q *GetQuery) Execute(ctx context.Context, input any) (any, error) {
	if q.store == nil {
		return nil, ErrProviderNotSet
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type: %w", ErrInvalidInput)
	}

	serviceID, _ := inputMap["service_id"].(string)
	if serviceID == "" {
		return nil, fmt.Errorf("service_id is required: %w", ErrInvalidInput)
	}

	service, err := q.store.Get(ctx, serviceID)
	if err != nil {
		return nil, fmt.Errorf("get service %s: %w", serviceID, err)
	}

	result := map[string]any{
		"id":              service.ID,
		"name":            service.Name,
		"model_id":        service.ModelID,
		"status":          string(service.Status),
		"replicas":        service.Replicas,
		"endpoints":       service.Endpoints,
		"resource_class":  string(service.ResourceClass),
		"active_replicas": service.ActiveReplicas,
	}

	if q.provider != nil && service.Status == ServiceStatusRunning {
		metrics, err := q.provider.GetMetrics(ctx, serviceID)
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

type ListQuery struct {
	store ServiceStore
}

func NewListQuery(store ServiceStore) *ListQuery {
	return &ListQuery{store: store}
}

func (q *ListQuery) Name() string {
	return "service.list"
}

func (q *ListQuery) Domain() string {
	return "service"
}

func (q *ListQuery) Description() string {
	return "List all services with optional filtering"
}

func (q *ListQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"status": {
				Name: "status",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Filter by status",
					Enum:        []any{string(ServiceStatusCreating), string(ServiceStatusRunning), string(ServiceStatusStopped), string(ServiceStatusFailed)},
				},
			},
			"model_id": {
				Name: "model_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Filter by model ID",
				},
			},
			"limit": {
				Name: "limit",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Maximum number of results",
					Min:         ptrFloat(1),
					Max:         ptrFloat(100),
				},
			},
			"offset": {
				Name: "offset",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Offset for pagination",
					Min:         ptrFloat(0),
				},
			},
		},
	}
}

func (q *ListQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
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
			"total": {
				Name:   "total",
				Schema: unit.Schema{Type: "number"},
			},
		},
	}
}

func (q *ListQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{},
			Output:      map[string]any{"services": []map[string]any{{"id": "svc-abc123", "model_id": "llama3-70b", "status": "running", "replicas": 2}}, "total": 1},
			Description: "List all services",
		},
		{
			Input:       map[string]any{"status": "running"},
			Output:      map[string]any{"services": []map[string]any{{"id": "svc-abc123", "model_id": "llama3-70b", "status": "running", "replicas": 2}}, "total": 1},
			Description: "List running services",
		},
	}
}

func (q *ListQuery) Execute(ctx context.Context, input any) (any, error) {
	if q.store == nil {
		return nil, ErrProviderNotSet
	}

	inputMap, _ := input.(map[string]any)

	filter := ServiceFilter{
		Limit:  100,
		Offset: 0,
	}

	if s, ok := inputMap["status"].(string); ok && s != "" {
		filter.Status = ServiceStatus(s)
	}
	if m, ok := inputMap["model_id"].(string); ok && m != "" {
		filter.ModelID = m
	}
	if limit, ok := toInt(inputMap["limit"]); ok && limit > 0 {
		filter.Limit = limit
	}
	if offset, ok := toInt(inputMap["offset"]); ok && offset >= 0 {
		filter.Offset = offset
	}

	services, total, err := q.store.List(ctx, filter)
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

type RecommendQuery struct {
	provider ServiceProvider
}

func NewRecommendQuery(provider ServiceProvider) *RecommendQuery {
	return &RecommendQuery{provider: provider}
}

func (q *RecommendQuery) Name() string {
	return "service.recommend"
}

func (q *RecommendQuery) Domain() string {
	return "service"
}

func (q *RecommendQuery) Description() string {
	return "Get recommended configuration for a model"
}

func (q *RecommendQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"model_id": {
				Name: "model_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Model ID to get recommendation for",
				},
			},
			"hint": {
				Name: "hint",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Optional hint for recommendation (e.g., 'high-throughput', 'cost-effective')",
				},
			},
		},
		Required: []string{"model_id"},
	}
}

func (q *RecommendQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"resource_class":      {Name: "resource_class", Schema: unit.Schema{Type: "string"}},
			"replicas":            {Name: "replicas", Schema: unit.Schema{Type: "number"}},
			"expected_throughput": {Name: "expected_throughput", Schema: unit.Schema{Type: "number"}},
		},
	}
}

func (q *RecommendQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"model_id": "llama3-70b"},
			Output:      map[string]any{"resource_class": "large", "replicas": 2, "expected_throughput": 100.0},
			Description: "Get recommendation for llama3-70b",
		},
		{
			Input:       map[string]any{"model_id": "mistral-7b", "hint": "high-throughput"},
			Output:      map[string]any{"resource_class": "medium", "replicas": 4, "expected_throughput": 200.0},
			Description: "Get high-throughput recommendation",
		},
	}
}

func (q *RecommendQuery) Execute(ctx context.Context, input any) (any, error) {
	if q.provider == nil {
		return nil, ErrProviderNotSet
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type: %w", ErrInvalidInput)
	}

	modelID, _ := inputMap["model_id"].(string)
	if modelID == "" {
		return nil, fmt.Errorf("model_id is required: %w", ErrInvalidInput)
	}

	hint, _ := inputMap["hint"].(string)

	rec, err := q.provider.GetRecommendation(ctx, modelID, hint)
	if err != nil {
		return nil, fmt.Errorf("get recommendation: %w", err)
	}

	return map[string]any{
		"resource_class":      string(rec.ResourceClass),
		"replicas":            rec.Replicas,
		"expected_throughput": rec.ExpectedThroughput,
	}, nil
}
