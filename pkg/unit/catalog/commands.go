package catalog

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

// EventPublisher interface for publishing events.
type EventPublisher interface {
	Publish(event any) error
}

// CreateRecipeCommand creates a new recipe in the store.
type CreateRecipeCommand struct {
	store  RecipeStore
	events EventPublisher
}

func NewCreateRecipeCommand(store RecipeStore) *CreateRecipeCommand {
	return &CreateRecipeCommand{store: store}
}

func NewCreateRecipeCommandWithEvents(store RecipeStore, events EventPublisher) *CreateRecipeCommand {
	return &CreateRecipeCommand{store: store, events: events}
}

func (c *CreateRecipeCommand) Name() string        { return "catalog.create_recipe" }
func (c *CreateRecipeCommand) Domain() string      { return "catalog" }
func (c *CreateRecipeCommand) Description() string { return "Create or add a recipe to the catalog" }

func (c *CreateRecipeCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"id": {
				Name: "id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Recipe identifier (auto-generated if not provided)",
				},
			},
			"name": {
				Name: "name",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Recipe name",
				},
			},
			"description": {
				Name: "description",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Recipe description",
				},
			},
			"version": {
				Name: "version",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Recipe version",
				},
			},
			"profile": {
				Name: "profile",
				Schema: unit.Schema{
					Type:        "object",
					Description: "Hardware profile this recipe targets",
				},
			},
			"engine": {
				Name: "engine",
				Schema: unit.Schema{
					Type:        "object",
					Description: "Inference engine configuration",
				},
			},
			"models": {
				Name: "models",
				Schema: unit.Schema{
					Type:        "array",
					Description: "List of models in this recipe",
				},
			},
		},
		Required: []string{"name", "profile", "engine"},
	}
}

func (c *CreateRecipeCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"recipe_id": {
				Name:   "recipe_id",
				Schema: unit.Schema{Type: "string"},
			},
		},
	}
}

func (c *CreateRecipeCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input: map[string]any{
				"name":    "NVIDIA RTX 4090 LLM",
				"profile": map[string]any{"gpu_vendor": "NVIDIA", "gpu_model": "RTX 4090", "vram_min_gb": 24, "os": "linux"},
				"engine":  map[string]any{"type": "vllm", "image": "vllm/vllm-openai:latest"},
			},
			Output:      map[string]any{"recipe_id": "recipe-abc123"},
			Description: "Create a recipe for NVIDIA RTX 4090",
		},
	}
}

func (c *CreateRecipeCommand) Execute(ctx context.Context, input any) (any, error) {
	if c.store == nil {
		return nil, ErrProviderNotSet
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type: %w", ErrInvalidInput)
	}

	name, _ := inputMap["name"].(string)
	if name == "" {
		return nil, fmt.Errorf("name is required: %w", ErrInvalidInput)
	}

	recipe := &Recipe{
		ID:      generateRecipeID(),
		Name:    name,
		Version: "1.0.0",
	}

	if id, ok := inputMap["id"].(string); ok && id != "" {
		recipe.ID = id
	}
	if desc, ok := inputMap["description"].(string); ok {
		recipe.Description = desc
	}
	if ver, ok := inputMap["version"].(string); ok && ver != "" {
		recipe.Version = ver
	}
	if author, ok := inputMap["author"].(string); ok {
		recipe.Author = author
	}
	if verified, ok := inputMap["verified"].(bool); ok {
		recipe.Verified = verified
	}

	if profileMap, ok := inputMap["profile"].(map[string]any); ok {
		recipe.Profile = parseHardwareProfile(profileMap)
	}

	if engineMap, ok := inputMap["engine"].(map[string]any); ok {
		recipe.Engine = parseRecipeEngine(engineMap)
	}

	if modelsRaw, ok := inputMap["models"].([]any); ok {
		for _, m := range modelsRaw {
			if mMap, ok := m.(map[string]any); ok {
				recipe.Models = append(recipe.Models, parseRecipeModel(mMap))
			}
		}
	}

	if tagsRaw, ok := inputMap["tags"].([]any); ok {
		for _, t := range tagsRaw {
			if tag, ok := t.(string); ok {
				recipe.Tags = append(recipe.Tags, tag)
			}
		}
	}

	if err := c.store.Create(ctx, recipe); err != nil {
		return nil, fmt.Errorf("create recipe: %w", err)
	}

	if c.events != nil {
		if err := c.events.Publish(NewRecipeCreatedEvent(recipe)); err != nil {
			slog.Warn("failed to publish catalog.recipe_created event", "error", err)
		}
	}

	return map[string]any{"recipe_id": recipe.ID}, nil
}

