package skill

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// builtinSkills returns 5 test builtin skills matching the architecture spec.
func builtinSkills() []*Skill {
	return []*Skill{
		{
			ID:          "setup-llm",
			Name:        "Deploy LLM on New Hardware",
			Category:    CategorySetup,
			Description: "Guide Agent to set up LLM inference on a new machine",
			Trigger: SkillTrigger{
				Keywords:  []string{"setup", "install", "configure", "deploy"},
				ToolNames: []string{"catalog_match", "catalog_apply_recipe"},
				AlwaysOn:  false,
			},
			Content:  "# Deploy LLM\n\nStep 1: Detect hardware.\nStep 2: Match recipe.\nStep 3: Deploy.",
			Priority: 10,
			Enabled:  true,
			Source:   SourceBuiltin,
		},
		{
			ID:          "troubleshoot-gpu",
			Name:        "GPU Troubleshooting",
			Category:    CategoryTroubleshoot,
			Description: "Diagnose and fix GPU issues",
			Trigger: SkillTrigger{
				Keywords: []string{"gpu", "cuda", "error", "crash"},
				AlwaysOn: false,
			},
			Content:  "# GPU Troubleshooting\n\nCheck drivers.\nCheck VRAM.\nRestart engine.",
			Priority: 8,
			Enabled:  true,
			Source:   SourceBuiltin,
		},
		{
			ID:          "optimize-inference",
			Name:        "Inference Performance Optimization",
			Category:    CategoryOptimize,
			Description: "Optimize inference throughput and latency",
			Trigger: SkillTrigger{
				Keywords:  []string{"slow", "performance", "optimize", "throughput", "latency"},
				ToolNames: []string{"resource_status"},
				AlwaysOn:  false,
			},
			Content:  "# Optimize Inference\n\nAdjust batch size.\nTune resource limits.\nEnable quantization.",
			Priority: 7,
			Enabled:  true,
			Source:   SourceBuiltin,
		},
		{
			ID:          "manage-models",
			Name:        "Model Management Best Practices",
			Category:    CategoryManage,
			Description: "Best practices for managing models",
			Trigger: SkillTrigger{
				Keywords:  []string{"model", "list", "delete", "update"},
				ToolNames: []string{"model_list", "model_delete"},
				AlwaysOn:  false,
			},
			Content:  "# Model Management\n\nList models regularly.\nDelete unused models.\nKeep versions consistent.",
			Priority: 6,
			Enabled:  true,
			Source:   SourceBuiltin,
		},
		{
			ID:          "recipe-advisor",
			Name:        "Recipe Selection Guide",
			Category:    CategorySetup,
			Description: "Guide for selecting and customizing recipes",
			Trigger: SkillTrigger{
				Keywords:  []string{"recipe", "hardware", "recommend"},
				ToolNames: []string{"catalog_list", "catalog_match"},
				AlwaysOn:  true,
			},
			Content:  "# Recipe Advisor\n\nMatch hardware profile.\nReview recipe details.\nApply with one command.",
			Priority: 9,
			Enabled:  true,
			Source:   SourceBuiltin,
		},
	}
}

func newStoreWithBuiltins(t *testing.T) *MemoryStore {
	t.Helper()
	store := NewMemoryStore()
	ctx := context.Background()
	for _, sk := range builtinSkills() {
		require.NoError(t, store.Add(ctx, sk))
	}
	return store
}

func TestMemoryStore_AddAndGet(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	sk := &Skill{
		ID:      "test-skill",
		Name:    "Test Skill",
		Source:  SourceUser,
		Enabled: true,
	}

	err := store.Add(ctx, sk)
	require.NoError(t, err)

	got, err := store.Get(ctx, "test-skill")
	require.NoError(t, err)
	assert.Equal(t, "test-skill", got.ID)
	assert.Equal(t, "Test Skill", got.Name)
}

func TestMemoryStore_Add_Duplicate(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	sk := &Skill{ID: "dup", Name: "Dup", Source: SourceUser}
	require.NoError(t, store.Add(ctx, sk))
	err := store.Add(ctx, sk)
	assert.ErrorIs(t, err, ErrSkillAlreadyExists)
}

func TestMemoryStore_Get_NotFound(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	_, err := store.Get(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrSkillNotFound)
}

func TestMemoryStore_List(t *testing.T) {
	store := newStoreWithBuiltins(t)
	ctx := context.Background()

	skills, total, err := store.List(ctx, SkillFilter{})
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, skills, 5)
}

