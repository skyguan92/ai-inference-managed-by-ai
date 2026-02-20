package catalog

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryStore_CreateAndGet(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	recipe := createTestRecipe("recipe-001", "Test Recipe", "NVIDIA")

	err := store.Create(ctx, recipe)
	require.NoError(t, err)

	got, err := store.Get(ctx, "recipe-001")
	require.NoError(t, err)
	assert.Equal(t, recipe.ID, got.ID)
	assert.Equal(t, recipe.Name, got.Name)
}

func TestMemoryStore_CreateDuplicate(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	recipe := createTestRecipe("recipe-001", "Test Recipe", "NVIDIA")
	require.NoError(t, store.Create(ctx, recipe))

	err := store.Create(ctx, recipe)
	assert.ErrorIs(t, err, ErrRecipeAlreadyExists)
}

func TestMemoryStore_GetNotFound(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	_, err := store.Get(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrRecipeNotFound)
}

func TestMemoryStore_List(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	for i, tc := range []struct {
		id        string
		name      string
		gpuVendor string
	}{
		{"r1", "NVIDIA Recipe", "NVIDIA"},
		{"r2", "AMD Recipe", "AMD"},
		{"r3", "Apple Recipe", "Apple"},
	} {
		r := createTestRecipe(tc.id, tc.name, tc.gpuVendor)
		r.Verified = i%2 == 0 // r1 and r3 verified
		require.NoError(t, store.Create(ctx, r))
	}

	t.Run("list all", func(t *testing.T) {
		recipes, total, err := store.List(ctx, RecipeFilter{})
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, recipes, 3)
	})

	t.Run("filter by gpu_vendor", func(t *testing.T) {
		recipes, total, err := store.List(ctx, RecipeFilter{GPUVendor: "NVIDIA"})
		require.NoError(t, err)
		assert.Equal(t, 1, total)
		assert.Len(t, recipes, 1)
		assert.Equal(t, "NVIDIA", recipes[0].Profile.GPUVendor)
	})

	t.Run("filter verified only", func(t *testing.T) {
		recipes, total, err := store.List(ctx, RecipeFilter{VerifiedOnly: true})
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, recipes, 2)
		for _, r := range recipes {
			assert.True(t, r.Verified)
		}
	})

	t.Run("limit and offset", func(t *testing.T) {
		recipes, total, err := store.List(ctx, RecipeFilter{Limit: 2})
		require.NoError(t, err)
		assert.Equal(t, 3, total)
		assert.Len(t, recipes, 2)
	})
}

func TestMemoryStore_Delete(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	recipe := createTestRecipe("recipe-001", "Test Recipe", "NVIDIA")
	require.NoError(t, store.Create(ctx, recipe))

	err := store.Delete(ctx, "recipe-001")
	require.NoError(t, err)

	_, err = store.Get(ctx, "recipe-001")
	assert.ErrorIs(t, err, ErrRecipeNotFound)
}

func TestMemoryStore_DeleteNotFound(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	err := store.Delete(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrRecipeNotFound)
}

func TestMemoryStore_Update(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	recipe := createTestRecipe("recipe-001", "Old Name", "NVIDIA")
	require.NoError(t, store.Create(ctx, recipe))

	recipe.Name = "New Name"
	require.NoError(t, store.Update(ctx, recipe))

	got, err := store.Get(ctx, "recipe-001")
	require.NoError(t, err)
	assert.Equal(t, "New Name", got.Name)
}

func TestMemoryStore_ListByTag(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	r1 := createTestRecipe("r1", "Recipe 1", "NVIDIA")
	r1.Tags = []string{"llm", "24gb"}
	r2 := createTestRecipe("r2", "Recipe 2", "AMD")
	r2.Tags = []string{"asr"}

	require.NoError(t, store.Create(ctx, r1))
	require.NoError(t, store.Create(ctx, r2))

	recipes, total, err := store.List(ctx, RecipeFilter{Tags: []string{"llm"}})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "r1", recipes[0].ID)
}