// ValidateRecipeCommand validates a recipe's format and required fields.
type ValidateRecipeCommand struct{}

func NewValidateRecipeCommand() *ValidateRecipeCommand {
	return &ValidateRecipeCommand{}
}

func (c *ValidateRecipeCommand) Name() string        { return "catalog.validate_recipe" }
func (c *ValidateRecipeCommand) Domain() string      { return "catalog" }
func (c *ValidateRecipeCommand) Description() string { return "Validate recipe format and required fields" }

func (c *ValidateRecipeCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"recipe": {
				Name: "recipe",
				Schema: unit.Schema{
					Type:        "object",
					Description: "Recipe object to validate",
				},
			},
		},
		Required: []string{"recipe"},
	}
}

func (c *ValidateRecipeCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"valid":  {Name: "valid", Schema: unit.Schema{Type: "boolean"}},
			"issues": {Name: "issues", Schema: unit.Schema{Type: "array", Items: &unit.Schema{Type: "string"}}},
		},
	}
}

func (c *ValidateRecipeCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input: map[string]any{
				"recipe": map[string]any{
					"name": "Test Recipe", "profile": map[string]any{"gpu_vendor": "NVIDIA"}, "engine": map[string]any{"type": "vllm", "image": "vllm/vllm-openai:latest"},
				},
			},
			Output:      map[string]any{"valid": true, "issues": []string{}},
			Description: "Validate a recipe",
		},
	}
}

func (c *ValidateRecipeCommand) Execute(ctx context.Context, input any) (any, error) {
	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type: %w", ErrInvalidInput)
	}

	recipeMap, ok := inputMap["recipe"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("recipe is required: %w", ErrInvalidInput)
	}

	var issues []string

	if name, _ := recipeMap["name"].(string); name == "" {
		issues = append(issues, "name is required")
	}

	if engineMap, ok := recipeMap["engine"].(map[string]any); !ok {
		issues = append(issues, "engine is required")
	} else {
		if t, _ := engineMap["type"].(string); t == "" {
			issues = append(issues, "engine.type is required")
		}
		if img, _ := engineMap["image"].(string); img == "" {
			issues = append(issues, "engine.image is required")
		}
	}

	if _, ok := recipeMap["profile"].(map[string]any); !ok {
		issues = append(issues, "profile is required")
	}

	return map[string]any{
		"valid":  len(issues) == 0,
		"issues": issues,
	}, nil
}

// ApplyRecipeCommand triggers deployment of a recipe's engine and models.
type ApplyRecipeCommand struct {
	store  RecipeStore
	events EventPublisher
}

func NewApplyRecipeCommand(store RecipeStore) *ApplyRecipeCommand {
	return &ApplyRecipeCommand{store: store}
}

func NewApplyRecipeCommandWithEvents(store RecipeStore, events EventPublisher) *ApplyRecipeCommand {
	return &ApplyRecipeCommand{store: store, events: events}
}

func (c *ApplyRecipeCommand) Name() string        { return "catalog.apply_recipe" }
func (c *ApplyRecipeCommand) Domain() string      { return "catalog" }
func (c *ApplyRecipeCommand) Description() string { return "Deploy a recipe: pull engine image and models" }

func (c *ApplyRecipeCommand) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"recipe_id": {
				Name: "recipe_id",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Recipe identifier",
				},
			},
			"skip_engine": {
				Name: "skip_engine",
				Schema: unit.Schema{
					Type:        "boolean",
					Description: "Skip engine image pull",
				},
			},
			"skip_models": {
				Name: "skip_models",
				Schema: unit.Schema{
					Type:        "boolean",
					Description: "Skip model downloads",
				},
			},
		},
		Required: []string{"recipe_id"},
	}
}

func (c *ApplyRecipeCommand) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"engine_ready": {Name: "engine_ready", Schema: unit.Schema{Type: "boolean"}},
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

func (c *ApplyRecipeCommand) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"recipe_id": "recipe-abc123"},
			Output:      map[string]any{"engine_ready": true, "models": []map[string]any{{"name": "llama3", "status": "ready"}}},
			Description: "Apply a recipe to deploy engine and models",
		},
	}
}

