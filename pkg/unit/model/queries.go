package model

import (
	"context"
	"fmt"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type GetQuery struct {
	store ModelStore
}

func NewGetQuery(store ModelStore) *GetQuery {
	return &GetQuery{store: store}
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
	if q.store == nil {
		return nil, ErrProviderNotSet
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type: %w", ErrInvalidInput)
	}

	modelID, _ := inputMap["model_id"].(string)
	if modelID == "" {
		return nil, ErrInvalidModelID
	}

	model, err := q.store.Get(ctx, modelID)
	if err != nil {
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

	return result, nil
}

type ListQuery struct {
	store ModelStore
}

func NewListQuery(store ModelStore) *ListQuery {
	return &ListQuery{store: store}
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
	if q.store == nil {
		return nil, ErrProviderNotSet
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

	return map[string]any{
		"items": items,
		"total": total,
	}, nil
}

type SearchQuery struct {
	provider ModelProvider
}

func NewSearchQuery(provider ModelProvider) *SearchQuery {
	return &SearchQuery{provider: provider}
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
	if q.provider == nil {
		return nil, ErrProviderNotSet
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type: %w", ErrInvalidInput)
	}

	query, _ := inputMap["query"].(string)
	if query == "" {
		return nil, fmt.Errorf("query is required: %w", ErrInvalidInput)
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

	return map[string]any{"results": items}, nil
}

type EstimateResourcesQuery struct {
	store    ModelStore
	provider ModelProvider
}

func NewEstimateResourcesQuery(store ModelStore, provider ModelProvider) *EstimateResourcesQuery {
	return &EstimateResourcesQuery{store: store, provider: provider}
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
	if q.store == nil || q.provider == nil {
		return nil, ErrProviderNotSet
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type: %w", ErrInvalidInput)
	}

	modelID, _ := inputMap["model_id"].(string)
	if modelID == "" {
		return nil, ErrInvalidModelID
	}

	model, err := q.store.Get(ctx, modelID)
	if err != nil {
		return nil, fmt.Errorf("get model %s: %w", modelID, err)
	}

	if model.Requirements != nil {
		return map[string]any{
			"memory_min":         model.Requirements.MemoryMin,
			"memory_recommended": model.Requirements.MemoryRecommended,
			"gpu_type":           model.Requirements.GPUType,
		}, nil
	}

	req, err := q.provider.EstimateResources(ctx, modelID)
	if err != nil {
		return nil, fmt.Errorf("estimate resources for model %s: %w", modelID, err)
	}

	return map[string]any{
		"memory_min":         req.MemoryMin,
		"memory_recommended": req.MemoryRecommended,
		"gpu_type":           req.GPUType,
	}, nil
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
