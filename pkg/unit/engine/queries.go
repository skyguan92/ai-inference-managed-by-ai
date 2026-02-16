package engine

import (
	"context"
	"fmt"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type GetQuery struct {
	store EngineStore
}

func NewGetQuery(store EngineStore) *GetQuery {
	return &GetQuery{store: store}
}

func (q *GetQuery) Name() string {
	return "engine.get"
}

func (q *GetQuery) Domain() string {
	return "engine"
}

func (q *GetQuery) Description() string {
	return "Get detailed information about an engine"
}

func (q *GetQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"name": {
				Name: "name",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Engine name",
				},
			},
		},
		Required: []string{"name"},
	}
}

func (q *GetQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"name":         {Name: "name", Schema: unit.Schema{Type: "string"}},
			"type":         {Name: "type", Schema: unit.Schema{Type: "string"}},
			"status":       {Name: "status", Schema: unit.Schema{Type: "string"}},
			"version":      {Name: "version", Schema: unit.Schema{Type: "string"}},
			"capabilities": {Name: "capabilities", Schema: unit.Schema{Type: "array", Items: &unit.Schema{Type: "string"}}},
			"models":       {Name: "models", Schema: unit.Schema{Type: "array", Items: &unit.Schema{Type: "string"}}},
			"process_id":   {Name: "process_id", Schema: unit.Schema{Type: "string"}},
			"path":         {Name: "path", Schema: unit.Schema{Type: "string"}},
		},
	}
}

func (q *GetQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"name": "ollama"},
			Output:      map[string]any{"name": "ollama", "type": "ollama", "status": "running", "version": "0.1.26", "capabilities": []string{"chat", "completion"}, "models": []string{"llama3", "mistral"}},
			Description: "Get Ollama engine details",
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

	name, _ := inputMap["name"].(string)
	if name == "" {
		return nil, ErrInvalidEngineName
	}

	engine, err := q.store.Get(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("get engine %s: %w", name, err)
	}

	return map[string]any{
		"name":         engine.Name,
		"type":         string(engine.Type),
		"status":       string(engine.Status),
		"version":      engine.Version,
		"capabilities": engine.Capabilities,
		"models":       engine.Models,
		"process_id":   engine.ProcessID,
		"path":         engine.Path,
	}, nil
}

type ListQuery struct {
	store EngineStore
}

func NewListQuery(store EngineStore) *ListQuery {
	return &ListQuery{store: store}
}

func (q *ListQuery) Name() string {
	return "engine.list"
}

func (q *ListQuery) Domain() string {
	return "engine"
}

func (q *ListQuery) Description() string {
	return "List all engines with optional filtering"
}

func (q *ListQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"type": {
				Name: "type",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Filter by engine type",
					Enum: []any{
						string(EngineTypeOllama),
						string(EngineTypeVLLM),
						string(EngineTypeSGLang),
						string(EngineTypeWhisper),
						string(EngineTypeTTS),
						string(EngineTypeDiffusion),
						string(EngineTypeTransformers),
						string(EngineTypeHuggingFace),
						string(EngineTypeVideo),
						string(EngineTypeRerank),
					},
				},
			},
			"status": {
				Name: "status",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Filter by status",
					Enum:        []any{string(EngineStatusStopped), string(EngineStatusStarting), string(EngineStatusRunning), string(EngineStatusStopping), string(EngineStatusError), string(EngineStatusInstalling)},
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
			Output:      map[string]any{"items": []map[string]any{{"name": "ollama", "type": "ollama", "status": "running"}}, "total": 1},
			Description: "List all engines",
		},
		{
			Input:       map[string]any{"status": "running"},
			Output:      map[string]any{"items": []map[string]any{{"name": "ollama", "type": "ollama", "status": "running"}}, "total": 1},
			Description: "List running engines",
		},
	}
}

