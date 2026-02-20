package catalog

import (
	"context"
	"fmt"
	"sort"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit/ptrs"
)

// MatchQuery matches recipes to a given hardware profile using the scoring algorithm.
type MatchQuery struct {
	store  RecipeStore
	events unit.EventPublisher
}

func NewMatchQuery(store RecipeStore) *MatchQuery {
	return &MatchQuery{store: store}
}

func NewMatchQueryWithEvents(store RecipeStore, events unit.EventPublisher) *MatchQuery {
	return &MatchQuery{store: store, events: events}
}

func (q *MatchQuery) Name() string        { return "catalog.match" }
func (q *MatchQuery) Domain() string      { return "catalog" }
func (q *MatchQuery) Description() string { return "Match best recipes based on hardware profile" }

func (q *MatchQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"gpu_vendor": {
				Name: "gpu_vendor",
				Schema: unit.Schema{
					Type:        "string",
					Description: "GPU vendor to match (NVIDIA, AMD, Apple)",
				},
			},
			"gpu_model": {
				Name: "gpu_model",
				Schema: unit.Schema{
					Type:        "string",
					Description: "GPU model to match",
				},
			},
			"gpu_arch": {
				Name: "gpu_arch",
				Schema: unit.Schema{
					Type:        "string",
					Description: "GPU architecture to match",
				},
			},
			"vram_gb": {
				Name: "vram_gb",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Available VRAM in GB",
				},
			},
			"os": {
				Name: "os",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Operating system (linux, windows, darwin)",
				},
			},
			"tags": {
				Name: "tags",
				Schema: unit.Schema{
					Type:        "array",
					Description: "Tags to filter by",
					Items:       &unit.Schema{Type: "string"},
				},
			},
			"limit": {
				Name: "limit",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Maximum number of results",
					Min:         ptrs.Float64(1),
					Max:         ptrs.Float64(50),
				},
			},
		},
	}
}

func (q *MatchQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"recipes": {
				Name: "recipes",
				Schema: unit.Schema{
					Type:        "array",
					Description: "Matched recipes ordered by score (descending)",
					Items:       &unit.Schema{Type: "object"},
				},
			},
		},
	}
}

func (q *MatchQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input: map[string]any{
				"gpu_vendor": "NVIDIA",
				"gpu_model":  "RTX 4090",
				"vram_gb":    24,
				"os":         "linux",
			},
			Output: map[string]any{
				"recipes": []map[string]any{
					{"recipe": map[string]any{"id": "recipe-abc", "name": "RTX 4090 LLM"}, "score": 85},
				},
			},
			Description: "Match recipes for an NVIDIA RTX 4090",
		},
	}
}

func (q *MatchQuery) Execute(ctx context.Context, input any) (any, error) {
	ec := unit.NewExecutionContext(q.events, q.Domain(), q.Name())
	ec.PublishStarted(input)

	if q.store == nil {
		err := ErrProviderNotSet
		ec.PublishFailed(err)
		return nil, err
	}

	inputMap, _ := input.(map[string]any)

	gpuVendor, _ := inputMap["gpu_vendor"].(string)
	gpuModel, _ := inputMap["gpu_model"].(string)
	gpuArch, _ := inputMap["gpu_arch"].(string)
	os, _ := inputMap["os"].(string)

	vramGB := 0
	if v, ok := toInt(inputMap["vram_gb"]); ok {
		vramGB = v
	}

	limit := 10
	if l, ok := toInt(inputMap["limit"]); ok && l > 0 {
		limit = l
	}

	var filterTags []string
	if tagsRaw, ok := inputMap["tags"].([]any); ok {
		for _, t := range tagsRaw {
			if tag, ok := t.(string); ok {
				filterTags = append(filterTags, tag)
			}
		}
	}

	recipes, _, err := q.store.List(ctx, RecipeFilter{Tags: filterTags, Limit: 0})
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("list recipes: %w", err)
	}

	profile := HardwareProfile{
		GPUVendor: gpuVendor,
		GPUModel:  gpuModel,
		GPUArch:   gpuArch,
		VRAMMinGB: vramGB,
		OS:        os,
	}

	matches := make([]MatchResult, 0, len(recipes))
	for _, r := range recipes {
		score := scoreRecipe(r, profile)
		if score > 0 {
			matches = append(matches, MatchResult{Recipe: r, Score: score})
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	if limit > 0 && len(matches) > limit {
		matches = matches[:limit]
	}

	results := make([]map[string]any, len(matches))
	for i, m := range matches {
		results[i] = map[string]any{
			"recipe": recipeToMap(m.Recipe),
			"score":  m.Score,
		}
	}

	output := map[string]any{"recipes": results}
	ec.PublishCompleted(output)
	return output, nil
}

// scoreRecipe computes a hardware match score for a recipe using the documented algorithm:
//
//	GPU Vendor exact match: +40
//	GPU Model exact match:  +30
//	GPU Arch compatible:    +15
//	VRAM satisfies minimum: +10
//	OS match:               +5
func scoreRecipe(recipe Recipe, hw HardwareProfile) int {
	score := 0
	p := recipe.Profile

	if hw.GPUVendor != "" && p.GPUVendor == hw.GPUVendor {
		score += 40
	}
	if hw.GPUModel != "" && p.GPUModel == hw.GPUModel {
		score += 30
	}
	if hw.GPUArch != "" && p.GPUArch == hw.GPUArch {
		score += 15
	}
	if hw.VRAMMinGB > 0 && hw.VRAMMinGB >= p.VRAMMinGB {
		score += 10
	}
	if hw.OS != "" && p.OS == hw.OS {
		score += 5
	}

	return score
}

// GetQuery retrieves a single recipe by ID.
type GetQuery struct {
	store  RecipeStore
	events unit.EventPublisher
}

func NewGetQuery(store RecipeStore) *GetQuery {
	return &GetQuery{store: store}
}

func NewGetQueryWithEvents(store RecipeStore, events unit.EventPublisher) *GetQuery {
	return &GetQuery{store: store, events: events}
}

func (q *GetQuery) Name() string        { return "catalog.get" }
func (q *GetQuery) Domain() string      { return "catalog" }
func (q *GetQuery) Description() string { return "Get a specific recipe by ID" }

func (q *GetQuery) InputSchema() unit.Schema {
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
		},
		Required: []string{"recipe_id"},
	}
}

