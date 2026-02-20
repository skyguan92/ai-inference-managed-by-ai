package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func newSkillTestRoot(t *testing.T) *RootCommand {
	t.Helper()
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)
	return &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     NewOutputOptions(),
	}
}

func newSkillTestRootWithBuf(t *testing.T) (*RootCommand, *bytes.Buffer) {
	t.Helper()
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)
	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputJSON, Writer: buf},
	}
	return root, buf
}

// ---------------------------------------------------------------------------
// NewSkillCommand — structure
// ---------------------------------------------------------------------------

func TestNewSkillCommand(t *testing.T) {
	root := newSkillTestRoot(t)
	cmd := NewSkillCommand(root)

	assert.NotNil(t, cmd)
	assert.Equal(t, "skill", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
}

func TestNewSkillCommand_Subcommands(t *testing.T) {
	root := newSkillTestRoot(t)
	cmd := NewSkillCommand(root)

	subCmds := cmd.Commands()
	assert.Len(t, subCmds, 5)

	names := make([]string, len(subCmds))
	for i, c := range subCmds {
		names[i] = c.Name()
	}
	assert.Contains(t, names, "list")
	assert.Contains(t, names, "get")
	assert.Contains(t, names, "add")
	assert.Contains(t, names, "remove")
	assert.Contains(t, names, "search")
}

// ---------------------------------------------------------------------------
// NewSkillListCommand
// ---------------------------------------------------------------------------

func TestNewSkillListCommand_Structure(t *testing.T) {
	root := newSkillTestRoot(t)
	cmd := NewSkillListCommand(root)

	assert.NotNil(t, cmd)
	assert.Equal(t, "list", cmd.Use)
	assert.Contains(t, cmd.Aliases, "ls")
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Example)
}

func TestNewSkillListCommand_Flags(t *testing.T) {
	root := newSkillTestRoot(t)
	cmd := NewSkillListCommand(root)

	categoryFlag := cmd.Flags().Lookup("category")
	require.NotNil(t, categoryFlag)
	assert.Equal(t, "c", categoryFlag.Shorthand)
	assert.Equal(t, "", categoryFlag.DefValue)

	sourceFlag := cmd.Flags().Lookup("source")
	require.NotNil(t, sourceFlag)
	assert.Equal(t, "s", sourceFlag.Shorthand)
	assert.Equal(t, "", sourceFlag.DefValue)

	enabledFlag := cmd.Flags().Lookup("enabled")
	require.NotNil(t, enabledFlag)
	assert.Equal(t, "e", enabledFlag.Shorthand)
	assert.Equal(t, "false", enabledFlag.DefValue)
}

func TestRunSkillList_GatewayError(t *testing.T) {
	root, _ := newSkillTestRootWithBuf(t)

	err := runSkillList(context.Background(), root, "", "", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list skills failed")
}

func TestRunSkillList_WithFilters(t *testing.T) {
	tests := []struct {
		name        string
		category    string
		source      string
		enabledOnly bool
	}{
		{"no filters", "", "", false},
		{"setup category", "setup", "", false},
		{"troubleshoot category", "troubleshoot", "", false},
		{"optimize category", "optimize", "", false},
		{"manage category", "manage", "", false},
		{"builtin source", "", "builtin", false},
		{"user source", "", "user", false},
		{"community source", "", "community", false},
		{"enabled only", "", "", true},
		{"all filters", "setup", "builtin", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root, _ := newSkillTestRootWithBuf(t)
			err := runSkillList(context.Background(), root, tc.category, tc.source, tc.enabledOnly)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "list skills failed")
		})
	}
}

// ---------------------------------------------------------------------------
// NewSkillGetCommand
// ---------------------------------------------------------------------------

func TestNewSkillGetCommand_Structure(t *testing.T) {
	root := newSkillTestRoot(t)
	cmd := NewSkillGetCommand(root)

	assert.NotNil(t, cmd)
	assert.Equal(t, "get <skill-id>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Example)
	assert.NotNil(t, cmd.Args) // cobra.ExactArgs(1)
}

func TestRunSkillGet_GatewayError(t *testing.T) {
	root, _ := newSkillTestRootWithBuf(t)

	err := runSkillGet(context.Background(), root, "setup-llm")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get skill failed")
}

func TestRunSkillGet_DifferentIDs(t *testing.T) {
	ids := []string{"setup-llm", "troubleshoot-gpu", "optimize-vram", "manage-services"}
	for _, id := range ids {
		t.Run(id, func(t *testing.T) {
			root, _ := newSkillTestRootWithBuf(t)
			err := runSkillGet(context.Background(), root, id)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "get skill failed")
		})
	}
}

// ---------------------------------------------------------------------------
// NewSkillAddCommand
// ---------------------------------------------------------------------------

