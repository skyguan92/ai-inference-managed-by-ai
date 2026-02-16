package workflow

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	ErrEmptyInput        = fmt.Errorf("empty input")
	ErrInvalidFormat     = fmt.Errorf("invalid format")
	ErrWorkflowNameEmpty = fmt.Errorf("workflow name is empty")
	ErrNoSteps           = fmt.Errorf("no steps defined")
)

func ParseYAML(data []byte) (*WorkflowDef, error) {
	if len(data) == 0 {
		return nil, ErrEmptyInput
	}

	var def WorkflowDef
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidFormat, err)
	}

	if err := normalizeWorkflowDef(&def); err != nil {
		return nil, err
	}

	return &def, nil
}

func ParseJSON(data []byte) (*WorkflowDef, error) {
	if len(data) == 0 {
		return nil, ErrEmptyInput
	}

	var def WorkflowDef
	if err := json.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidFormat, err)
	}

	if err := normalizeWorkflowDef(&def); err != nil {
		return nil, err
	}

	return &def, nil
}

func Parse(data []byte, format string) (*WorkflowDef, error) {
	switch strings.ToLower(format) {
	case "yaml", "yml":
		return ParseYAML(data)
	case "json":
		return ParseJSON(data)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

func ParseFile(path string) (*WorkflowDef, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	format := "yaml"
	if ext == ".json" {
		format = "json"
	}

	return Parse(data, format)
}

func ParseReader(r io.Reader, format string) (*WorkflowDef, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("failed to read: %w", err)
	}

	return Parse(data, format)
}

func normalizeWorkflowDef(def *WorkflowDef) error {
	if def.Name == "" {
		return ErrWorkflowNameEmpty
	}

	if len(def.Steps) == 0 {
		return ErrNoSteps
	}

	if def.Config == nil {
		def.Config = make(map[string]any)
	}

	if def.Output == nil {
		def.Output = make(map[string]any)
	}

	for i := range def.Steps {
		step := &def.Steps[i]

		if step.Input == nil {
			step.Input = make(map[string]any)
		}

		if step.DependsOn == nil {
			step.DependsOn = []string{}
		}

		if step.OnFailure == "" {
			step.OnFailure = "abort"
		}

		if step.Retry != nil {
			if step.Retry.MaxAttempts < 1 {
				step.Retry.MaxAttempts = 1
			}
			if step.Retry.DelaySeconds < 0 {
				step.Retry.DelaySeconds = 0
			}
		}
	}

	return nil
}

func (def *WorkflowDef) ToYAML() ([]byte, error) {
	return yaml.Marshal(def)
}

func (def *WorkflowDef) ToJSON() ([]byte, error) {
	return json.MarshalIndent(def, "", "  ")
}

func (def *WorkflowDef) GetStepByID(id string) *WorkflowStep {
	for i := range def.Steps {
		if def.Steps[i].ID == id {
			return &def.Steps[i]
		}
	}
	return nil
}

func (def *WorkflowDef) StepIDs() []string {
	ids := make([]string, len(def.Steps))
	for i, step := range def.Steps {
		ids[i] = step.ID
	}
	return ids
}