func (q *GetQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"recipe": {Name: "recipe", Schema: unit.Schema{Type: "object"}},
		},
	}
}

func (q *GetQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input:       map[string]any{"recipe_id": "recipe-abc123"},
			Output:      map[string]any{"recipe": map[string]any{"id": "recipe-abc123", "name": "RTX 4090 LLM"}},
			Description: "Get a recipe by ID",
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

	recipeID, _ := inputMap["recipe_id"].(string)
	if recipeID == "" {
		err := fmt.Errorf("recipe_id is required: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	recipe, err := q.store.Get(ctx, recipeID)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("get recipe %s: %w", recipeID, err)
	}

	output := map[string]any{"recipe": recipeToMap(*recipe)}
	ec.PublishCompleted(output)
	return output, nil
}

// ListQuery lists recipes with optional filters.
type ListQuery struct {
	store  RecipeStore
	events unit.EventPublisher
}

func NewListQuery(store RecipeStore) *ListQuery {
	return &ListQuery{store: store}
}

func NewListQueryWithEvents(store RecipeStore, events unit.EventPublisher) *ListQuery {
	return &ListQuery{store: store, events: events}
}

func (q *ListQuery) Name() string        { return "catalog.list" }
func (q *ListQuery) Domain() string      { return "catalog" }
func (q *ListQuery) Description() string { return "List all recipes with optional filtering" }

func (q *ListQuery) InputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"tags": {
				Name: "tags",
				Schema: unit.Schema{
					Type:        "array",
					Description: "Filter by tags",
					Items:       &unit.Schema{Type: "string"},
				},
			},
			"gpu_vendor": {
				Name: "gpu_vendor",
				Schema: unit.Schema{
					Type:        "string",
					Description: "Filter by GPU vendor",
				},
			},
			"verified_only": {
				Name: "verified_only",
				Schema: unit.Schema{
					Type:        "boolean",
					Description: "Return only verified recipes",
				},
			},
			"limit": {
				Name: "limit",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Maximum number of results",
					Min:         ptrs.Float64(1),
					Max:         ptrs.Float64(100),
				},
			},
			"offset": {
				Name: "offset",
				Schema: unit.Schema{
					Type:        "number",
					Description: "Offset for pagination",
					Min:         ptrs.Float64(0),
				},
			},
		},
	}
}

