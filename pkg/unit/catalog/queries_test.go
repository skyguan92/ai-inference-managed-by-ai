package catalog

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedStore(t *testing.T) *MemoryStore {
	t.Helper()
	store := NewMemoryStore()
	ctx := context.Background()

	recipes := []*Recipe{
		{
			ID: "r1", Name: "NVIDIA RTX 4090 LLM", Verified: true,
			Tags: []string{"llm", "24gb"},
			Profile: HardwareProfile{
				GPUVendor: "NVIDIA", GPUModel: "RTX 4090", GPUArch: "sm_89", VRAMMinGB: 24, OS: "linux",
			},
			Engine: RecipeEngine{Type: "vllm", Image: "vllm:latest"},
			Models: []RecipeModel{{Name: "Llama3", Source: "ollama", Repo: "llama3", Type: "llm"}},
		},
		{
			ID: "r2", Name: "NVIDIA A100 LLM", Verified: true,
			Tags: []string{"llm", "80gb"},
			Profile: HardwareProfile{
				GPUVendor: "NVIDIA", GPUModel: "A100", GPUArch: "sm_80", VRAMMinGB: 80, OS: "linux",
			},
			Engine: RecipeEngine{Type: "vllm", Image: "vllm:latest"},
			Models: []RecipeModel{{Name: "Llama3 70B", Source: "ollama", Repo: "llama3:70b", Type: "llm"}},
		},
		{
			ID: "r3", Name: "Apple Silicon LLM", Verified: false,
			Tags: []string{"llm", "apple"},
			Profile: HardwareProfile{
				GPUVendor: "Apple", GPUModel: "M3 Max", VRAMMinGB: 36, OS: "darwin", UnifiedMem: true,
			},
			Engine: RecipeEngine{Type: "ollama", Image: "ollama/ollama:latest"},
			Models: []RecipeModel{{Name: "Llama3", Source: "ollama", Repo: "llama3", Type: "llm"}},
		},
	}

	for _, r := range recipes {
		require.NoError(t, store.Create(ctx, r))
	}
	return store
}

func TestMatchQuery(t *testing.T) {
	store := seedStore(t)
	q := NewMatchQuery(store)
	ctx := context.Background()

	t.Run("metadata", func(t *testing.T) {
		assert.Equal(t, "catalog.match", q.Name())
		assert.Equal(t, "catalog", q.Domain())
	})

	t.Run("exact NVIDIA RTX 4090 match gets highest score", func(t *testing.T) {
		input := map[string]any{
			"gpu_vendor": "NVIDIA",
			"gpu_model":  "RTX 4090",
			"gpu_arch":   "sm_89",
			"vram_gb":    24,
			"os":         "linux",
		}
		result, err := q.Execute(ctx, input)
		require.NoError(t, err)

		m := result.(map[string]any)
		recipes := m["recipes"].([]map[string]any)
		require.NotEmpty(t, recipes)

		// RTX 4090 should be first with highest score
		top := recipes[0]
		recipe := top["recipe"].(map[string]any)
		assert.Equal(t, "r1", recipe["id"])
		assert.Equal(t, 100, top["score"].(int)) // 40+30+15+10+5
	})

	t.Run("NVIDIA vendor match only", func(t *testing.T) {
		input := map[string]any{"gpu_vendor": "NVIDIA"}
		result, err := q.Execute(ctx, input)
		require.NoError(t, err)

		m := result.(map[string]any)
		recipes := m["recipes"].([]map[string]any)
		assert.Len(t, recipes, 2)
		// All results should have NVIDIA vendor
		for _, r := range recipes {
			recipe := r["recipe"].(map[string]any)
			profile := recipe["profile"].(map[string]any)
			assert.Equal(t, "NVIDIA", profile["gpu_vendor"])
		}
	})

	t.Run("Apple Silicon match", func(t *testing.T) {
		input := map[string]any{
			"gpu_vendor": "Apple",
			"gpu_model":  "M3 Max",
			"os":         "darwin",
			"vram_gb":    36,
		}
		result, err := q.Execute(ctx, input)
		require.NoError(t, err)

		m := result.(map[string]any)
		recipes := m["recipes"].([]map[string]any)
		require.NotEmpty(t, recipes)
		top := recipes[0]
		recipe := top["recipe"].(map[string]any)
		assert.Equal(t, "r3", recipe["id"])
		assert.Equal(t, 85, top["score"].(int)) // 40+30+10+5 (no arch match)
	})

	t.Run("no match returns empty", func(t *testing.T) {
		input := map[string]any{"gpu_vendor": "Intel"}
		result, err := q.Execute(ctx, input)
		require.NoError(t, err)
		m := result.(map[string]any)
		recipes := m["recipes"].([]map[string]any)
		assert.Empty(t, recipes)
	})

	t.Run("limit parameter", func(t *testing.T) {
		input := map[string]any{"gpu_vendor": "NVIDIA", "limit": 1}
		result, err := q.Execute(ctx, input)
		require.NoError(t, err)
		m := result.(map[string]any)
		recipes := m["recipes"].([]map[string]any)
		assert.Len(t, recipes, 1)
	})

	t.Run("nil store returns error", func(t *testing.T) {
		nilQ := NewMatchQuery(nil)
		_, err := nilQ.Execute(ctx, map[string]any{})
		assert.ErrorIs(t, err, ErrProviderNotSet)
	})
}

