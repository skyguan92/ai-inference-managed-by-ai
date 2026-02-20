package catalog

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecipesResource(t *testing.T) {
	store := seedStore(t)
	r := NewRecipesResource(store)
	ctx := context.Background()

	t.Run("URI and domain", func(t *testing.T) {
		assert.Equal(t, "asms://catalog/recipes", r.URI())
		assert.Equal(t, "catalog", r.Domain())
	})

	t.Run("Get returns all recipes", func(t *testing.T) {
		data, err := r.Get(ctx)
		require.NoError(t, err)
		m := data.(map[string]any)
		assert.Equal(t, 3, m["total"].(int))
		recipes := m["recipes"].([]map[string]any)
		assert.Len(t, recipes, 3)
	})

	t.Run("nil store returns error", func(t *testing.T) {
		nilR := NewRecipesResource(nil)
		_, err := nilR.Get(ctx)
		assert.ErrorIs(t, err, ErrProviderNotSet)
	})

	t.Run("Schema is defined", func(t *testing.T) {
		schema := r.Schema()
		assert.Equal(t, "object", schema.Type)
	})
}

func TestRecipeResourceFactory(t *testing.T) {
	store := seedStore(t)
	factory := NewRecipeResourceFactory(store)

	t.Run("Pattern", func(t *testing.T) {
		assert.Equal(t, "asms://catalog/recipe/*", factory.Pattern())
	})

	t.Run("CanCreate matches catalog recipe URIs", func(t *testing.T) {
		assert.True(t, factory.CanCreate("asms://catalog/recipe/r1"))
		assert.False(t, factory.CanCreate("asms://model/m1"))
		assert.False(t, factory.CanCreate("asms://catalog/recipes"))
	})

	t.Run("Create returns resource", func(t *testing.T) {
		res, err := factory.Create("asms://catalog/recipe/r1")
		require.NoError(t, err)
		assert.Equal(t, "asms://catalog/recipe/r1", res.URI())
	})

	t.Run("Create with empty ID returns error", func(t *testing.T) {
		_, err := factory.Create("asms://catalog/recipe/")
		assert.Error(t, err)
	})
}

func TestRecipeResource(t *testing.T) {
	store := seedStore(t)
	r := NewRecipeResource("r1", store)
	ctx := context.Background()

	t.Run("URI and domain", func(t *testing.T) {
		assert.Equal(t, "asms://catalog/recipe/r1", r.URI())
		assert.Equal(t, "catalog", r.Domain())
	})

	t.Run("Get returns recipe data", func(t *testing.T) {
		data, err := r.Get(ctx)
		require.NoError(t, err)
		m := data.(map[string]any)
		assert.Equal(t, "r1", m["id"])
		assert.Equal(t, "NVIDIA RTX 4090 LLM", m["name"])
	})

	t.Run("Get not found", func(t *testing.T) {
		notFoundR := NewRecipeResource("nonexistent", store)
		_, err := notFoundR.Get(ctx)
		assert.Error(t, err)
	})

	t.Run("Schema is defined", func(t *testing.T) {
		schema := r.Schema()
		assert.Equal(t, "object", schema.Type)
		assert.Contains(t, schema.Properties, "id")
		assert.Contains(t, schema.Properties, "profile")
	})

	t.Run("Watch closes on context cancel", func(t *testing.T) {
		cancelCtx, cancel := context.WithCancel(ctx)
		ch, err := r.Watch(cancelCtx)
		require.NoError(t, err)
		cancel()
		// Channel should close after context cancel
		_, open := <-ch
		assert.False(t, open)
	})
}