func (q *ListQuery) Execute(ctx context.Context, input any) (any, error) {
	if q.store == nil {
		return nil, ErrProviderNotSet
	}

	inputMap, _ := input.(map[string]any)

	filter := EngineFilter{
		Limit:  100,
		Offset: 0,
	}

	if t, ok := inputMap["type"].(string); ok && t != "" {
		filter.Type = EngineType(t)
	}
	if s, ok := inputMap["status"].(string); ok && s != "" {
		filter.Status = EngineStatus(s)
	}
	if limit, ok := toInt(inputMap["limit"]); ok && limit > 0 {
		filter.Limit = limit
	}
	if offset, ok := toInt(inputMap["offset"]); ok && offset >= 0 {
		filter.Offset = offset
	}

	engines, total, err := q.store.List(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("list engines: %w", err)
	}

	items := make([]map[string]any, len(engines))
	for i, e := range engines {
		items[i] = map[string]any{
			"name":   e.Name,
			"type":   string(e.Type),
			"status": string(e.Status),
		}
	}

	return map[string]any{
		"items": items,
		"total": total,
	}, nil
}

type FeaturesQuery struct {
	store    EngineStore
	provider EngineProvider
}

func NewFeaturesQuery(store EngineStore, provider EngineProvider) *FeaturesQuery {
	return &FeaturesQuery{store: store, provider: provider}
}

func (q *FeaturesQuery) Name() string {
	return "engine.features"
}

func (q *FeaturesQuery) Domain() string {
	return "engine"
}

func (q *FeaturesQuery) Description() string {
	return "Get engine features and capabilities"
}

func (q *FeaturesQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"name": {
				Name: "name",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Engine name",
				},
			},
		},
		Required: []string{"name"},
	}
}

func (q *FeaturesQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"supports_streaming":    {Name: "supports_streaming", Schema: unit.Schema{Type: "boolean"}},
			"supports_batch":        {Name: "supports_batch", Schema: unit.Schema{Type: "boolean"}},
			"supports_multimodal":   {Name: "supports_multimodal", Schema: unit.Schema{Type: "boolean"}},
			"supports_tools":        {Name: "supports_tools", Schema: unit.Schema{Type: "boolean"}},
			"supports_embedding":    {Name: "supports_embedding", Schema: unit.Schema{Type: "boolean"}},
			"max_concurrent":        {Name: "max_concurrent", Schema: unit.Schema{Type: "number"}},
			"max_context_length":    {Name: "max_context_length", Schema: unit.Schema{Type: "number"}},
			"max_batch_size":        {Name: "max_batch_size", Schema: unit.Schema{Type: "number"}},
			"supports_gpu_layers":   {Name: "supports_gpu_layers", Schema: unit.Schema{Type: "boolean"}},
			"supports_quantization": {Name: "supports_quantization", Schema: unit.Schema{Type: "boolean"}},
		},
	}
}

func (q *FeaturesQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"name": "ollama"},
			Output:      map[string]any{"supports_streaming": true, "supports_batch": true, "max_concurrent": 10, "max_context_length": 8192},
			Description: "Get Ollama engine features",
		},
	}
}

func (q *FeaturesQuery) Execute(ctx context.Context, input any) (any, error) {
	if q.store == nil || q.provider == nil {
		return nil, ErrProviderNotSet
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type: %w", ErrInvalidInput)
	}

	name, _ := inputMap["name"].(string)
	if name == "" {
		return nil, ErrInvalidEngineName
	}

	_, err := q.store.Get(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("get engine %s: %w", name, err)
	}

	features, err := q.provider.GetFeatures(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("get features for engine %s: %w", name, err)
	}

	return map[string]any{
		"supports_streaming":    features.SupportsStreaming,
		"supports_batch":        features.SupportsBatch,
		"supports_multimodal":   features.SupportsMultimodal,
		"supports_tools":        features.SupportsTools,
		"supports_embedding":    features.SupportsEmbedding,
		"max_concurrent":        features.MaxConcurrent,
		"max_context_length":    features.MaxContextLength,
		"max_batch_size":        features.MaxBatchSize,
		"supports_gpu_layers":   features.SupportsGPULayers,
		"supports_quantization": features.SupportsQuantization,
	}, nil
}

func ptrFloat(v float64) *float64 {
	return &v
}
