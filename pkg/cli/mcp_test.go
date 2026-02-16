package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestNewMCPCommand(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     NewOutputOptions(),
	}

	cmd := NewMCPCommand(root)
	assert.NotNil(t, cmd)
	assert.Equal(t, "mcp", cmd.Use)

	subCommands := cmd.Commands()
	assert.Len(t, subCommands, 2)
}

func TestNewMCPServeCommand(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     NewOutputOptions(),
	}

	cmd := NewMCPServeCommand(root)
	assert.NotNil(t, cmd)
	assert.Equal(t, "serve", cmd.Use)
	assert.NotNil(t, cmd.RunE)
}

func TestNewMCPSSECommand(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     NewOutputOptions(),
	}

	cmd := NewMCPSSECommand(root)
	assert.NotNil(t, cmd)
	assert.Equal(t, "sse", cmd.Use)
	assert.NotNil(t, cmd.RunE)

	addrFlag := cmd.Flags().Lookup("addr")
	assert.NotNil(t, addrFlag)
	assert.Equal(t, defaultMCPAddr, addrFlag.DefValue)
}

func TestMCPSSECommand_DefaultAddr(t *testing.T) {
	assert.Equal(t, "127.0.0.1:9091", defaultMCPAddr)
}
