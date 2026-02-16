package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseYAML(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid workflow",
			input: `
name: test_workflow
description: A test workflow
config:
  model: llama3.2
steps:
  - id: step1
    type: inference.chat
    input:
      model: "${config.model}"
      message: "Hello"
`,
			wantErr: false,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
			errMsg:  "empty input",
		},
		{
			name:    "missing name",
			input:   "steps:\n  - id: s1\n    type: test",
			wantErr: true,
			errMsg:  "name is empty",
		},
		{
			name:    "missing steps",
			input:   "name: test",
			wantErr: true,
			errMsg:  "no steps defined",
		},
		{
			name:    "invalid yaml",
			input:   "name: [invalid",
			wantErr: true,
			errMsg:  "invalid format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def, err := ParseYAML([]byte(tt.input))
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, def)
			} else {
				require.NoError(t, err)
				require.NotNil(t, def)
				assert.NotEmpty(t, def.Name)
				assert.NotEmpty(t, def.Steps)
			}
		})
	}
}

func TestParseJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid workflow",
			input: `{
				"name": "test_workflow",
				"description": "A test workflow",
				"config": {"model": "llama3.2"},
				"steps": [
					{
						"id": "step1",
						"type": "inference.chat",
						"input": {"model": "llama3.2"}
					}
				]
			}`,
			wantErr: false,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
			errMsg:  "empty input",
		},
		{
			name:    "missing name",
			input:   `{"steps": [{"id": "s1", "type": "test"}]}`,
			wantErr: true,
			errMsg:  "name is empty",
		},
		{
			name:    "missing steps",
			input:   `{"name": "test"}`,
			wantErr: true,
			errMsg:  "no steps defined",
		},
		{
			name:    "invalid json",
			input:   `{invalid`,
			wantErr: true,
			errMsg:  "invalid format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def, err := ParseJSON([]byte(tt.input))
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				assert.Nil(t, def)
			} else {
				require.NoError(t, err)
				require.NotNil(t, def)
				assert.NotEmpty(t, def.Name)
				assert.NotEmpty(t, def.Steps)
			}
		})
	}
}

func TestParseAutoFormat(t *testing.T) {
	yamlInput := `
name: yaml_test
steps:
  - id: s1
    type: test
`
	jsonInput := `{"name": "json_test", "steps": [{"id": "s1", "type": "test"}]}`

	def, err := Parse([]byte(yamlInput), "yaml")
	require.NoError(t, err)
	assert.Equal(t, "yaml_test", def.Name)

	def, err = Parse([]byte(jsonInput), "json")
	require.NoError(t, err)
	assert.Equal(t, "json_test", def.Name)

	_, err = Parse([]byte("test"), "unknown")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format")
}

func TestNormalizeWorkflowDef(t *testing.T) {
	def := &WorkflowDef{
		Name:  "test",
		Steps: []WorkflowStep{{ID: "s1", Type: "test"}},
	}

	err := normalizeWorkflowDef(def)
	require.NoError(t, err)

	assert.NotNil(t, def.Config)
	assert.NotNil(t, def.Output)
	assert.NotNil(t, def.Steps[0].Input)
	assert.NotNil(t, def.Steps[0].DependsOn)
	assert.Equal(t, "abort", def.Steps[0].OnFailure)
}

func TestWorkflowDefMethods(t *testing.T) {
	def := &WorkflowDef{
		Name: "test",
		Steps: []WorkflowStep{
			{ID: "step1", Type: "type1"},
			{ID: "step2", Type: "type2"},
		},
	}

	step := def.GetStepByID("step1")
	require.NotNil(t, step)
	assert.Equal(t, "step1", step.ID)

	assert.Nil(t, def.GetStepByID("nonexistent"))

	ids := def.StepIDs()
	assert.ElementsMatch(t, []string{"step1", "step2"}, ids)
}

func TestWorkflowDefSerialization(t *testing.T) {
	original := &WorkflowDef{
		Name:        "test",
		Description: "Test workflow",
		Config:      map[string]any{"model": "llama3.2"},
		Steps: []WorkflowStep{
			{
				ID:        "step1",
				Type:      "inference.chat",
				Input:     map[string]any{"message": "Hello"},
				DependsOn: []string{},
				OnFailure: "abort",
			},
		},
		Output: map[string]any{"result": "${steps.step1.response}"},
	}

	yamlBytes, err := original.ToYAML()
	require.NoError(t, err)
	assert.Contains(t, string(yamlBytes), "name: test")

	jsonBytes, err := original.ToJSON()
	require.NoError(t, err)
	assert.Contains(t, string(jsonBytes), `"name": "test"`)

	fromYAML, err := ParseYAML(yamlBytes)
	require.NoError(t, err)
	assert.Equal(t, original.Name, fromYAML.Name)

	fromJSON, err := ParseJSON(jsonBytes)
	require.NoError(t, err)
	assert.Equal(t, original.Name, fromJSON.Name)
}