func TestNewSkillAddCommand_Structure(t *testing.T) {
	root := newSkillTestRoot(t)
	cmd := NewSkillAddCommand(root)

	assert.NotNil(t, cmd)
	assert.Equal(t, "add", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotEmpty(t, cmd.Example)
}

func TestNewSkillAddCommand_Flags(t *testing.T) {
	root := newSkillTestRoot(t)
	cmd := NewSkillAddCommand(root)

	fileFlag := cmd.Flags().Lookup("file")
	require.NotNil(t, fileFlag)
	assert.Equal(t, "f", fileFlag.Shorthand)
	assert.Equal(t, "", fileFlag.DefValue)

	sourceFlag := cmd.Flags().Lookup("source")
	require.NotNil(t, sourceFlag)
	assert.Equal(t, "s", sourceFlag.Shorthand)
	assert.Equal(t, "user", sourceFlag.DefValue)

	// --file is required
	annotations := fileFlag.Annotations
	require.NotNil(t, annotations)
	_, required := annotations["cobra_annotation_bash_completion_one_required_flag"]
	assert.True(t, required)
}

func TestRunSkillAdd_FileNotFound(t *testing.T) {
	root, _ := newSkillTestRootWithBuf(t)

	err := runSkillAdd(context.Background(), root, "/nonexistent/path/skill.md", "user")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read file")
}

func TestRunSkillAdd_ValidFile_GatewayError(t *testing.T) {
	// Write a temporary skill file.
	tmp := t.TempDir()
	skillPath := filepath.Join(tmp, "test-skill.md")
	content := `---
id: test-skill
name: "Test Skill"
category: manage
description: "A test skill"
enabled: true
source: user
---

# Test Skill

Some skill content here.
`
	require.NoError(t, os.WriteFile(skillPath, []byte(content), 0600))

	root, _ := newSkillTestRootWithBuf(t)
	err := runSkillAdd(context.Background(), root, skillPath, "user")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "add skill failed")
}

func TestRunSkillAdd_WithCommunitySource(t *testing.T) {
	tmp := t.TempDir()
	skillPath := filepath.Join(tmp, "community-skill.md")
	require.NoError(t, os.WriteFile(skillPath, []byte("# Community Skill\n\nContent."), 0600))

	root, _ := newSkillTestRootWithBuf(t)
	err := runSkillAdd(context.Background(), root, skillPath, "community")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "add skill failed")
}

// ---------------------------------------------------------------------------
// NewSkillRemoveCommand
// ---------------------------------------------------------------------------

func TestNewSkillRemoveCommand_Structure(t *testing.T) {
	root := newSkillTestRoot(t)
	cmd := NewSkillRemoveCommand(root)

	assert.NotNil(t, cmd)
	assert.Equal(t, "remove <skill-id>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Example)
	assert.NotNil(t, cmd.Args) // cobra.ExactArgs(1)
}

func TestRunSkillRemove_GatewayError(t *testing.T) {
	root, _ := newSkillTestRootWithBuf(t)

	err := runSkillRemove(context.Background(), root, "my-skill-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "remove skill failed")
}

func TestRunSkillRemove_DifferentIDs(t *testing.T) {
	ids := []string{"user-skill-1", "community-custom", "my-gpu-tip"}
	for _, id := range ids {
		t.Run(id, func(t *testing.T) {
			root, _ := newSkillTestRootWithBuf(t)
			err := runSkillRemove(context.Background(), root, id)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "remove skill failed")
		})
	}
}

// ---------------------------------------------------------------------------
// NewSkillSearchCommand
// ---------------------------------------------------------------------------

func TestNewSkillSearchCommand_Structure(t *testing.T) {
	root := newSkillTestRoot(t)
	cmd := NewSkillSearchCommand(root)

	assert.NotNil(t, cmd)
	assert.Equal(t, "search <query>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Example)
	assert.NotNil(t, cmd.Args) // cobra.ExactArgs(1)
}

func TestNewSkillSearchCommand_CategoryFlag(t *testing.T) {
	root := newSkillTestRoot(t)
	cmd := NewSkillSearchCommand(root)

	categoryFlag := cmd.Flags().Lookup("category")
	require.NotNil(t, categoryFlag)
	assert.Equal(t, "c", categoryFlag.Shorthand)
	assert.Equal(t, "", categoryFlag.DefValue)
}

func TestRunSkillSearch_GatewayError(t *testing.T) {
	root, _ := newSkillTestRootWithBuf(t)

	err := runSkillSearch(context.Background(), root, "gpu", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "search skills failed")
}

func TestRunSkillSearch_WithCategory(t *testing.T) {
	root, _ := newSkillTestRootWithBuf(t)

	err := runSkillSearch(context.Background(), root, "performance", "optimize")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "search skills failed")
}

func TestRunSkillSearch_DifferentQueries(t *testing.T) {
	tests := []struct {
		query    string
		category string
	}{
		{"gpu", ""},
		{"memory", "optimize"},
		{"install", "setup"},
		{"error", "troubleshoot"},
		{"vllm deployment", ""},
	}

	for _, tc := range tests {
		t.Run(tc.query, func(t *testing.T) {
			root, _ := newSkillTestRootWithBuf(t)
			err := runSkillSearch(context.Background(), root, tc.query, tc.category)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "search skills failed")
		})
	}
}

// ---------------------------------------------------------------------------
// readFileContent helper
// ---------------------------------------------------------------------------

func TestReadFileContent_Success(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "test.md")
	expected := "# Hello\n\nworld"
	require.NoError(t, os.WriteFile(path, []byte(expected), 0600))

	content, err := readFileContent(path)
	require.NoError(t, err)
	assert.Equal(t, expected, content)
}

func TestReadFileContent_NotFound(t *testing.T) {
	_, err := readFileContent("/nonexistent/path/file.md")
	require.Error(t, err)
}

func TestReadFileContent_EmptyFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "empty.md")
	require.NoError(t, os.WriteFile(path, []byte(""), 0600))

	content, err := readFileContent(path)
	require.NoError(t, err)
	assert.Equal(t, "", content)
}
