package model

import (
	"context"
	"fmt"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type GetQuery struct {
	store  ModelStore
	events unit.EventPublisher
}

func NewGetQuery(store ModelStore) *GetQuery {
	return &GetQuery{store: store}
}

func NewGetQueryWithEvents(store ModelStore, events unit.EventPublisher) *GetQuery {
	return &GetQuery{store: store, events: events}
}

func (q *GetQuery) Name() string {
	return "model.get"
}

func (q *GetQuery) Domain() string {
	return "model"
}

func (q *GetQuery) Description() string {
	return "Get detailed information about a model"
}

func (q *GetQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"model_id": {
				Name: "model_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Model identifier",
				},
			},
		},
		Required: []string{"model_id"},
	}
}

func (q *GetQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"id":           {Name: "id", Schema: unit.Schema{Type: "string"}},
			"name":         {Name: "name", Schema: unit.Schema{Type: "string"}},
			"type":         {Name: "type", Schema: unit.Schema{Type: "string"}},
			"format":       {Name: "format", Schema: unit.Schema{Type: "string"}},
			"status":       {Name: "status", Schema: unit.Schema{Type: "string"}},
			"size":         {Name: "size", Schema: unit.Schema{Type: "number"}},
			"requirements": {Name: "requirements", Schema: unit.Schema{Type: "object"}},
		},
	}
}

func (q *GetQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"model_id": "model-abc123"},
			Output:      map[string]any{"id": "model-abc123", "name": "llama3", "type": "llm", "format": "gguf", "status": "ready", "size": 4500000000},
			Description: "Get model details",
		},
	}
}

func (q *GetQuery) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(q.events, q.Domain(), q.Name())
	ec.PublishStarted(input)

	if q.store == nil {
		err := ErrProviderNotSet
		ec.PublishFailed(err)
		return nil, err
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		err := fmt.Errorf("invalid input type: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	modelID, _ := inputMap["model_id"].(string)
	if modelID == "" {
		err := ErrInvalidModelID
		ec.PublishFailed(err)
		return nil, err
	}

	model, err := q.store.Get(ctx, modelID)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("get model %s: %w", modelID, err)
	}

	result := map[string]any{
		"id":     model.ID,
		"name":   model.Name,
		"type":   string(model.Type),
		"format": string(model.Format),
		"status": string(model.Status),
		"size":   model.Size,
	}

	if model.Requirements != nil {
		result["requirements"] = map[string]any{
			"memory_min":         model.Requirements.MemoryMin,
			"memory_recommended": model.Requirements.MemoryRecommended,
			"gpu_type":           model.Requirements.GPUType,
			"gpu_memory":         model.Requirements.GPUMemory,
		}
	}

	ec.PublishCompleted(result)
	return result, nil
}

type ListQuery struct {
	store  ModelStore
	events unit.EventPublisher
}

func NewListQuery(store ModelStore) *ListQuery {
	return &ListQuery{store: store}
}

func NewListQueryWithEvents(store ModelStore, events unit.EventPublisher) *ListQuery {
	return &ListQuery{store: store, events: events}
}

func (q *ListQuery) Name() string {
	return "model.list"
}

func (q *ListQuery) Domain() string {
	return "model"
}

func (q *ListQuery) Description() string {
	return "List all models with optional filtering"
}

