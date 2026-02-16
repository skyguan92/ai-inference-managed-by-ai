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

func TestNewInferenceCommand(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     NewOutputOptions(),
	}

	cmd := NewInferenceCommand(root)
	assert.NotNil(t, cmd)
	assert.Equal(t, "inference", cmd.Use)

	subCommands := cmd.Commands()
	assert.Len(t, subCommands, 2)
}

func TestInferenceChatCommand_Flags(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     NewOutputOptions(),
	}

	cmd := NewInferenceChatCommand(root)

	modelFlag := cmd.Flags().Lookup("model")
	assert.NotNil(t, modelFlag)
	assert.Equal(t, "m", modelFlag.Shorthand)

	messageFlag := cmd.Flags().Lookup("message")
	assert.NotNil(t, messageFlag)

	temperatureFlag := cmd.Flags().Lookup("temperature")
	assert.NotNil(t, temperatureFlag)

	maxTokensFlag := cmd.Flags().Lookup("max-tokens")
	assert.NotNil(t, maxTokensFlag)
}

func TestInferenceChatCommand_MissingModel(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputTable, Writer: buf},
	}

	err := runInferenceChat(context.Background(), root, "", "Hello", 0.7, 0, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "model is required")
}

func TestInferenceChatCommand_MissingMessage(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputTable, Writer: buf},
	}

	err := runInferenceChat(context.Background(), root, "llama3.2", "", 0.7, 0, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "message is required")
}

func TestInferenceChatCommand_StreamNotSupported(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputTable, Writer: buf},
	}

	err := runInferenceChat(context.Background(), root, "llama3.2", "Hello", 0.7, 0, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "streaming is not yet supported")
}

func TestInferenceEmbedCommand_Flags(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     NewOutputOptions(),
	}

	cmd := NewInferenceEmbedCommand(root)

	modelFlag := cmd.Flags().Lookup("model")
	assert.NotNil(t, modelFlag)

	inputFlag := cmd.Flags().Lookup("input")
	assert.NotNil(t, inputFlag)
}

func TestInferenceEmbedCommand_MissingModel(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputTable, Writer: buf},
	}

	err := runInferenceEmbed(context.Background(), root, "", "Hello")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "model is required")
}

func TestInferenceEmbedCommand_MissingInput(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputTable, Writer: buf},
	}

	err := runInferenceEmbed(context.Background(), root, "nomic-embed-text", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "input is required")
}