func TestMemoryStore_List_FilterCategory(t *testing.T) {
	store := newStoreWithBuiltins(t)
	ctx := context.Background()

	skills, total, err := store.List(ctx, SkillFilter{Category: CategorySetup})
	require.NoError(t, err)
	assert.Equal(t, 2, total) // setup-llm + recipe-advisor
	for _, sk := range skills {
		assert.Equal(t, CategorySetup, sk.Category)
	}
}

func TestMemoryStore_List_FilterSource(t *testing.T) {
	store := newStoreWithBuiltins(t)
	ctx := context.Background()

	userSkill := &Skill{ID: "user-1", Name: "User Skill", Source: SourceUser, Enabled: true}
	require.NoError(t, store.Add(ctx, userSkill))

	skills, total, err := store.List(ctx, SkillFilter{Source: SourceUser})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "user-1", skills[0].ID)
}

func TestMemoryStore_List_FilterEnabledOnly(t *testing.T) {
	store := newStoreWithBuiltins(t)
	ctx := context.Background()

	disabled := &Skill{ID: "disabled-1", Name: "Disabled", Source: SourceUser, Enabled: false}
	require.NoError(t, store.Add(ctx, disabled))

	skills, _, err := store.List(ctx, SkillFilter{EnabledOnly: true})
	require.NoError(t, err)
	for _, sk := range skills {
		assert.True(t, sk.Enabled)
	}
}

func TestMemoryStore_Remove_Builtin(t *testing.T) {
	store := newStoreWithBuiltins(t)
	ctx := context.Background()

	err := store.Remove(ctx, "setup-llm")
	assert.ErrorIs(t, err, ErrBuiltinSkillImmutable)
}

func TestMemoryStore_Remove_User(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	sk := &Skill{ID: "user-skill", Name: "User Skill", Source: SourceUser}
	require.NoError(t, store.Add(ctx, sk))

	err := store.Remove(ctx, "user-skill")
	require.NoError(t, err)

	_, err = store.Get(ctx, "user-skill")
	assert.ErrorIs(t, err, ErrSkillNotFound)
}

func TestMemoryStore_Remove_NotFound(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	err := store.Remove(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrSkillNotFound)
}

func TestMemoryStore_Update(t *testing.T) {
	store := newStoreWithBuiltins(t)
	ctx := context.Background()

	sk, err := store.Get(ctx, "setup-llm")
	require.NoError(t, err)

	sk.Enabled = false
	require.NoError(t, store.Update(ctx, sk))

	updated, err := store.Get(ctx, "setup-llm")
	require.NoError(t, err)
	assert.False(t, updated.Enabled)
}

func TestMemoryStore_Search(t *testing.T) {
	store := newStoreWithBuiltins(t)
	ctx := context.Background()

	results, err := store.Search(ctx, "gpu", "")
	require.NoError(t, err)
	require.NotEmpty(t, results)
	assert.Equal(t, "troubleshoot-gpu", results[0].ID)
}

func TestMemoryStore_Search_WithCategory(t *testing.T) {
	store := newStoreWithBuiltins(t)
	ctx := context.Background()

	results, err := store.Search(ctx, "recipe", CategorySetup)
	require.NoError(t, err)
	for _, sk := range results {
		assert.Equal(t, CategorySetup, sk.Category)
	}
}

func TestParseSkillFile(t *testing.T) {
	content := `---
id: test-parse
name: "Test Parse Skill"
category: manage
description: "A test skill"
trigger:
  keywords: ["test", "parse"]
  always_on: false
priority: 5
enabled: true
source: user
---

# Test Parse Skill

This is the skill body.`

	sk, err := ParseSkillFile(content)
	require.NoError(t, err)
	assert.Equal(t, "test-parse", sk.ID)
	assert.Equal(t, "Test Parse Skill", sk.Name)
	assert.Equal(t, CategoryManage, sk.Category)
	assert.Equal(t, "user", sk.Source)
	assert.Equal(t, []string{"test", "parse"}, sk.Trigger.Keywords)
	assert.True(t, sk.Enabled)
	assert.Contains(t, sk.Content, "This is the skill body.")
}

func TestParseSkillFile_NoFrontMatter(t *testing.T) {
	_, err := ParseSkillFile("# Just a markdown file")
	assert.ErrorIs(t, err, ErrSkillInvalid)
}

func TestParseSkillFile_MissingID(t *testing.T) {
	content := "---\nname: No ID Skill\ncategory: manage\n---\n\nBody."
	_, err := ParseSkillFile(content)
	assert.ErrorIs(t, err, ErrSkillInvalid)
}
