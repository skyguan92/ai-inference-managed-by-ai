package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewVersionCommand(t *testing.T) {
	root := &RootCommand{
		opts: NewOutputOptions(),
	}

	cmd := NewVersionCommand(root)
	assert.NotNil(t, cmd)
	assert.Equal(t, "version", cmd.Use)
}

func TestPrintVersion_Table(t *testing.T) {
	buf := &bytes.Buffer{}
	opts := &OutputOptions{
		Format: OutputTable,
		Writer: buf,
	}

	printVersion(opts)

	output := buf.String()
	assert.Contains(t, output, "AIMA version")
}

func TestPrintVersion_JSON(t *testing.T) {
	buf := &bytes.Buffer{}
	opts := &OutputOptions{
		Format: OutputJSON,
		Writer: buf,
	}

	printVersion(opts)

	output := buf.String()
	assert.Contains(t, output, `"version"`)
	assert.Contains(t, output, `"buildDate"`)
	assert.Contains(t, output, `"gitCommit"`)
}

func TestPrintVersion_YAML(t *testing.T) {
	buf := &bytes.Buffer{}
	opts := &OutputOptions{
		Format: OutputYAML,
		Writer: buf,
	}

	printVersion(opts)

	output := buf.String()
	assert.Contains(t, output, "version:")
	assert.Contains(t, output, "buildDate:")
	assert.Contains(t, output, "gitCommit:")
}

func TestSetVersion(t *testing.T) {
	SetVersion("1.0.0", "2024-01-01", "abc123")

	assert.Equal(t, "1.0.0", GetVersion())
	assert.Equal(t, "2024-01-01", GetBuildDate())
	assert.Equal(t, "abc123", GetGitCommit())
}
