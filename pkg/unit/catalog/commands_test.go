package catalog

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateRecipeCommand(t *testing.T) {
	store := NewMemoryStore()
	cmd := NewCreateRecipeCommand(store)
	ctx := context.Background()

	t.Run("metadata", func(t *testing.T) {
		assert.Equal(t, "catalog.create_recipe", cmd.Name())
		assert.Equal(t, "catalog", cmd.Domain())
		assert.NotEmpty(t, cmd.Description())
		assert.NotEmpty(t, cmd.Examples())
	})

	t.Run("create recipe successfully", func(t *testing.T) {
		input := map[string]any{
			"name":        "NVIDIA RTX 4090 LLM",
			"description": "Test recipe",
			"version":     "1.0.0",
			"profile": map[string]any{
				"gpu_vendor":  "NVIDIA",
				"gpu_model":   "RTX 4090",
				"vram_min_gb": 24,
				"os":          "linux",
			},
			"engine": map[string]any{
				"type":  "vllm",
				"image": "vllm/vllm-openai:latest",
			},
			"models": []any{
				map[string]any{
					"name":   "Llama 3",
					"source": "ollama",
					"repo":   "llama3:8b",
					"type":   "llm",
				},
			},
			"verified": true,
			"tags":     []any{"llm", "consumer"},
		}

		result, err := cmd.Execute(ctx, input)
		require.NoError(t, err)

		resultMap, ok := result.(map[string]any)
		require.True(t, ok)
		recipeID, ok := resultMap["recipe_id"].(string)
		require.True(t, ok)
		assert.NotEmpty(t, recipeID)

		// Verify stored
		recipe, err := store.Get(ctx, recipeID)
		require.NoError(t, err)
		assert.Equal(t, "NVIDIA RTX 4090 LLM", recipe.Name)
		assert.Equal(t, "NVIDIA", recipe.Profile.GPUVendor)
		assert.Equal(t, "vllm", recipe.Engine.Type)
		assert.Len(t, recipe.Models, 1)
		assert.True(t, recipe.Verified)
		assert.Equal(t, []string{"llm", "consumer"}, recipe.Tags)
	})

	t.Run("missing name returns error", func(t *testing.T) {
		input := map[string]any{
			"profile": map[string]any{"gpu_vendor": "NVIDIA"},
			"engine":  map[string]any{"type": "vllm", "image": "vllm:latest"},
		}
		_, err := cmd.Execute(ctx, input)
		assert.Error(t, err)
	})

	t.Run("invalid input type returns error", func(t *testing.T) {
		_, err := cmd.Execute(ctx, "not-a-map")
		assert.Error(t, err)
	})

	t.Run("nil store returns error", func(t *testing.T) {
		nilCmd := NewCreateRecipeCommand(nil)
		_, err := nilCmd.Execute(ctx, map[string]any{"name": "test", "profile": map[string]any{}, "engine": map[string]any{}})
		assert.ErrorIs(t, err, ErrProviderNotSet)
	})
}

func TestValidateRecipeCommand(t *testing.T) {
	cmd := NewValidateRecipeCommand()
	ctx := context.Background()

	t.Run("metadata", func(t *testing.T) {
		assert.Equal(t, "catalog.validate_recipe", cmd.Name())
		assert.Equal(t, "catalog", cmd.Domain())
	})

	t.Run("valid recipe", func(t *testing.T) {
		input := map[string]any{
			"recipe": map[string]any{
				"name":    "Test Recipe",
				"profile": map[string]any{"gpu_vendor": "NVIDIA"},
				"engine":  map[string]any{"type": "vllm", "image": "vllm:latest"},
			},
		}
		result, err := cmd.Execute(ctx, input)
		require.NoError(t, err)
		m := result.(map[string]any)
		assert.True(t, m["valid"].(bool))
		assert.Empty(t, m["issues"])
	})

	t.Run("missing name", func(t *testing.T) {
		input := map[string]any{
			"recipe": map[string]any{
				"profile": map[string]any{"gpu_vendor": "NVIDIA"},
				"engine":  map[string]any{"type": "vllm", "image": "vllm:latest"},
			},
		}
		result, err := cmd.Execute(ctx, input)
		require.NoError(t, err)
		m := result.(map[string]any)
		assert.False(t, m["valid"].(bool))
		assert.NotEmpty(t, m["issues"])
	})

	t.Run("missing engine", func(t *testing.T) {
		input := map[string]any{
			"recipe": map[string]any{
				"name":    "Test",
				"profile": map[string]any{"gpu_vendor": "NVIDIA"},
			},
		}
		result, err := cmd.Execute(ctx, input)
		require.NoError(t, err)
		m := result.(map[string]any)
		assert.False(t, m["valid"].(bool))
	})

	t.Run("missing profile", func(t *testing.T) {
		input := map[string]any{
			"recipe": map[string]any{
				"name":   "Test",
				"engine": map[string]any{"type": "vllm", "image": "vllm:latest"},
			},
		}
		result, err := cmd.Execute(ctx, input)
		require.NoError(t, err)
		m := result.(map[string]any)
		assert.False(t, m["valid"].(bool))
	})
}

func TestApplyRecipeCommand(t *testing.T) {
	store := NewMemoryStore()
	recipe := createTestRecipe("recipe-001", "Test Recipe", "NVIDIA")
	require.NoError(t, store.Create(context.Background(), recipe))

	cmd := NewApplyRecipeCommand(store)
	ctx := context.Background()

	t.Run("metadata", func(t *testing.T) {
		assert.Equal(t, "catalog.apply_recipe", cmd.Name())
		assert.Equal(t, "catalog", cmd.Domain())
	})

	t.Run("apply recipe", func(t *testing.T) {
		input := map[string]any{"recipe_id": "recipe-001"}
		result, err := cmd.Execute(ctx, input)
		require.NoError(t, err)

		m := result.(map[string]any)
		assert.Contains(t, m, "engine_ready")
		assert.Contains(t, m, "models")
	})

	t.Run("skip engine", func(t *testing.T) {
		input := map[string]any{"recipe_id": "recipe-001", "skip_engine": true}
		result, err := cmd.Execute(ctx, input)
		require.NoError(t, err)
		m := result.(map[string]any)
		assert.True(t, m["engine_ready"].(bool))
	})

	t.Run("skip models", func(t *testing.T) {
		input := map[string]any{"recipe_id": "recipe-001", "skip_models": true}
		result, err := cmd.Execute(ctx, input)
		require.NoError(t, err)
		m := result.(map[string]any)
		models := m["models"].([]map[string]any)
		assert.Empty(t, models)
	})

	t.Run("recipe not found", func(t *testing.T) {
		input := map[string]any{"recipe_id": "nonexistent"}
		_, err := cmd.Execute(ctx, input)
		assert.Error(t, err)
	})

	t.Run("missing recipe_id", func(t *testing.T) {
		_, err := cmd.Execute(ctx, map[string]any{})
		assert.Error(t, err)
	})

	t.Run("nil store returns error", func(t *testing.T) {
		nilCmd := NewApplyRecipeCommand(nil)
		_, err := nilCmd.Execute(ctx, map[string]any{"recipe_id": "recipe-001"})
		assert.ErrorIs(t, err, ErrProviderNotSet)
	})
}
