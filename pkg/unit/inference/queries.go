package inference

import (
	"context"
	"fmt"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

type ModelsQuery struct {
	provider InferenceProvider
	events   unit.EventPublisher
}

func NewModelsQuery(provider InferenceProvider) *ModelsQuery {
	return &ModelsQuery{provider: provider}
}

func NewModelsQueryWithEvents(provider InferenceProvider, events unit.EventPublisher) *ModelsQuery {
	return &ModelsQuery{provider: provider, events: events}
}

func (q *ModelsQuery) Name() string {
	return "inference.models"
}

func (q *ModelsQuery) Domain() string {
	return "inference"
}

func (q *ModelsQuery) Description() string {
	return "List available inference models"
}

func (q *ModelsQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"type": {
				Name: "type",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Filter by model type (llm, asr, tts, embedding, diffusion, video_gen, detection, rerank)",
					Enum:        []any{"llm", "asr", "tts", "embedding", "diffusion", "video_gen", "detection", "rerank"},
				},
			},
		},
	}
}

func (q *ModelsQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"models": {
				Name: "models",
				Schema: unit.Schema{
					Type:  "array",
					Items: &unit.Schema{Type: "object"},
				},
			},
		},
	}
}

func (q *ModelsQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{},
			Output:      map[string]any{"models": []map[string]any{{"id": "llama3", "name": "Llama 3", "type": "llm"}}},
			Description: "List all models",
		},
		{
			Input:       map[string]any{"type": "llm"},
			Output:      map[string]any{"models": []map[string]any{{"id": "llama3", "name": "Llama 3", "type": "llm"}}},
			Description: "List LLM models only",
		},
	}
}

func (q *ModelsQuery) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(q.events, q.Domain(), q.Name())
	ec.PublishStarted(input)

	if q.provider == nil {
		err := ErrProviderNotSet
		ec.PublishFailed(err)
		return nil, err
	}

	inputMap, _ := input.(map[string]any)
	modelType, _ := inputMap["type"].(string)

	models, err := q.provider.ListModels(ctx, modelType)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("list models: %w", err)
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

	output := map[string]any{"models": items}
	ec.PublishCompleted(output)
	return output, nil
}

type VoicesQuery struct {
	provider InferenceProvider
	events   unit.EventPublisher
}

func NewVoicesQuery(provider InferenceProvider) *VoicesQuery {
	return &VoicesQuery{provider: provider}
}

func NewVoicesQueryWithEvents(provider InferenceProvider, events unit.EventPublisher) *VoicesQuery {
	return &VoicesQuery{provider: provider, events: events}
}

func (q *VoicesQuery) Name() string {
	return "inference.voices"
}

func (q *VoicesQuery) Domain() string {
	return "inference"
}

func (q *VoicesQuery) Description() string {
	return "List available voices for text-to-speech"
}

func (q *VoicesQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"model": {
				Name: "model",
				Schema: unit.Schema{
					Type:        "string",
					Description: "TTS model to get voices for",
				},
			},
		},
	}
}

func (q *VoicesQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"voices": {
				Name: "voices",
				Schema: unit.Schema{
					Type:  "array",
					Items: &unit.Schema{Type: "object"},
				},
			},
		},
	}
}

func (q *VoicesQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{},
			Output:      map[string]any{"voices": []map[string]any{{"id": "alloy", "name": "Alloy", "language": "en"}}},
			Description: "List available voices",
		},
		{
			Input:       map[string]any{"model": "tts-1"},
			Output:      map[string]any{"voices": []map[string]any{{"id": "alloy", "name": "Alloy", "language": "en"}}},
			Description: "List voices for specific TTS model",
		},
	}
}

func (q *VoicesQuery) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(q.events, q.Domain(), q.Name())
	ec.PublishStarted(input)

	if q.provider == nil {
		err := ErrProviderNotSet
		ec.PublishFailed(err)
		return nil, err
	}

	inputMap, _ := input.(map[string]any)
	model, _ := inputMap["model"].(string)

	voices, err := q.provider.ListVoices(ctx, model)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("list voices: %w", err)
	}

	items := make([]map[string]any, len(voices))
	for i, v := range voices {
		items[i] = map[string]any{
			"id":          v.ID,
			"name":        v.Name,
			"language":    v.Language,
			"gender":      v.Gender,
			"description": v.Description,
		}
	}

	output := map[string]any{"voices": items}
	ec.PublishCompleted(output)
	return output, nil
}
