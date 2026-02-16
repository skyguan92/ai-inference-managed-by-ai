package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDAGValidator_Validate(t *testing.T) {
	validator := NewDAGValidator()

	tests := []struct {
		name        string
		workflow    *WorkflowDef
		wantValid   bool
		wantErrCnt  int
		errContains string
	}{
		{
			name: "valid simple workflow",
			workflow: &WorkflowDef{
				Name: "valid",
				Steps: []WorkflowStep{
					{ID: "s1", Type: "inference.chat", Input: map[string]any{}},
				},
			},
			wantValid: true,
		},
		{
			name: "valid workflow with dependencies",
			workflow: &WorkflowDef{
				Name: "valid_deps",
				Steps: []WorkflowStep{
					{ID: "s1", Type: "type1", Input: map[string]any{}},
					{ID: "s2", Type: "type2", Input: map[string]any{}, DependsOn: []string{"s1"}},
					{ID: "s3", Type: "type3", Input: map[string]any{}, DependsOn: []string{"s1", "s2"}},
				},
			},
			wantValid: true,
		},
		{
			name: "empty step ID",
			workflow: &WorkflowDef{
				Name: "empty_id",
				Steps: []WorkflowStep{
					{ID: "", Type: "type1", Input: map[string]any{}},
				},
			},
			wantValid:  false,
			wantErrCnt: 1,
		},
		{
			name: "duplicate step IDs",
			workflow: &WorkflowDef{
				Name: "dup_id",
				Steps: []WorkflowStep{
					{ID: "s1", Type: "type1", Input: map[string]any{}},
					{ID: "s1", Type: "type2", Input: map[string]any{}},
				},
			},
			wantValid:  false,
			wantErrCnt: 1,
		},
		{
			name: "empty step type",
			workflow: &WorkflowDef{
				Name: "empty_type",
				Steps: []WorkflowStep{
					{ID: "s1", Type: "", Input: map[string]any{}},
				},
			},
			wantValid:  false,
			wantErrCnt: 1,
		},
		{
			name: "invalid on_failure value",
			workflow: &WorkflowDef{
				Name: "bad_on_failure",
				Steps: []WorkflowStep{
					{ID: "s1", Type: "type1", Input: map[string]any{}, OnFailure: "invalid"},
				},
			},
			wantValid:  false,
			wantErrCnt: 1,
		},
		{
			name: "non-existent dependency",
			workflow: &WorkflowDef{
				Name: "bad_dep",
				Steps: []WorkflowStep{
					{ID: "s1", Type: "type1", Input: map[string]any{}, DependsOn: []string{"nonexistent"}},
				},
			},
			wantValid:  false,
			wantErrCnt: 1,
		},
		{
			name: "self dependency",
			workflow: &WorkflowDef{
				Name: "self_dep",
				Steps: []WorkflowStep{
					{ID: "s1", Type: "type1", Input: map[string]any{}, DependsOn: []string{"s1"}},
				},
			},
			wantValid:  false,
			wantErrCnt: 1,
		},
		{
			name: "circular dependency",
			workflow: &WorkflowDef{
				Name: "circular",
				Steps: []WorkflowStep{
					{ID: "s1", Type: "type1", Input: map[string]any{}, DependsOn: []string{"s2"}},
					{ID: "s2", Type: "type2", Input: map[string]any{}, DependsOn: []string{"s1"}},
				},
			},
			wantValid:  false,
			wantErrCnt: 1,
		},
		{
			name: "valid variable references",
			workflow: &WorkflowDef{
				Name:   "valid_refs",
				Config: map[string]any{"model": "llama3.2"},
				Steps: []WorkflowStep{
					{ID: "s1", Type: "type1", Input: map[string]any{"m": "${config.model}"}},
					{ID: "s2", Type: "type2", Input: map[string]any{"data": "${steps.s1.output}"}, DependsOn: []string{"s1"}},
				},
			},
			wantValid: true,
		},
		{
			name: "invalid steps variable reference",
			workflow: &WorkflowDef{
				Name: "bad_step_ref",
				Steps: []WorkflowStep{
					{ID: "s1", Type: "type1", Input: map[string]any{"data": "${steps.nonexistent.output}"}},
				},
			},
			wantValid:  false,
			wantErrCnt: 1,
		},
		{
			name: "invalid variable source",
			workflow: &WorkflowDef{
				Name: "bad_var_source",
				Steps: []WorkflowStep{
					{ID: "s1", Type: "type1", Input: map[string]any{"data": "${unknown.field}"}},
				},
			},
			wantValid:  false,
			wantErrCnt: 1,
		},
		{
			name: "valid output references",
			workflow: &WorkflowDef{
				Name: "valid_output",
				Steps: []WorkflowStep{
					{ID: "s1", Type: "type1", Input: map[string]any{}},
				},
				Output: map[string]any{"result": "${steps.s1.output}"},
			},
			wantValid: true,
		},
		{
			name: "invalid output reference",
			workflow: &WorkflowDef{
				Name: "bad_output",
				Steps: []WorkflowStep{
					{ID: "s1", Type: "type1", Input: map[string]any{}},
				},
				Output: map[string]any{"result": "${steps.nonexistent.output}"},
			},
			wantValid:  false,
			wantErrCnt: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.Validate(tt.workflow)
			assert.Equal(t, tt.wantValid, result.Valid)
			if !tt.wantValid {
				assert.GreaterOrEqual(t, len(result.Errors), tt.wantErrCnt)
				if tt.errContains != "" {
					found := false
					for _, e := range result.Errors {
						if contains(e.Message, tt.errContains) {
							found = true
							break
						}
					}
					assert.True(t, found, "expected error containing '%s'", tt.errContains)
				}
			}
		})
	}
}