func (q *ListQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"recipes": {
				Name: "recipes",
				Schema: unit.Schema{
					Type:  "array",
					Items: &unit.Schema{Type: "object"},
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
			Output:      map[string]any{"recipes": []map[string]any{{"id": "recipe-abc123", "name": "RTX 4090 LLM"}}, "total": 1},
			Description: "List all recipes",
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

	filter := RecipeFilter{Limit: 100}

	if gpuVendor, ok := inputMap["gpu_vendor"].(string); ok {
		filter.GPUVendor = gpuVendor
	}
	if verifiedOnly, ok := inputMap["verified_only"].(bool); ok {
		filter.VerifiedOnly = verifiedOnly
	}
	if tagsRaw, ok := inputMap["tags"].([]any); ok {
		for _, t := range tagsRaw {
			if tag, ok := t.(string); ok {
				filter.Tags = append(filter.Tags, tag)
			}
		}
	}
	if l, ok := toInt(inputMap["limit"]); ok && l > 0 {
		filter.Limit = l
	}
	if offset, ok := toInt(inputMap["offset"]); ok && offset >= 0 {
		filter.Offset = offset
	}

	recipes, total, err := q.store.List(ctx, filter)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("list recipes: %w", err)
	}

	items := make([]map[string]any, len(recipes))
	for i, r := range recipes {
		items[i] = recipeToMap(r)
	}

	output := map[string]any{
		"recipes": items,
		"total":   total,
	}
	ec.PublishCompleted(output)
	return output, nil
}

// CheckStatusQuery checks whether a recipe's engine and models are locally available.
type CheckStatusQuery struct {
	store  RecipeStore
	events unit.EventPublisher
}

func NewCheckStatusQuery(store RecipeStore) *CheckStatusQuery {
	return &CheckStatusQuery{store: store}
}

func NewCheckStatusQueryWithEvents(store RecipeStore, events unit.EventPublisher) *CheckStatusQuery {
	return &CheckStatusQuery{store: store, events: events}
}

func (q *CheckStatusQuery) Name() string        { return "catalog.check_status" }
func (q *CheckStatusQuery) Domain() string      { return "catalog" }
func (q *CheckStatusQuery) Description() string { return "Check if a recipe's artifacts are locally available" }

func (q *CheckStatusQuery) InputSchema() unit.Schema {
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
		},
		Required: []string{"recipe_id"},
	}
}

func (q *CheckStatusQuery) OutputSchema() unit.Schema {
	return unit.Schema{
		Type: "object",
		Properties: map[string]unit.Field{
			"engine_ready":  {Name: "engine_ready", Schema: unit.Schema{Type: "boolean"}},
			"models_ready":  {Name: "models_ready", Schema: unit.Schema{Type: "array", Items: &unit.Schema{Type: "object"}}},
		},
	}
}

func (q *CheckStatusQuery) Examples() []unit.Example {
	return []unit.Example{
		{
			Input: map[string]any{"recipe_id": "recipe-abc123"},
			Output: map[string]any{
				"engine_ready": true,
				"models_ready": []map[string]any{{"name": "llama3", "ready": true}},
			},
			Description: "Check recipe status",
		},
	}
}

func (q *CheckStatusQuery) Execute(ctx context.Context, input any) (any, error) {
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

	recipeID, _ := inputMap["recipe_id"].(string)
	if recipeID == "" {
		err := fmt.Errorf("recipe_id is required: %w", ErrInvalidInput)
		ec.PublishFailed(err)
		return nil, err
	}

	recipe, err := q.store.Get(ctx, recipeID)
	if err != nil {
		ec.PublishFailed(err)
		return nil, fmt.Errorf("get recipe %s: %w", recipeID, err)
	}

	// Without a real docker/model provider, we return false for all artifacts.
	// The service layer is responsible for actual availability checks.
	modelsReady := make([]map[string]any, len(recipe.Models))
	for i, m := range recipe.Models {
		modelsReady[i] = map[string]any{
			"name":  m.Name,
			"ready": false,
		}
	}

	output := map[string]any{
		"engine_ready": false,
		"models_ready": modelsReady,
	}
	ec.PublishCompleted(output)
	return output, nil
}

// recipeToMap converts a Recipe to a map for JSON serialisation.
func recipeToMap(r Recipe) map[string]any {
	models := make([]map[string]any, len(r.Models))
	for i, m := range r.Models {
		models[i] = map[string]any{
			"name":            m.Name,
			"source":          m.Source,
			"repo":            m.Repo,
			"tag":             m.Tag,
			"type":            m.Type,
			"format":          m.Format,
			"memory_required": m.MemoryRequired,
		}
	}

	return map[string]any{
		"id":          r.ID,
		"name":        r.Name,
		"description": r.Description,
		"version":     r.Version,
		"author":      r.Author,
		"verified":    r.Verified,
		"tags":        r.Tags,
		"profile": map[string]any{
			"gpu_vendor":     r.Profile.GPUVendor,
			"gpu_model":      r.Profile.GPUModel,
			"gpu_arch":       r.Profile.GPUArch,
			"vram_min_gb":    r.Profile.VRAMMinGB,
			"cpu_arch":       r.Profile.CPUArch,
			"os":             r.Profile.OS,
			"unified_memory": r.Profile.UnifiedMem,
		},
		"engine": map[string]any{
			"type":            r.Engine.Type,
			"image":           r.Engine.Image,
			"fallback_images": r.Engine.FallbackImages,
			"config":          r.Engine.Config,
		},
		"models": models,
	}
}