func (q *ListQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"type": {
				Name: "type",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Filter by model type",
				},
			},
			"status": {
				Name: "status",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Filter by status",
				},
			},
			"format": {
				Name: "format",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Filter by format",
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
			"items": {
				Name: "items",
				Schema: unit.Schema{
					Type: "array",
					Items: &unit.Schema{
						Type: "object",
						Properties: map[string]unit.Field{
							"id":     {Name: "id", Schema: unit.Schema{Type: "string"}},
							"name":   {Name: "name", Schema: unit.Schema{Type: "string"}},
							"type":   {Name: "type", Schema: unit.Schema{Type: "string"}},
							"status": {Name: "status", Schema: unit.Schema{Type: "string"}},
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
			Output:      map[string]any{"items": []map[string]any{{"id": "model-abc123", "name": "llama3", "type": "llm", "status": "ready"}}, "total": 1},
			Description: "List all models",
		},
		{
			Input:       map[string]any{"type": "llm", "limit": 10},
			Output:      map[string]any{"items": []map[string]any{{"id": "model-abc123", "name": "llama3", "type": "llm", "status": "ready"}}, "total": 1},
			Description: "List LLM models with limit",
		},
	}
}

func (q *ListQuery) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(q.events, q.Domain(), q.Name())
	ec.PublishStarted(input)

	if q.store == nil {
		err := ErrProviderNotSet
		ec.PublishFailed(err)
		return nil, err
	}

	inputMap, _ := input.(map[string]any)

	filter := ModelFilter{
		Limit:  100,
		Offset: 0,
	}

	if t, ok := inputMap["type"].(string); ok && t != "" {
		filter.Type = ModelType(t)
	}
	if s, ok := inputMap["status"].(string); ok && s != "" {
		filter.Status = ModelStatus(s)
	}
	if f, ok := inputMap["format"].(string); ok && f != "" {
		filter.Format = ModelFormat(f)
	}
	if limit, ok := toInt(inputMap["limit"]); ok && limit > 0 {
		filter.Limit = limit
	}
	if offset, ok := toInt(inputMap["offset"]); ok && offset >= 0 {
		filter.Offset = offset
	}

	models, total, err := q.store.List(ctx, filter)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("list models: %w", err)
	}

	items := make([]map[string]any, len(models))
	for i, m := range models {
		items[i] = map[string]any{
			"id":     m.ID,
			"name":   m.Name,
			"type":   string(m.Type),
			"status": string(m.Status),
		}
	}

	output := map[string]any{
		"items": items,
		"total": total,
	}
	ec.PublishCompleted(output)
	return output, nil
}

type SearchQuery struct {
	provider ModelProvider
	events   unit.EventPublisher
}

func NewSearchQuery(provider ModelProvider) *SearchQuery {
	return &SearchQuery{provider: provider}
}

func NewSearchQueryWithEvents(provider ModelProvider, events unit.EventPublisher) *SearchQuery {
	return &SearchQuery{provider: provider, events: events}
}

func (q *SearchQuery) Name() string {
	return "model.search"
}

func (q *SearchQuery) Domain() string {
	return "model"
}

func (q *SearchQuery) Description() string {
	return "Search for models in remote sources"
}

func (q *SearchQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"query": {
				Name: "query",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Search query",
				},
			},
			"source": {
				Name: "source",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Source to search (ollama, huggingface, modelscope)",
				},
			},
			"type": {
				Name: "type",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Filter by model type",
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
		},
		Required: []string{"query"},
	}
}

func (q *SearchQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"results": {
				Name: "results",
				Schema: unit.Schema{
					Type: "array",
					Items: &unit.Schema{
						Type: "object",
						Properties: map[string]unit.Field{
							"id":          {Name: "id", Schema: unit.Schema{Type: "string"}},
							"name":        {Name: "name", Schema: unit.Schema{Type: "string"}},
							"type":        {Name: "type", Schema: unit.Schema{Type: "string"}},
							"source":      {Name: "source", Schema: unit.Schema{Type: "string"}},
							"description": {Name: "description", Schema: unit.Schema{Type: "string"}},
							"downloads":   {Name: "downloads", Schema: unit.Schema{Type: "number"}},
						},
					},
				},
			},
		},
	}
}

func (q *SearchQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"query": "llama", "source": "ollama"},
			Output:      map[string]any{"results": []map[string]any{{"id": "llama3", "name": "Llama 3", "type": "llm", "source": "ollama", "downloads": 1000000}}},
			Description: "Search for llama models on Ollama",
		},
	}
}

