package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func newCatalogTestRoot(t *testing.T) *RootCommand {
	t.Helper()
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)
	return &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     NewOutputOptions(),
	}
}

func newCatalogTestRootWithBuf(t *testing.T) (*RootCommand, *bytes.Buffer) {
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
// NewCatalogCommand — structure
// ---------------------------------------------------------------------------

func TestNewCatalogCommand(t *testing.T) {
	root := newCatalogTestRoot(t)
	cmd := NewCatalogCommand(root)

	assert.NotNil(t, cmd)
	assert.Equal(t, "catalog", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
}

func TestNewCatalogCommand_Subcommands(t *testing.T) {
	root := newCatalogTestRoot(t)
	cmd := NewCatalogCommand(root)

	subCmds := cmd.Commands()
	assert.Len(t, subCmds, 5)

	names := make([]string, len(subCmds))
	for i, c := range subCmds {
		names[i] = c.Name()
	}
	assert.Contains(t, names, "list")
	assert.Contains(t, names, "get")
	assert.Contains(t, names, "match")
	assert.Contains(t, names, "apply")
	assert.Contains(t, names, "validate")
}

// ---------------------------------------------------------------------------
// NewCatalogListCommand
// ---------------------------------------------------------------------------

func TestNewCatalogListCommand_Structure(t *testing.T) {
	root := newCatalogTestRoot(t)
	cmd := NewCatalogListCommand(root)

	assert.NotNil(t, cmd)
	assert.Equal(t, "list", cmd.Use)
	assert.Contains(t, cmd.Aliases, "ls")
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Example)
}

func TestNewCatalogListCommand_Flags(t *testing.T) {
	root := newCatalogTestRoot(t)
	cmd := NewCatalogListCommand(root)

	gpuFlag := cmd.Flags().Lookup("gpu-vendor")
	require.NotNil(t, gpuFlag)
	assert.Equal(t, "", gpuFlag.DefValue)

	verifiedFlag := cmd.Flags().Lookup("verified")
	require.NotNil(t, verifiedFlag)
	assert.Equal(t, "false", verifiedFlag.DefValue)

	tagsFlag := cmd.Flags().Lookup("tags")
	require.NotNil(t, tagsFlag)
}

func TestRunCatalogList_GatewayError(t *testing.T) {
	root, _ := newCatalogTestRootWithBuf(t)

	err := runCatalogList(context.Background(), root, "", false, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "list recipes failed")
}

func TestRunCatalogList_WithFilters(t *testing.T) {
	tests := []struct {
		name         string
		gpuVendor    string
		verifiedOnly bool
		tags         []string
	}{
		{"no filters", "", false, nil},
		{"gpu vendor nvidia", "nvidia", false, nil},
		{"verified only", "", true, nil},
		{"with tags", "", false, []string{"llm", "chat"}},
		{"all filters", "nvidia", true, []string{"llm"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root, _ := newCatalogTestRootWithBuf(t)
			err := runCatalogList(context.Background(), root, tc.gpuVendor, tc.verifiedOnly, tc.tags)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "list recipes failed")
		})
	}
}

// ---------------------------------------------------------------------------
// NewCatalogGetCommand
// ---------------------------------------------------------------------------

