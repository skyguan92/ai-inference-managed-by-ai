package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func TestNewExecCommand(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     NewOutputOptions(),
	}

	cmd := NewExecCommand(root)
	assert.NotNil(t, cmd)
	assert.Contains(t, cmd.Use, "exec")
	assert.NotNil(t, cmd.RunE)
}

func TestExecCommand_WithJSONInput(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts: &OutputOptions{
			Format: OutputJSON,
			Writer: buf,
		},
	}

	cmd := NewExecCommand(root)
	cmd.SetArgs([]string{"model.list", "--input", "{}"})

	err := cmd.Execute()
	assert.Error(t, err)
}

func TestExtractFlags(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("name", "", "name")
	cmd.Flags().Int("count", 0, "count")
	cmd.Flags().Bool("verbose", false, "verbose")

	cmd.ParseFlags([]string{"--name", "test", "--count", "5", "--verbose"})

	flags := cmd.Flags()
	result := extractFlags(flags)

	assert.Equal(t, "test", result["name"])
	assert.Equal(t, "5", result["count"])
	assert.Equal(t, "true", result["verbose"])
}

func TestRunExec_MissingUnit(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts: &OutputOptions{
			Format: OutputJSON,
			Writer: buf,
		},
	}

	cmd := &cobra.Command{}
	cmd.ParseFlags([]string{})

	err := runExec(context.Background(), root, "nonexistent.unit", "", cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRunExec_InvalidJSON(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)

	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts: &OutputOptions{
			Format: OutputJSON,
			Writer: buf,
		},
	}

	cmd := &cobra.Command{}
	cmd.ParseFlags([]string{})

	err := runExec(context.Background(), root, "model.list", "{invalid json", cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse input JSON")
}
