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

func TestNewEngineCommand(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     NewOutputOptions(),
	}

	cmd := NewEngineCommand(root)
	assert.NotNil(t, cmd)
	assert.Equal(t, "engine", cmd.Use)

	subCommands := cmd.Commands()
	assert.Len(t, subCommands, 3)

	commandNames := make([]string, len(subCommands))
	for i, c := range subCommands {
		commandNames[i] = c.Name()
	}
	assert.Contains(t, commandNames, "start")
	assert.Contains(t, commandNames, "stop")
	assert.Contains(t, commandNames, "list")
}

func TestEngineStartCommand_Flags(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     NewOutputOptions(),
	}

	cmd := NewEngineStartCommand(root)
	assert.NotNil(t, cmd)
	assert.Equal(t, "start <engine>", cmd.Use)

	configFlag := cmd.Flags().Lookup("config")
	assert.NotNil(t, configFlag)
	assert.Equal(t, "c", configFlag.Shorthand)
}

func TestEngineStopCommand_Flags(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     NewOutputOptions(),
	}

	cmd := NewEngineStopCommand(root)
	assert.NotNil(t, cmd)
	assert.Equal(t, "stop <engine>", cmd.Use)

	forceFlag := cmd.Flags().Lookup("force")
	assert.NotNil(t, forceFlag)
	assert.Equal(t, "f", forceFlag.Shorthand)

	timeoutFlag := cmd.Flags().Lookup("timeout")
	assert.NotNil(t, timeoutFlag)
}

func TestEngineListCommand_Flags(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     NewOutputOptions(),
	}

	cmd := NewEngineListCommand(root)

	typeFlag := cmd.Flags().Lookup("type")
	assert.NotNil(t, typeFlag)

	statusFlag := cmd.Flags().Lookup("status")
	assert.NotNil(t, statusFlag)
}

func TestEngineListCommand_Execute(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputJSON, Writer: buf},
	}

	err := runEngineList(context.Background(), root, "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRunEngineStart(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputJSON, Writer: buf},
	}

	err := runEngineStart(context.Background(), root, "ollama", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRunEngineStart_WithConfig(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputJSON, Writer: buf},
	}

	err := runEngineStart(context.Background(), root, "ollama", "/path/to/config.json")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRunEngineStop(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputJSON, Writer: buf},
	}

	err := runEngineStop(context.Background(), root, "ollama", false, 30)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRunEngineStop_Force(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputJSON, Writer: buf},
	}

	err := runEngineStop(context.Background(), root, "ollama", true, 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRunEngineList_WithFilters(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputJSON, Writer: buf},
	}

	err := runEngineList(context.Background(), root, "ollama", "running")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
