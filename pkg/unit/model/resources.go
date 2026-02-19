package model

import (
	"context"
	"fmt"
	"strings"
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

// ModelResourceFactory creates ModelResource instances dynamically based on URI patterns.
type ModelResourceFactory struct {
	store ModelStore
}

func NewModelResourceFactory(store ModelStore) *ModelResourceFactory {
	return &ModelResourceFactory{store: store}
}

func (f *ModelResourceFactory) CanCreate(uri string) bool {
	return strings.HasPrefix(uri, "asms://model/")
}

func (f *ModelResourceFactory) Create(uri string) (unit.Resource, error) {
	modelID := strings.TrimPrefix(uri, "asms://model/")
	if modelID == "" {
		return nil, fmt.Errorf("invalid model URI: %s", uri)
	}
	return NewModelResource(modelID, f.store), nil
}

func (f *ModelResourceFactory) Pattern() string {
	return "asms://model/*"
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

// EngineCompatibility describes what model formats and types an engine supports.
type EngineCompatibility struct {
	Engine              string   `json:"engine"`
	SupportedTypes      []string `json:"supported_types"`
	SupportedFormats    []string `json:"supported_formats"`
	GPURequired         bool     `json:"gpu_required"`
	QuantizationSupport []string `json:"quantization_support"`
	Notes               string   `json:"notes,omitempty"`
}

// CompatibilityResource is a static resource that exposes the engine/model compatibility matrix.
// URI: asms://models/compatibility
type CompatibilityResource struct{}

func NewCompatibilityResource() *CompatibilityResource {
	return &CompatibilityResource{}
}

func (r *CompatibilityResource) URI() string {
	return "asms://models/compatibility"
}

func (r *CompatibilityResource) Domain() string {
	return "model"
}

func (r *CompatibilityResource) Schema() unit.Schema {
	return unit.Schema{
		Type:        "object",
		Description: "Engine/model format compatibility matrix with GPU requirements and quantization support",
		Properties: map[string]unit.Field{
			"engines": {
				Name: "engines",
				Schema: unit.Schema{
					Type:        "array",
					Description: "List of engine compatibility entries",
					Items: &unit.Schema{
						Type: "object",
						Properties: map[string]unit.Field{
							"engine":               {Name: "engine", Schema: unit.Schema{Type: "string"}},
							"supported_types":      {Name: "supported_types", Schema: unit.Schema{Type: "array", Items: &unit.Schema{Type: "string"}}},
							"supported_formats":    {Name: "supported_formats", Schema: unit.Schema{Type: "array", Items: &unit.Schema{Type: "string"}}},
							"gpu_required":         {Name: "gpu_required", Schema: unit.Schema{Type: "boolean"}},
							"quantization_support": {Name: "quantization_support", Schema: unit.Schema{Type: "array", Items: &unit.Schema{Type: "string"}}},
							"notes":                {Name: "notes", Schema: unit.Schema{Type: "string"}},
						},
					},
				},
			},
			"updated_at": {Name: "updated_at", Schema: unit.Schema{Type: "number", Description: "Unix timestamp of last update"}},
		},
	}
}

func (r *CompatibilityResource) Get(_ context.Context) (any, error) {
	engines := []EngineCompatibility{
		{
			Engine:              "vllm",
			SupportedTypes:      []string{"llm", "vlm"},
			SupportedFormats:    []string{"safetensors", "gguf"},
			GPURequired:         true,
			QuantizationSupport: []string{"awq", "gptq", "fp8", "int8", "int4"},
			Notes:               "Supports tensor parallelism and continuous batching. GGUF support via llama.cpp backend.",
		},
		{
			Engine:              "whisper",
			SupportedTypes:      []string{"asr"},
			SupportedFormats:    []string{"safetensors", "pytorch"},
			GPURequired:         false,
			QuantizationSupport: []string{"int8"},
			Notes:               "Supports CPU inference. GPU accelerates transcription significantly.",
		},
		{
			Engine:              "tts",
			SupportedTypes:      []string{"tts"},
			SupportedFormats:    []string{"safetensors", "pytorch", "onnx"},
			GPURequired:         false,
			QuantizationSupport: []string{},
			Notes:               "Text-to-speech engine. CPU inference supported; GPU improves throughput.",
		},
		{
			Engine:              "ollama",
			SupportedTypes:      []string{"llm", "vlm", "embedding"},
			SupportedFormats:    []string{"gguf"},
			GPURequired:         false,
			QuantizationSupport: []string{"q4_0", "q4_k_m", "q5_0", "q5_k_m", "q6_k", "q8_0", "f16"},
			Notes:               "Optimised for local inference. Supports CPU and GPU via Metal/CUDA/ROCm.",
		},
	}

	result := make([]map[string]any, 0, len(engines))
	for _, e := range engines {
		result = append(result, map[string]any{
			"engine":               e.Engine,
			"supported_types":      e.SupportedTypes,
			"supported_formats":    e.SupportedFormats,
			"gpu_required":         e.GPURequired,
			"quantization_support": e.QuantizationSupport,
			"notes":                e.Notes,
		})
	}

	return map[string]any{
		"engines":    result,
		"updated_at": time.Now().Unix(),
	}, nil
}

func (r *CompatibilityResource) Watch(ctx context.Context) (<-chan unit.ResourceUpdate, error) {
	ch := make(chan unit.ResourceUpdate, 1)
	// Compatibility matrix is static; signal done when context is cancelled.
	go func() {
		defer close(ch)
		<-ctx.Done()
	}()
	return ch, nil
}
