package skill

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListQuery_Execute(t *testing.T) {
	store := newStoreWithBuiltins(t)
	q := NewListQuery(store)
	ctx := context.Background()

	output, err := q.Execute(ctx, map[string]any{})
	require.NoError(t, err)

	result := output.(map[string]any)
	skills := result["skills"].([]map[string]any)
	assert.Len(t, skills, 5)
	assert.Equal(t, 5, result["total"])
}

func TestListQuery_Execute_FilterCategory(t *testing.T) {
	store := newStoreWithBuiltins(t)
	q := NewListQuery(store)
	ctx := context.Background()

	output, err := q.Execute(ctx, map[string]any{"category": "setup"})
	require.NoError(t, err)

	result := output.(map[string]any)
	skills := result["skills"].([]map[string]any)
	assert.Equal(t, 2, result["total"])
	for _, sk := range skills {
		assert.Equal(t, "setup", sk["category"])
	}
}

func TestListQuery_Execute_EnabledOnly(t *testing.T) {
	store := newStoreWithBuiltins(t)
	ctx := context.Background()

	disabled := &Skill{ID: "off-skill", Name: "Off", Source: SourceUser, Enabled: false}
	require.NoError(t, store.Add(ctx, disabled))

	q := NewListQuery(store)
	output, err := q.Execute(ctx, map[string]any{"enabled_only": true})
	require.NoError(t, err)

	result := output.(map[string]any)
	skills := result["skills"].([]map[string]any)
	for _, sk := range skills {
		assert.Equal(t, true, sk["enabled"])
	}
}

func TestGetQuery_Execute(t *testing.T) {
	store := newStoreWithBuiltins(t)
	q := NewGetQuery(store)
	ctx := context.Background()

	output, err := q.Execute(ctx, map[string]any{"skill_id": "setup-llm"})
	require.NoError(t, err)

	result := output.(map[string]any)
	sk := result["skill"].(map[string]any)
	assert.Equal(t, "setup-llm", sk["id"])
	assert.Equal(t, "Deploy LLM on New Hardware", sk["name"])
}

func TestGetQuery_Execute_NotFound(t *testing.T) {
	store := NewMemoryStore()
	q := NewGetQuery(store)
	ctx := context.Background()

	_, err := q.Execute(ctx, map[string]any{"skill_id": "nonexistent"})
	assert.Error(t, err)
}

func TestGetQuery_Execute_MissingID(t *testing.T) {
	store := NewMemoryStore()
	q := NewGetQuery(store)
	ctx := context.Background()

	_, err := q.Execute(ctx, map[string]any{})
	assert.Error(t, err)
}

func TestSearchQuery_Execute(t *testing.T) {
	store := newStoreWithBuiltins(t)
	q := NewSearchQuery(store)
	ctx := context.Background()

	output, err := q.Execute(ctx, map[string]any{"query": "gpu"})
	require.NoError(t, err)

	result := output.(map[string]any)
	skills := result["skills"].([]map[string]any)
	require.NotEmpty(t, skills)
	assert.Equal(t, "troubleshoot-gpu", skills[0]["id"])
}

func TestSearchQuery_Execute_WithCategory(t *testing.T) {
	store := newStoreWithBuiltins(t)
	q := NewSearchQuery(store)
	ctx := context.Background()

	output, err := q.Execute(ctx, map[string]any{"query": "deploy", "category": "setup"})
	require.NoError(t, err)

	result := output.(map[string]any)
	skills := result["skills"].([]map[string]any)
	for _, sk := range skills {
		assert.Equal(t, "setup", sk["category"])
	}
}

func TestSearchQuery_Execute_MissingQuery(t *testing.T) {
	store := NewMemoryStore()
	q := NewSearchQuery(store)
	ctx := context.Background()

	_, err := q.Execute(ctx, map[string]any{})
	assert.Error(t, err)
}

func TestQueries_Metadata(t *testing.T) {
	store := NewMemoryStore()

	queries := []interface {
		Name() string
		Domain() string
	}{
		NewListQuery(store),
		NewGetQuery(store),
		NewSearchQuery(store),
	}

	expected := map[string]string{
		"skill.list":   "skill",
		"skill.get":    "skill",
		"skill.search": "skill",
	}

	for _, q := range queries {
		assert.Equal(t, expected[q.Name()], q.Domain())
	}
}