func TestScoreRecipe(t *testing.T) {
	recipe := Recipe{
		Profile: HardwareProfile{
			GPUVendor: "NVIDIA",
			GPUModel:  "RTX 4090",
			GPUArch:   "sm_89",
			VRAMMinGB: 24,
			OS:        "linux",
		},
	}

	tests := []struct {
		name          string
		hw            HardwareProfile
		expectedScore int
	}{
		{
			name: "full match",
			hw: HardwareProfile{
				GPUVendor: "NVIDIA", GPUModel: "RTX 4090", GPUArch: "sm_89", VRAMMinGB: 24, OS: "linux",
			},
			expectedScore: 100, // 40+30+15+10+5
		},
		{
			name:          "vendor only",
			hw:            HardwareProfile{GPUVendor: "NVIDIA"},
			expectedScore: 40,
		},
		{
			name:          "vendor and model",
			hw:            HardwareProfile{GPUVendor: "NVIDIA", GPUModel: "RTX 4090"},
			expectedScore: 70,
		},
		{
			name:          "insufficient VRAM",
			hw:            HardwareProfile{GPUVendor: "NVIDIA", VRAMMinGB: 8},
			expectedScore: 40, // VRAM too low, no +10
		},
		{
			name:          "no match",
			hw:            HardwareProfile{GPUVendor: "AMD"},
			expectedScore: 0,
		},
		{
			name:          "empty profile",
			hw:            HardwareProfile{},
			expectedScore: 0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			score := scoreRecipe(recipe, tc.hw)
			assert.Equal(t, tc.expectedScore, score)
		})
	}
}

func TestGetQuery(t *testing.T) {
	store := seedStore(t)
	q := NewGetQuery(store)
	ctx := context.Background()

	t.Run("metadata", func(t *testing.T) {
		assert.Equal(t, "catalog.get", q.Name())
		assert.Equal(t, "catalog", q.Domain())
	})

	t.Run("get existing recipe", func(t *testing.T) {
		result, err := q.Execute(ctx, map[string]any{"recipe_id": "r1"})
		require.NoError(t, err)
		m := result.(map[string]any)
		recipe := m["recipe"].(map[string]any)
		assert.Equal(t, "r1", recipe["id"])
		assert.Equal(t, "NVIDIA RTX 4090 LLM", recipe["name"])
	})

	t.Run("recipe not found", func(t *testing.T) {
		_, err := q.Execute(ctx, map[string]any{"recipe_id": "nonexistent"})
		assert.Error(t, err)
	})

	t.Run("missing recipe_id", func(t *testing.T) {
		_, err := q.Execute(ctx, map[string]any{})
		assert.Error(t, err)
	})

	t.Run("nil store returns error", func(t *testing.T) {
		nilQ := NewGetQuery(nil)
		_, err := nilQ.Execute(ctx, map[string]any{"recipe_id": "r1"})
		assert.ErrorIs(t, err, ErrProviderNotSet)
	})
}

func TestListQuery(t *testing.T) {
	store := seedStore(t)
	q := NewListQuery(store)
	ctx := context.Background()

	t.Run("metadata", func(t *testing.T) {
		assert.Equal(t, "catalog.list", q.Name())
		assert.Equal(t, "catalog", q.Domain())
	})

	t.Run("list all", func(t *testing.T) {
		result, err := q.Execute(ctx, map[string]any{})
		require.NoError(t, err)
		m := result.(map[string]any)
		assert.Equal(t, 3, m["total"].(int))
	})

	t.Run("filter verified only", func(t *testing.T) {
		result, err := q.Execute(ctx, map[string]any{"verified_only": true})
		require.NoError(t, err)
		m := result.(map[string]any)
		assert.Equal(t, 2, m["total"].(int))
	})

	t.Run("filter by gpu_vendor", func(t *testing.T) {
		result, err := q.Execute(ctx, map[string]any{"gpu_vendor": "NVIDIA"})
		require.NoError(t, err)
		m := result.(map[string]any)
		assert.Equal(t, 2, m["total"].(int))
	})

	t.Run("nil store returns error", func(t *testing.T) {
		nilQ := NewListQuery(nil)
		_, err := nilQ.Execute(ctx, map[string]any{})
		assert.ErrorIs(t, err, ErrProviderNotSet)
	})
}

func TestCheckStatusQuery(t *testing.T) {
	store := seedStore(t)
	q := NewCheckStatusQuery(store)
	ctx := context.Background()

	t.Run("metadata", func(t *testing.T) {
		assert.Equal(t, "catalog.check_status", q.Name())
		assert.Equal(t, "catalog", q.Domain())
	})

	t.Run("check existing recipe", func(t *testing.T) {
		result, err := q.Execute(ctx, map[string]any{"recipe_id": "r1"})
		require.NoError(t, err)
		m := result.(map[string]any)
		assert.Contains(t, m, "engine_ready")
		assert.Contains(t, m, "models_ready")
		modelsReady := m["models_ready"].([]map[string]any)
		assert.Len(t, modelsReady, 1)
		assert.Equal(t, "Llama3", modelsReady[0]["name"])
	})

	t.Run("recipe not found", func(t *testing.T) {
		_, err := q.Execute(ctx, map[string]any{"recipe_id": "nonexistent"})
		assert.Error(t, err)
	})

	t.Run("nil store returns error", func(t *testing.T) {
		nilQ := NewCheckStatusQuery(nil)
		_, err := nilQ.Execute(ctx, map[string]any{"recipe_id": "r1"})
		assert.ErrorIs(t, err, ErrProviderNotSet)
	})
}