func TestNewCatalogGetCommand_Structure(t *testing.T) {
	root := newCatalogTestRoot(t)
	cmd := NewCatalogGetCommand(root)

	assert.NotNil(t, cmd)
	assert.Equal(t, "get <recipe-id>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Example)
	assert.NotNil(t, cmd.Args) // cobra.ExactArgs(1)
}

func TestRunCatalogGet_GatewayError(t *testing.T) {
	root, _ := newCatalogTestRootWithBuf(t)

	err := runCatalogGet(context.Background(), root, "my-recipe-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "get recipe failed")
}

func TestRunCatalogGet_DifferentIDs(t *testing.T) {
	ids := []string{"recipe-001", "llama3-nvidia-rtx4090", "mistral-cpu-only"}
	for _, id := range ids {
		t.Run(id, func(t *testing.T) {
			root, _ := newCatalogTestRootWithBuf(t)
			err := runCatalogGet(context.Background(), root, id)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "get recipe failed")
		})
	}
}

// ---------------------------------------------------------------------------
// NewCatalogMatchCommand
// ---------------------------------------------------------------------------

func TestNewCatalogMatchCommand_Structure(t *testing.T) {
	root := newCatalogTestRoot(t)
	cmd := NewCatalogMatchCommand(root)

	assert.NotNil(t, cmd)
	assert.Equal(t, "match", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotEmpty(t, cmd.Example)
}

func TestNewCatalogMatchCommand_Flags(t *testing.T) {
	root := newCatalogTestRoot(t)
	cmd := NewCatalogMatchCommand(root)

	gpuVendorFlag := cmd.Flags().Lookup("gpu-vendor")
	require.NotNil(t, gpuVendorFlag)
	assert.Equal(t, "", gpuVendorFlag.DefValue)

	gpuModelFlag := cmd.Flags().Lookup("gpu-model")
	require.NotNil(t, gpuModelFlag)

	vramFlag := cmd.Flags().Lookup("vram")
	require.NotNil(t, vramFlag)
	assert.Equal(t, "0", vramFlag.DefValue)

	osFlag := cmd.Flags().Lookup("os")
	require.NotNil(t, osFlag)

	limitFlag := cmd.Flags().Lookup("limit")
	require.NotNil(t, limitFlag)
	assert.Equal(t, "5", limitFlag.DefValue)
}

func TestRunCatalogMatch_GatewayError(t *testing.T) {
	root, _ := newCatalogTestRootWithBuf(t)

	err := runCatalogMatch(context.Background(), root, "", "", 0, "", 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "match recipes failed")
}

func TestRunCatalogMatch_WithHardwareFilters(t *testing.T) {
	tests := []struct {
		name      string
		gpuVendor string
		gpuModel  string
		vramGB    int
		osName    string
		limit     int
	}{
		{"nvidia 24gb", "nvidia", "", 24, "", 5},
		{"specific gpu model", "nvidia", "RTX 4090", 24, "linux", 3},
		{"amd gpu", "amd", "", 16, "", 5},
		{"apple silicon", "apple", "M2", 0, "darwin", 5},
		{"no hardware specified", "", "", 0, "", 10},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root, _ := newCatalogTestRootWithBuf(t)
			err := runCatalogMatch(context.Background(), root, tc.gpuVendor, tc.gpuModel, tc.vramGB, tc.osName, tc.limit)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "match recipes failed")
		})
	}
}

// ---------------------------------------------------------------------------
// NewCatalogApplyCommand
// ---------------------------------------------------------------------------

func TestNewCatalogApplyCommand_Structure(t *testing.T) {
	root := newCatalogTestRoot(t)
	cmd := NewCatalogApplyCommand(root)

	assert.NotNil(t, cmd)
	assert.Equal(t, "apply <recipe-id>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotEmpty(t, cmd.Example)
	assert.NotNil(t, cmd.Args) // cobra.ExactArgs(1)
}

func TestRunCatalogApply_GatewayError(t *testing.T) {
	root, _ := newCatalogTestRootWithBuf(t)

	err := runCatalogApply(context.Background(), root, "my-recipe-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "apply recipe failed")
}

// ---------------------------------------------------------------------------
// NewCatalogValidateCommand
// ---------------------------------------------------------------------------

func TestNewCatalogValidateCommand_Structure(t *testing.T) {
	root := newCatalogTestRoot(t)
	cmd := NewCatalogValidateCommand(root)

	assert.NotNil(t, cmd)
	assert.Equal(t, "validate", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Example)
}

func TestNewCatalogValidateCommand_FileFlag(t *testing.T) {
	root := newCatalogTestRoot(t)
	cmd := NewCatalogValidateCommand(root)

	fileFlag := cmd.Flags().Lookup("file")
	require.NotNil(t, fileFlag)
	assert.Equal(t, "f", fileFlag.Shorthand)
	assert.Equal(t, "", fileFlag.DefValue)

	// --file is required
	annotations := fileFlag.Annotations
	require.NotNil(t, annotations)
	_, required := annotations["cobra_annotation_bash_completion_one_required_flag"]
	assert.True(t, required)
}

func TestRunCatalogValidate_GatewayError(t *testing.T) {
	root, _ := newCatalogTestRootWithBuf(t)

	err := runCatalogValidate(context.Background(), root, "/path/to/recipe.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "validate recipe failed")
}

func TestRunCatalogValidate_DifferentFilePaths(t *testing.T) {
	paths := []string{
		"/home/user/recipe.yaml",
		"./relative/recipe.yaml",
		"recipe.yaml",
	}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			root, _ := newCatalogTestRootWithBuf(t)
			err := runCatalogValidate(context.Background(), root, path)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "validate recipe failed")
		})
	}
}
