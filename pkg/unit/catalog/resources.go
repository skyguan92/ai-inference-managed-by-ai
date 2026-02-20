package catalog

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

// RecipesResource is a static resource exposing all recipes.
// URI: asms://catalog/recipes
type RecipesResource struct {
	store RecipeStore
}

func NewRecipesResource(store RecipeStore) *RecipesResource {
	return &RecipesResource{store: store}
}

func (r *RecipesResource) URI() string    { return "asms://catalog/recipes" }
func (r *RecipesResource) Domain() string { return "catalog" }

func (r *RecipesResource) Schema() unit.Schema {
	return unit.Schema{
		Type:        "object",
		Description: "All available recipes in the catalog",
		Properties: map[string]unit.Field{
			"recipes": {
				Name: "recipes",
				Schema: unit.Schema{
					Type:        "array",
					Description: "List of all recipes",
					Items:       &unit.Schema{Type: "object"},
				},
			},
			"total": {Name: "total", Schema: unit.Schema{Type: "number"}},
		},
	}
}

func (r *RecipesResource) Get(ctx context.Context) (any, error) {
	if r.store == nil {
		return nil, ErrProviderNotSet
	}

	recipes, total, err := r.store.List(ctx, RecipeFilter{Limit: 0})
	if err != nil {
		return nil, fmt.Errorf("list recipes: %w", err)
	}

	items := make([]map[string]any, len(recipes))
	for i, recipe := range recipes {
		items[i] = recipeToMap(recipe)
	}

	return map[string]any{
		"recipes": items,
		"total":   total,
	}, nil
}

func (r *RecipesResource) Watch(ctx context.Context) (<-chan unit.ResourceUpdate, error) {
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

// RecipeResource is a factory-created resource for a specific recipe.
// URI pattern: asms://catalog/recipe/{id}
type RecipeResource struct {
	recipeID string
	store    RecipeStore
}

func NewRecipeResource(recipeID string, store RecipeStore) *RecipeResource {
	return &RecipeResource{recipeID: recipeID, store: store}
}

// RecipeResourceFactory creates RecipeResource instances dynamically.
type RecipeResourceFactory struct {
	store RecipeStore
}

func NewRecipeResourceFactory(store RecipeStore) *RecipeResourceFactory {
	return &RecipeResourceFactory{store: store}
}

func (f *RecipeResourceFactory) CanCreate(uri string) bool {
	return strings.HasPrefix(uri, "asms://catalog/recipe/")
}

func (f *RecipeResourceFactory) Create(uri string) (unit.Resource, error) {
	recipeID := strings.TrimPrefix(uri, "asms://catalog/recipe/")
	if recipeID == "" {
		return nil, fmt.Errorf("invalid catalog recipe URI: %s", uri)
	}
	return NewRecipeResource(recipeID, f.store), nil
}

func (f *RecipeResourceFactory) Pattern() string {
	return "asms://catalog/recipe/*"
}

func (r *RecipeResource) URI() string    { return fmt.Sprintf("asms://catalog/recipe/%s", r.recipeID) }
func (r *RecipeResource) Domain() string { return "catalog" }

func (r *RecipeResource) Schema() unit.Schema {
	return unit.Schema{
		Type:        "object",
		Description: "Specific recipe details",
		Properties: map[string]unit.Field{
			"id":          {Name: "id", Schema: unit.Schema{Type: "string"}},
			"name":        {Name: "name", Schema: unit.Schema{Type: "string"}},
			"description": {Name: "description", Schema: unit.Schema{Type: "string"}},
			"version":     {Name: "version", Schema: unit.Schema{Type: "string"}},
			"verified":    {Name: "verified", Schema: unit.Schema{Type: "boolean"}},
			"profile":     {Name: "profile", Schema: unit.Schema{Type: "object"}},
			"engine":      {Name: "engine", Schema: unit.Schema{Type: "object"}},
			"models":      {Name: "models", Schema: unit.Schema{Type: "array", Items: &unit.Schema{Type: "object"}}},
			"tags":        {Name: "tags", Schema: unit.Schema{Type: "array", Items: &unit.Schema{Type: "string"}}},
		},
	}
}

func (r *RecipeResource) Get(ctx context.Context) (any, error) {
	if r.store == nil {
		return nil, ErrProviderNotSet
	}

	recipe, err := r.store.Get(ctx, r.recipeID)
	if err != nil {
		return nil, fmt.Errorf("get recipe %s: %w", r.recipeID, err)
	}

	return recipeToMap(*recipe), nil
}

func (r *RecipeResource) Watch(ctx context.Context) (<-chan unit.ResourceUpdate, error) {
	ch := make(chan unit.ResourceUpdate, 1)
	// Recipes are relatively static; just signal done when context is cancelled.
	go func() {
		defer close(ch)
		<-ctx.Done()
	}()
	return ch, nil
}