func (q *SearchQuery) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(q.events, q.Domain(), q.Name())
	ec.PublishStarted(input)

	if q.provider == nil {
		err := ErrProviderNotSet
		ec.PublishFailed(err)
		return nil, err
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		err := fmt.Errorf("invalid input type: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	query, _ := inputMap["query"].(string)
	if query == "" {
		err := fmt.Errorf("query is required: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	source, _ := inputMap["source"].(string)
	modelType := ModelType("")
	if t, ok := inputMap["type"].(string); ok && t != "" {
		modelType = ModelType(t)
	}

	limit := 20
	if l, ok := toInt(inputMap["limit"]); ok && l > 0 {
		limit = l
	}

	results, err := q.provider.Search(ctx, query, source, modelType, limit)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("search models: %w", err)
	}

	items := make([]map[string]any, len(results))
	for i, r := range results {
		items[i] = map[string]any{
			"id":          r.ID,
			"name":        r.Name,
			"type":        string(r.Type),
			"source":      r.Source,
			"description": r.Description,
			"downloads":   r.Downloads,
		}
	}

	output := map[string]any{"results": items}
	ec.PublishCompleted(output)
	return output, nil
}

type EstimateResourcesQuery struct {
	store    ModelStore
	provider ModelProvider
	events   unit.EventPublisher
}

func NewEstimateResourcesQuery(store ModelStore, provider ModelProvider) *EstimateResourcesQuery {
	return &EstimateResourcesQuery{store: store, provider: provider}
}

func NewEstimateResourcesQueryWithEvents(store ModelStore, provider ModelProvider, events unit.EventPublisher) *EstimateResourcesQuery {
	return &EstimateResourcesQuery{store: store, provider: provider, events: events}
}

func (q *EstimateResourcesQuery) Name() string {
	return "model.estimate_resources"
}

func (q *EstimateResourcesQuery) Domain() string {
	return "model"
}

func (q *EstimateResourcesQuery) Description() string {
	return "Estimate resource requirements for a model"
}

func (q *EstimateResourcesQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"model_id": {
				Name: "model_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Model identifier",
				},
			},
		},
		Required: []string{"model_id"},
	}
}

func (q *EstimateResourcesQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"memory_min": {
				Name: "memory_min",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Minimum memory required in bytes",
				},
			},
			"memory_recommended": {
				Name: "memory_recommended",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Recommended memory in bytes",
				},
			},
			"gpu_type": {
				Name: "gpu_type",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Recommended GPU type",
				},
			},
		},
	}
}

func (q *EstimateResourcesQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"model_id": "model-abc123"},
			Output:      map[string]any{"memory_min": 8000000000, "memory_recommended": 16000000000, "gpu_type": "NVIDIA RTX 4090"},
			Description: "Estimate resources for a model",
		},
	}
}

func (q *EstimateResourcesQuery) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(q.events, q.Domain(), q.Name())
	ec.PublishStarted(input)

	if q.store == nil || q.provider == nil {
		err := ErrProviderNotSet
		ec.PublishFailed(err)
		return nil, err
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		err := fmt.Errorf("invalid input type: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	modelID, _ := inputMap["model_id"].(string)
	if modelID == "" {
		err := ErrInvalidModelID
		ec.PublishFailed(err)
		return nil, err
	}

	model, err := q.store.Get(ctx, modelID)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("get model %s: %w", modelID, err)
	}

	if model.Requirements != nil {
		output := map[string]any{
			"memory_min":         model.Requirements.MemoryMin,
			"memory_recommended": model.Requirements.MemoryRecommended,
			"gpu_type":           model.Requirements.GPUType,
		}
		ec.PublishCompleted(output)
		return output, nil
	}

	req, err := q.provider.EstimateResources(ctx, modelID)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("estimate resources for model %s: %w", modelID, err)
	}

	output := map[string]any{
		"memory_min":         req.MemoryMin,
		"memory_recommended": req.MemoryRecommended,
		"gpu_type":           req.GPUType,
	}
	ec.PublishCompleted(output)
	return output, nil
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
	case float32:
		return int(val), true
	default:
		return 0, false
	}
}

func ptrFloat(v float64) *float64 {
	return &v
}
