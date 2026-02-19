package cli

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
)

func makeWorkflowRoot(t *testing.T) *RootCommand {
	t.Helper()
	registry := unit.NewRegistry()
	// Register model.list so workflow execution can succeed
	registry.RegisterQuery(&testServiceQuery{
		name: "model.list",
		execute: func(ctx context.Context, input any) (any, error) {
			return map[string]any{"models": []map[string]any{}, "total": 0}, nil
		},
	})
	gw := gateway.NewGateway(registry)
	buf := &bytes.Buffer{}
	return &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputJSON, Writer: buf},
	}
}

func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "workflow-*.yaml")
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(f.Name()) })
	_, err = f.WriteString(content)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	return f.Name()
}

func TestNewWorkflowCommand(t *testing.T) {
	root := makeWorkflowRoot(t)
	cmd := NewWorkflowCommand(root)
	assert.NotNil(t, cmd)
	assert.Equal(t, "workflow", cmd.Use)
	assert.Contains(t, cmd.Aliases, "wf")

	subCommands := cmd.Commands()
	assert.Len(t, subCommands, 3)

	names := make([]string, len(subCommands))
	for i, c := range subCommands {
		names[i] = c.Name()
	}
	assert.Contains(t, names, "list")
	assert.Contains(t, names, "run")
	assert.Contains(t, names, "validate")
}

func TestWorkflowListCommand_Structure(t *testing.T) {
	root := makeWorkflowRoot(t)
	cmd := newWorkflowListCommand(root)
	assert.NotNil(t, cmd)
	assert.Equal(t, "list", cmd.Use)
	assert.Contains(t, cmd.Aliases, "ls")
}

func TestWorkflowRunCommand_Flags(t *testing.T) {
	root := makeWorkflowRoot(t)
	cmd := newWorkflowRunCommand(root)
	assert.NotNil(t, cmd)
	assert.Equal(t, "run <workflow>", cmd.Use)

	inputFlag := cmd.Flags().Lookup("input")
	assert.NotNil(t, inputFlag)
	assert.Equal(t, "i", inputFlag.Shorthand)
}

func TestWorkflowValidateCommand_Structure(t *testing.T) {
	root := makeWorkflowRoot(t)
	cmd := newWorkflowValidateCommand(root)
	assert.NotNil(t, cmd)
	assert.Equal(t, "validate <file>", cmd.Use)
}

func TestRunWorkflowList(t *testing.T) {
	root := makeWorkflowRoot(t)
	err := runWorkflowList(context.Background(), root)
	require.NoError(t, err)
}

func TestRunWorkflowRun_WithModelList(t *testing.T) {
	root := makeWorkflowRoot(t)
	err := runWorkflowRun(context.Background(), root, "my-workflow", "")
	require.NoError(t, err)
}

func TestRunWorkflowRun_WithInputJSON(t *testing.T) {
	root := makeWorkflowRoot(t)
	// Current impl ignores input JSON but shouldn't error
	err := runWorkflowRun(context.Background(), root, "my-workflow", `{"model":"llama3"}`)
	require.NoError(t, err)
}

func TestRunWorkflowRun_RegistryExecutesModelList(t *testing.T) {
	registry := unit.NewRegistry()
	executed := false
	registry.RegisterQuery(&testServiceQuery{
		name: "model.list",
		execute: func(ctx context.Context, input any) (any, error) {
			executed = true
			return map[string]any{"models": []map[string]any{}, "total": 0}, nil
		},
	})
	gw := gateway.NewGateway(registry)
	buf := &bytes.Buffer{}
	root := &RootCommand{
		gateway:  gw,
		registry: registry,
		opts:     &OutputOptions{Format: OutputJSON, Writer: buf},
	}

	err := runWorkflowRun(context.Background(), root, "test", "")
	require.NoError(t, err)
	assert.True(t, executed, "expected model.list to be executed")
}

func TestRunWorkflowValidate_NonExistentFile(t *testing.T) {
	root := makeWorkflowRoot(t)
	// Non-existent file: error printed, nil returned
	err := runWorkflowValidate(context.Background(), root, "/nonexistent/workflow.yaml")
	assert.NoError(t, err)
}

func TestRunWorkflowValidate_ValidWorkflow(t *testing.T) {
	tmpfile := writeTempYAML(t, `
name: test-workflow
steps:
  - id: step1
    type: model.list
    input: {}
`)
	root := makeWorkflowRoot(t)
	err := runWorkflowValidate(context.Background(), root, tmpfile)
	require.NoError(t, err)
}

func TestRunWorkflowValidate_EmptyNameAndSteps(t *testing.T) {
	tmpfile := writeTempYAML(t, `
name: ""
steps: []
`)
	root := makeWorkflowRoot(t)
	err := runWorkflowValidate(context.Background(), root, tmpfile)
	require.NoError(t, err)
}

func TestRunWorkflowValidate_NonExistentDependency(t *testing.T) {
	tmpfile := writeTempYAML(t, `
name: test-workflow
steps:
  - id: step1
    type: model.list
    input: {}
    depends_on:
      - nonexistent-step
`)
	root := makeWorkflowRoot(t)
	err := runWorkflowValidate(context.Background(), root, tmpfile)
	require.NoError(t, err)
}

func TestRunWorkflowValidate_DuplicateStepID(t *testing.T) {
	tmpfile := writeTempYAML(t, `
name: test-workflow
steps:
  - id: step1
    type: model.list
    input: {}
  - id: step1
    type: model.list
    input: {}
`)
	root := makeWorkflowRoot(t)
	err := runWorkflowValidate(context.Background(), root, tmpfile)
	require.NoError(t, err)
}

func TestRunWorkflowValidate_EmptyStepID(t *testing.T) {
	tmpfile := writeTempYAML(t, `
name: test-workflow
steps:
  - id: ""
    type: model.list
    input: {}
`)
	root := makeWorkflowRoot(t)
	err := runWorkflowValidate(context.Background(), root, tmpfile)
	require.NoError(t, err)
}

func TestRunWorkflowValidate_EmptyStepType(t *testing.T) {
	tmpfile := writeTempYAML(t, `
name: test-workflow
steps:
  - id: step1
    type: ""
    input: {}
`)
	root := makeWorkflowRoot(t)
	err := runWorkflowValidate(context.Background(), root, tmpfile)
	require.NoError(t, err)
}

func TestRunWorkflowValidate_MultipleStepsWithDependency(t *testing.T) {
	tmpfile := writeTempYAML(t, `
name: test-workflow
steps:
  - id: step1
    type: model.list
    input: {}
  - id: step2
    type: model.list
    input: {}
    depends_on:
      - step1
`)
	root := makeWorkflowRoot(t)
	err := runWorkflowValidate(context.Background(), root, tmpfile)
	require.NoError(t, err)
}
