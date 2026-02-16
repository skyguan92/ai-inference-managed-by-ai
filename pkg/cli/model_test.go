package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/config"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestNewModelCommand(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     NewOutputOptions(),
	}

	cmd := NewModelCommand(root)
	assert.NotNil(t, cmd)
	assert.Equal(t, "model", cmd.Use)

	subCommands := cmd.Commands()
	assert.Len(t, subCommands, 5)

	commandNames := make([]string, len(subCommands))
	for i, c := range subCommands {
		commandNames[i] = c.Name()
	}
	assert.Contains(t, commandNames, "pull")
	assert.Contains(t, commandNames, "list")
	assert.Contains(t, commandNames, "get")
	assert.Contains(t, commandNames, "delete")
	assert.Contains(t, commandNames, "create")
}

func TestModelPullCommand_Flags(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     NewOutputOptions(),
	}

	cmd := NewModelPullCommand(root)

	sourceFlag := cmd.Flags().Lookup("source")
	assert.NotNil(t, sourceFlag)
	assert.Equal(t, "s", sourceFlag.Shorthand)

	tagFlag := cmd.Flags().Lookup("tag")
	assert.NotNil(t, tagFlag)
}

func TestModelPullCommand_MissingRepo(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)
	cfg := config.Default()

	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputTable, Writer: buf},
		cfg:      cfg,
	}

	err := runModelPull(context.Background(), root, "", "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "repo is required")
}

func TestModelListCommand(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     NewOutputOptions(),
	}

	cmd := NewModelListCommand(root)
	assert.NotNil(t, cmd)

	typeFlag := cmd.Flags().Lookup("type")
	assert.NotNil(t, typeFlag)

	statusFlag := cmd.Flags().Lookup("status")
	assert.NotNil(t, statusFlag)

	limitFlag := cmd.Flags().Lookup("limit")
	assert.NotNil(t, limitFlag)
}

func TestModelGetCommand(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     NewOutputOptions(),
	}

	cmd := NewModelGetCommand(root)
	assert.NotNil(t, cmd)
	assert.Equal(t, "get <model_id>", cmd.Use)
	assert.NotNil(t, cmd.Args)
}

func TestModelDeleteCommand(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     NewOutputOptions(),
	}

	cmd := NewModelDeleteCommand(root)
	assert.NotNil(t, cmd)

	forceFlag := cmd.Flags().Lookup("force")
	assert.NotNil(t, forceFlag)
	assert.Equal(t, "f", forceFlag.Shorthand)
}

func TestModelCreateCommand_Flags(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     NewOutputOptions(),
	}

	cmd := NewModelCreateCommand(root)

	typeFlag := cmd.Flags().Lookup("type")
	assert.NotNil(t, typeFlag)

	formatFlag := cmd.Flags().Lookup("format")
	assert.NotNil(t, formatFlag)

	pathFlag := cmd.Flags().Lookup("path")
	assert.NotNil(t, pathFlag)
}

func TestModelCreateCommand_MissingName(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputTable, Writer: buf},
	}

	err := runModelCreate(context.Background(), root, "", "llm", "", "gguf", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestModelGet_Execute(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputJSON, Writer: buf},
	}

	err := runModelGet(context.Background(), root, "test-model")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestModelDelete_Execute(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputJSON, Writer: buf},
	}

	err := runModelDelete(context.Background(), root, "test-model", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestModelList_Execute(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputJSON, Writer: buf},
	}

	err := runModelList(context.Background(), root, "", "", "", 100)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestModelPull_Execute(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputJSON, Writer: buf},
		cfg:      config.Default(),
	}

	err := runModelPull(context.Background(), root, "ollama", "llama3", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