func (c *ApplyRecipeCommand) Execute(ctx context.Context, input any) (any, error) {
	if c.store == nil {
		return nil, ErrProviderNotSet
	}

	inputMap, ok := input.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid input type: %w", ErrInvalidInput)
	}

	recipeID, _ := inputMap["recipe_id"].(string)
	if recipeID == "" {
		return nil, fmt.Errorf("recipe_id is required: %w", ErrInvalidInput)
	}

	recipe, err := c.store.Get(ctx, recipeID)
	if err != nil {
		return nil, fmt.Errorf("get recipe %s: %w", recipeID, err)
	}

	skipEngine, _ := inputMap["skip_engine"].(bool)
	skipModels, _ := inputMap["skip_models"].(bool)

	// Return a deployment plan. Actual pulling is handled by the service layer.
	engineReady := skipEngine
	modelStatuses := make([]map[string]any, 0, len(recipe.Models))
	modelsReady := make([]ModelStatus, 0, len(recipe.Models))

	if !skipModels {
		for _, m := range recipe.Models {
			modelStatuses = append(modelStatuses, map[string]any{
				"name":   m.Name,
				"status": "pending",
			})
			modelsReady = append(modelsReady, ModelStatus{Name: m.Name, Ready: false})
		}
	}

	if c.events != nil {
		if err := c.events.Publish(NewRecipeAppliedEvent(recipeID, engineReady, modelsReady)); err != nil {
			slog.Warn("failed to publish catalog.recipe_applied event", "error", err)
		}
	}

	return map[string]any{
		"engine_ready": engineReady,
		"models":       modelStatuses,
	}, nil
}

// --- helpers ---

func generateRecipeID() string {
	return "recipe-" + uuid.New().String()[:8]
}

func parseHardwareProfile(m map[string]any) HardwareProfile {
	p := HardwareProfile{}
	if v, ok := m["gpu_vendor"].(string); ok {
		p.GPUVendor = v
	}
	if v, ok := m["gpu_model"].(string); ok {
		p.GPUModel = v
	}
	if v, ok := m["gpu_arch"].(string); ok {
		p.GPUArch = v
	}
	if v, ok := toInt(m["vram_min_gb"]); ok {
		p.VRAMMinGB = v
	}
	if v, ok := m["cpu_arch"].(string); ok {
		p.CPUArch = v
	}
	if v, ok := m["os"].(string); ok {
		p.OS = v
	}
	if v, ok := m["unified_memory"].(bool); ok {
		p.UnifiedMem = v
	}
	if tagsRaw, ok := m["tags"].([]any); ok {
		for _, t := range tagsRaw {
			if tag, ok := t.(string); ok {
				p.Tags = append(p.Tags, tag)
			}
		}
	}
	return p
}

func parseRecipeEngine(m map[string]any) RecipeEngine {
	e := RecipeEngine{}
	if v, ok := m["type"].(string); ok {
		e.Type = v
	}
	if v, ok := m["image"].(string); ok {
		e.Image = v
	}
	if v, ok := m["config"].(map[string]any); ok {
		e.Config = v
	}
	if fbRaw, ok := m["fallback_images"].([]any); ok {
		for _, fb := range fbRaw {
			if s, ok := fb.(string); ok {
				e.FallbackImages = append(e.FallbackImages, s)
			}
		}
	}
	return e
}

func parseRecipeModel(m map[string]any) RecipeModel {
	rm := RecipeModel{}
	if v, ok := m["name"].(string); ok {
		rm.Name = v
	}
	if v, ok := m["source"].(string); ok {
		rm.Source = v
	}
	if v, ok := m["repo"].(string); ok {
		rm.Repo = v
	}
	if v, ok := m["tag"].(string); ok {
		rm.Tag = v
	}
	if v, ok := m["type"].(string); ok {
		rm.Type = v
	}
	if v, ok := m["format"].(string); ok {
		rm.Format = v
	}
	if v, ok := m["mirror"].(string); ok {
		rm.Mirror = v
	}
	if v, ok := toInt64(m["memory_required"]); ok {
		rm.MemoryRequired = v
	}
	return rm
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

func toInt64(v any) (int64, bool) {
	switch val := v.(type) {
	case int:
		return int64(val), true
	case int32:
		return int64(val), true
	case int64:
		return val, true
	case float64:
		return int64(val), true
	case float32:
		return int64(val), true
	default:
		return 0, false
	}
}

