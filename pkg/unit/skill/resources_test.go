package skill

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkillsResource(t *testing.T) {
	store := newStoreWithBuiltins(t)
	r := NewSkillsResource(store)
	ctx := context.Background()

	assert.Equal(t, "asms://skills", r.URI())
	assert.Equal(t, "skill", r.Domain())

	data, err := r.Get(ctx)
	require.NoError(t, err)

	result := data.(map[string]any)
	skills := result["skills"].([]map[string]any)
	assert.Len(t, skills, 5)
	assert.Equal(t, 5, result["total"])
}

func TestSkillResource(t *testing.T) {
	store := newStoreWithBuiltins(t)
	r := NewSkillResource(store, "setup-llm")
	ctx := context.Background()

	assert.Equal(t, "asms://skill/setup-llm", r.URI())
	assert.Equal(t, "skill", r.Domain())

	data, err := r.Get(ctx)
	require.NoError(t, err)

	result := data.(map[string]any)
	assert.Equal(t, "setup-llm", result["id"])
	assert.Equal(t, "Deploy LLM on New Hardware", result["name"])
}

func TestSkillResource_NotFound(t *testing.T) {
	store := NewMemoryStore()
	r := NewSkillResource(store, "nonexistent")
	ctx := context.Background()

	_, err := r.Get(ctx)
	assert.Error(t, err)
}

func TestSkillResourceFactory_CanCreate(t *testing.T) {
	store := NewMemoryStore()
	f := NewSkillResourceFactory(store)

	assert.True(t, f.CanCreate("asms://skill/some-id"))
	assert.False(t, f.CanCreate("asms://skills"))
	assert.False(t, f.CanCreate("asms://other/resource"))
}

func TestSkillResourceFactory_Create(t *testing.T) {
	store := newStoreWithBuiltins(t)
	f := NewSkillResourceFactory(store)

	r, err := f.Create("asms://skill/setup-llm")
	require.NoError(t, err)
	assert.Equal(t, "asms://skill/setup-llm", r.URI())
}

func TestSkillResourceFactory_Create_InvalidURI(t *testing.T) {
	store := NewMemoryStore()
	f := NewSkillResourceFactory(store)

	_, err := f.Create("asms://other/resource")
	assert.Error(t, err)
}

func TestSkillResourceFactory_Pattern(t *testing.T) {
	store := NewMemoryStore()
	f := NewSkillResourceFactory(store)
	assert.Equal(t, "asms://skill/*", f.Pattern())
}

func TestSkillsResource_Schema(t *testing.T) {
	store := NewMemoryStore()
	r := NewSkillsResource(store)
	schema := r.Schema()
	assert.Equal(t, "object", schema.Type)
	assert.Contains(t, schema.Properties, "skills")
	assert.Contains(t, schema.Properties, "total")
}

func TestSkillResource_Schema(t *testing.T) {
	store := NewMemoryStore()
	r := NewSkillResource(store, "any")
	schema := r.Schema()
	assert.Equal(t, "object", schema.Type)
}
