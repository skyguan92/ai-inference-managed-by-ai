package skill

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddCommand_Execute(t *testing.T) {
	store := NewMemoryStore()
	cmd := NewAddCommand(store)
	ctx := context.Background()

	content := "---\nid: new-skill\nname: New Skill\ncategory: manage\nenabled: true\nsource: user\n---\n\nBody."
	output, err := cmd.Execute(ctx, map[string]any{"content": content})
	require.NoError(t, err)

	result := output.(map[string]any)
	assert.Equal(t, "new-skill", result["skill_id"])

	sk, err := store.Get(ctx, "new-skill")
	require.NoError(t, err)
	assert.Equal(t, "New Skill", sk.Name)
}

func TestAddCommand_Execute_MissingContent(t *testing.T) {
	store := NewMemoryStore()
	cmd := NewAddCommand(store)
	ctx := context.Background()

	_, err := cmd.Execute(ctx, map[string]any{})
	assert.Error(t, err)
}

func TestAddCommand_Execute_InvalidContent(t *testing.T) {
	store := NewMemoryStore()
	cmd := NewAddCommand(store)
	ctx := context.Background()

	_, err := cmd.Execute(ctx, map[string]any{"content": "not front-matter"})
	assert.Error(t, err)
}

func TestAddCommand_Execute_BuiltinSourceRejected(t *testing.T) {
	store := NewMemoryStore()
	cmd := NewAddCommand(store)
	ctx := context.Background()

	content := "---\nid: evil-builtin\nname: Evil\ncategory: manage\nenabled: true\nsource: builtin\n---\n\nBody."
	_, err := cmd.Execute(ctx, map[string]any{"content": content, "source": "builtin"})
	assert.Error(t, err)
}

func TestRemoveCommand_Execute(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	sk := &Skill{ID: "removable", Name: "Removable", Source: SourceUser}
	require.NoError(t, store.Add(ctx, sk))

	cmd := NewRemoveCommand(store)
	output, err := cmd.Execute(ctx, map[string]any{"skill_id": "removable"})
	require.NoError(t, err)

	result := output.(map[string]any)
	assert.Equal(t, true, result["success"])

	_, err = store.Get(ctx, "removable")
	assert.ErrorIs(t, err, ErrSkillNotFound)
}

func TestRemoveCommand_Execute_Builtin(t *testing.T) {
	store := newStoreWithBuiltins(t)
	cmd := NewRemoveCommand(store)
	ctx := context.Background()

	_, err := cmd.Execute(ctx, map[string]any{"skill_id": "setup-llm"})
	assert.Error(t, err)
}

func TestRemoveCommand_Execute_MissingID(t *testing.T) {
	store := NewMemoryStore()
	cmd := NewRemoveCommand(store)
	ctx := context.Background()

	_, err := cmd.Execute(ctx, map[string]any{})
	assert.Error(t, err)
}

func TestEnableCommand_Execute(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()

	sk := &Skill{ID: "disabled-skill", Name: "Disabled", Source: SourceUser, Enabled: false}
	require.NoError(t, store.Add(ctx, sk))

	cmd := NewEnableCommand(store)
	output, err := cmd.Execute(ctx, map[string]any{"skill_id": "disabled-skill"})
	require.NoError(t, err)

	result := output.(map[string]any)
	assert.Equal(t, true, result["success"])

	updated, err := store.Get(ctx, "disabled-skill")
	require.NoError(t, err)
	assert.True(t, updated.Enabled)
}

func TestEnableCommand_Execute_NotFound(t *testing.T) {
	store := NewMemoryStore()
	cmd := NewEnableCommand(store)
	ctx := context.Background()

	_, err := cmd.Execute(ctx, map[string]any{"skill_id": "nonexistent"})
	assert.Error(t, err)
}

func TestDisableCommand_Execute(t *testing.T) {
	store := newStoreWithBuiltins(t)
	ctx := context.Background()
	cmd := NewDisableCommand(store)

	output, err := cmd.Execute(ctx, map[string]any{"skill_id": "setup-llm"})
	require.NoError(t, err)

	result := output.(map[string]any)
	assert.Equal(t, true, result["success"])

	updated, err := store.Get(ctx, "setup-llm")
	require.NoError(t, err)
	assert.False(t, updated.Enabled)
}

func TestDisableCommand_Execute_MissingID(t *testing.T) {
	store := NewMemoryStore()
	cmd := NewDisableCommand(store)
	ctx := context.Background()

	_, err := cmd.Execute(ctx, map[string]any{})
	assert.Error(t, err)
}

func TestCommands_Metadata(t *testing.T) {
	store := NewMemoryStore()

	cmds := []interface {
		Name() string
		Domain() string
	}{
		NewAddCommand(store),
		NewRemoveCommand(store),
		NewEnableCommand(store),
		NewDisableCommand(store),
	}

	expected := map[string]string{
		"skill.add":     "skill",
		"skill.remove":  "skill",
		"skill.enable":  "skill",
		"skill.disable": "skill",
	}

	for _, cmd := range cmds {
		assert.Equal(t, expected[cmd.Name()], cmd.Domain())
	}
}