func TestDAGValidator_CycleDetection(t *testing.T) {
	validator := NewDAGValidator()

	tests := []struct {
		name      string
		steps     []WorkflowStep
		wantCycle bool
	}{
		{
			name: "no cycle",
			steps: []WorkflowStep{
				{ID: "a", Type: "t"},
				{ID: "b", Type: "t", DependsOn: []string{"a"}},
				{ID: "c", Type: "t", DependsOn: []string{"b"}},
			},
			wantCycle: false,
		},
		{
			name: "simple cycle",
			steps: []WorkflowStep{
				{ID: "a", Type: "t", DependsOn: []string{"b"}},
				{ID: "b", Type: "t", DependsOn: []string{"a"}},
			},
			wantCycle: true,
		},
		{
			name: "three node cycle",
			steps: []WorkflowStep{
				{ID: "a", Type: "t", DependsOn: []string{"c"}},
				{ID: "b", Type: "t", DependsOn: []string{"a"}},
				{ID: "c", Type: "t", DependsOn: []string{"b"}},
			},
			wantCycle: true,
		},
		{
			name: "diamond shape no cycle",
			steps: []WorkflowStep{
				{ID: "a", Type: "t"},
				{ID: "b", Type: "t", DependsOn: []string{"a"}},
				{ID: "c", Type: "t", DependsOn: []string{"a"}},
				{ID: "d", Type: "t", DependsOn: []string{"b", "c"}},
			},
			wantCycle: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := &WorkflowDef{Name: "test", Steps: tt.steps}
			result := validator.Validate(def)

			hasCycle := false
			for _, e := range result.Errors {
				if contains(e.Message, "circular") || contains(e.Message, "cycle") {
					hasCycle = true
					break
				}
			}

			assert.Equal(t, tt.wantCycle, hasCycle)
		})
	}
}

func TestTopologicalSort(t *testing.T) {
	tests := []struct {
		name      string
		steps     []WorkflowStep
		wantOrder []string
		wantError bool
	}{
		{
			name: "simple linear",
			steps: []WorkflowStep{
				{ID: "a", Type: "t"},
				{ID: "b", Type: "t", DependsOn: []string{"a"}},
				{ID: "c", Type: "t", DependsOn: []string{"b"}},
			},
			wantOrder: []string{"a", "b", "c"},
		},
		{
			name: "diamond shape",
			steps: []WorkflowStep{
				{ID: "d", Type: "t", DependsOn: []string{"b", "c"}},
				{ID: "b", Type: "t", DependsOn: []string{"a"}},
				{ID: "c", Type: "t", DependsOn: []string{"a"}},
				{ID: "a", Type: "t"},
			},
			wantOrder: []string{"a", "b", "c", "d"},
		},
		{
			name: "parallel steps",
			steps: []WorkflowStep{
				{ID: "a", Type: "t"},
				{ID: "b", Type: "t"},
				{ID: "c", Type: "t"},
			},
		},
		{
			name: "cycle detected",
			steps: []WorkflowStep{
				{ID: "a", Type: "t", DependsOn: []string{"b"}},
				{ID: "b", Type: "t", DependsOn: []string{"a"}},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			def := &WorkflowDef{Name: "test", Steps: tt.steps}
			sorted, err := TopologicalSort(def)

			if tt.wantError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Len(t, sorted, len(tt.steps))

			if tt.wantOrder != nil {
				for i, wantID := range tt.wantOrder {
					assert.Equal(t, wantID, sorted[i].ID)
				}
			}

			executed := make(map[string]bool)
			for _, step := range sorted {
				for _, dep := range step.DependsOn {
					assert.True(t, executed[dep], "dependency %s should be executed before %s", dep, step.ID)
				}
				executed[step.ID] = true
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
