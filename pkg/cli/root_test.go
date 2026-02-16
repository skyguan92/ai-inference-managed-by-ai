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

func TestNewRootCommand(t *testing.T) {
	root := NewRootCommand()
	assert.NotNil(t, root)
	assert.NotNil(t, root.Command())
	assert.NotNil(t, root.OutputOptions())
}

func TestRootCommand_Commands(t *testing.T) {
	root := NewRootCommand()
	cmd := root.Command()

	subCommands := cmd.Commands()
	assert.GreaterOrEqual(t, len(subCommands), 5)
}

func TestRootCommand_Accessors(t *testing.T) {
	registry := unit.NewRegistry()
	gw := gateway.NewGateway(registry)
	cfg := config.Default()
	opts := NewOutputOptions()

	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		cfg:      cfg,
		opts:     opts,
	}

	assert.Equal(t, gw, root.Gateway())
	assert.Equal(t, registry, root.Registry())
	assert.Equal(t, cfg, root.Config())
	assert.Equal(t, opts, root.OutputOptions())
}

func TestRootCommand_SetOutputWriter(t *testing.T) {
	root := NewRootCommand()
	buf := &bytes.Buffer{}
	root.SetOutputWriter(buf)

	root.opts.Writer = buf
	assert.Equal(t, buf, root.OutputOptions().Writer)
}

func TestGetVersion(t *testing.T) {
	version := GetVersion()
	assert.NotEmpty(t, version)
}

func TestGetBuildDate(t *testing.T) {
	date := GetBuildDate()
	assert.NotEmpty(t, date)
}

func TestGetGitCommit(t *testing.T) {
	commit := GetGitCommit()
	assert.NotEmpty(t, commit)
}

func TestRootCommand_PersistentPreRunE(t *testing.T) {
	root := NewRootCommand()
	cmd := root.Command()

	err := root.persistentPreRunE(cmd, []string{})
	require.NoError(t, err)

	assert.NotNil(t, root.Config())
	assert.NotNil(t, root.Gateway())
	assert.NotNil(t, root.Registry())
}

func TestRootCommand_Execute(t *testing.T) {
	root := NewRootCommand()
	cmd := root.Command()

	cmd.SetArgs([]string{"--help"})
	err := cmd.Execute()
	assert.NoError(t, err)
}

func TestRootCommand_ExecuteVersion(t *testing.T) {
	root := NewRootCommand()
	cmd := root.Command()

	buf := &bytes.Buffer{}
	root.opts.Writer = buf
	root.opts.Format = OutputJSON

	cmd.SetArgs([]string{"version"})
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "version")
}

func TestExecute_WithCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	root := NewRootCommand()
	cmd := root.Command()
	cmd.SetArgs([]string{"--help"})

	err := cmd.ExecuteContext(ctx)
	assert.NoError(t, err)
}
