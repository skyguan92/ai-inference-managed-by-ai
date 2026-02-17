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

func TestNewDeviceCommand(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     NewOutputOptions(),
	}

	cmd := NewDeviceCommand(root)
	assert.NotNil(t, cmd)
	assert.Equal(t, "device", cmd.Use)

	subCommands := cmd.Commands()
	assert.Len(t, subCommands, 3)

	commandNames := make([]string, len(subCommands))
	for i, c := range subCommands {
		commandNames[i] = c.Use
	}
	assert.Contains(t, commandNames, "detect")
	assert.Contains(t, commandNames, "info")
	assert.Contains(t, commandNames, "metrics")
}

func TestDeviceDetectCommand(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     NewOutputOptions(),
	}

	cmd := NewDeviceDetectCommand(root)
	assert.NotNil(t, cmd)
	assert.Equal(t, "detect", cmd.Use)
}

func TestDeviceDetectCommand_Execute(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputJSON, Writer: buf},
	}

	err := runDeviceDetect(context.Background(), root)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDeviceInfoCommand_Flags(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     NewOutputOptions(),
	}

	cmd := NewDeviceInfoCommand(root)

	idFlag := cmd.Flags().Lookup("id")
	assert.NotNil(t, idFlag)
	assert.Equal(t, "i", idFlag.Shorthand)
}

func TestDeviceMetricsCommand_Flags(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     NewOutputOptions(),
	}

	cmd := NewDeviceMetricsCommand(root)

	idFlag := cmd.Flags().Lookup("id")
	assert.NotNil(t, idFlag)
	assert.Equal(t, "i", idFlag.Shorthand)

	historyFlag := cmd.Flags().Lookup("history")
	assert.NotNil(t, historyFlag)
}

func TestRunDeviceInfo(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputJSON, Writer: buf},
	}

	err := runDeviceInfo(context.Background(), root, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRunDeviceInfo_WithDeviceID(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputJSON, Writer: buf},
	}

	err := runDeviceInfo(context.Background(), root, "gpu0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRunDeviceMetrics(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputJSON, Writer: buf},
	}

	err := runDeviceMetrics(context.Background(), root, "", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRunDeviceMetrics_WithDeviceID(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputJSON, Writer: buf},
	}

	err := runDeviceMetrics(context.Background(), root, "gpu0", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
