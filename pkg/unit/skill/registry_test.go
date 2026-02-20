package skill

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkillRegistry_MatchSkills_Keywords(t *testing.T) {
	store := newStoreWithBuiltins(t)
	reg := NewSkillRegistry(store)
	ctx := context.Background()

	skills, err := reg.MatchSkills(ctx, "I need to setup a new LLM", nil)
	require.NoError(t, err)
	require.NotEmpty(t, skills)

	ids := make([]string, len(skills))
	for i, sk := range skills {
		ids[i] = sk.ID
	}
	assert.Contains(t, ids, "setup-llm")
}

func TestSkillRegistry_MatchSkills_ToolNames(t *testing.T) {
	store := newStoreWithBuiltins(t)
	reg := NewSkillRegistry(store)
	ctx := context.Background()

	// catalog_match triggers setup-llm
	skills, err := reg.MatchSkills(ctx, "what should I do?", []string{"catalog_match"})
	require.NoError(t, err)

	ids := make([]string, len(skills))
	for i, sk := range skills {
		ids[i] = sk.ID
	}
	assert.Contains(t, ids, "setup-llm")
}

func TestSkillRegistry_MatchSkills_ExcludesAlwaysOn(t *testing.T) {
	store := newStoreWithBuiltins(t)
	reg := NewSkillRegistry(store)
	ctx := context.Background()

	// recipe-advisor is always_on; it should NOT appear in MatchSkills
	skills, err := reg.MatchSkills(ctx, "recommend a recipe", nil)
	require.NoError(t, err)

	for _, sk := range skills {
		assert.NotEqual(t, "recipe-advisor", sk.ID, "always_on skill should not appear in MatchSkills")
	}
}

func TestSkillRegistry_MatchSkills_SortedByPriority(t *testing.T) {
	store := newStoreWithBuiltins(t)
	reg := NewSkillRegistry(store)
	ctx := context.Background()

	// "deploy" matches setup-llm (priority 10); "model" matches manage-models (priority 6)
	skills, err := reg.MatchSkills(ctx, "setup and manage models", nil)
	require.NoError(t, err)

	for i := 1; i < len(skills); i++ {
		assert.GreaterOrEqual(t, skills[i-1].Priority, skills[i].Priority)
	}
}

func TestSkillRegistry_GetAlwaysOnSkills(t *testing.T) {
	store := newStoreWithBuiltins(t)
	reg := NewSkillRegistry(store)
	ctx := context.Background()

	skills, err := reg.GetAlwaysOnSkills(ctx)
	require.NoError(t, err)
	require.Len(t, skills, 1)
	assert.Equal(t, "recipe-advisor", skills[0].ID)
}

func TestSkillRegistry_GetAlwaysOnSkills_DisabledExcluded(t *testing.T) {
	store := newStoreWithBuiltins(t)
	ctx := context.Background()

	// Disable the always_on skill
	sk, err := store.Get(ctx, "recipe-advisor")
	require.NoError(t, err)
	sk.Enabled = false
	require.NoError(t, store.Update(ctx, sk))

	reg := NewSkillRegistry(store)
	skills, err := reg.GetAlwaysOnSkills(ctx)
	require.NoError(t, err)
	assert.Empty(t, skills)
}

func TestSkillRegistry_FormatForSystemPrompt(t *testing.T) {
	store := newStoreWithBuiltins(t)
	reg := NewSkillRegistry(store)
	ctx := context.Background()

	alwaysOn, err := reg.GetAlwaysOnSkills(ctx)
	require.NoError(t, err)

	prompt := reg.FormatForSystemPrompt(alwaysOn)
	assert.Contains(t, prompt, "## Active Skills")
	assert.Contains(t, prompt, "Recipe Advisor")
	assert.Contains(t, prompt, "Recipe Advisor")
}

func TestSkillRegistry_FormatForSystemPrompt_Empty(t *testing.T) {
	store := NewMemoryStore()
	reg := NewSkillRegistry(store)

	result := reg.FormatForSystemPrompt(nil)
	assert.Empty(t, result)
}

func TestSkillRegistry_MatchSkills_NoMatch(t *testing.T) {
	store := newStoreWithBuiltins(t)
	reg := NewSkillRegistry(store)
	ctx := context.Background()

	skills, err := reg.MatchSkills(ctx, "random unrelated text xyz", nil)
	require.NoError(t, err)
	assert.Empty(t, skills)
}
