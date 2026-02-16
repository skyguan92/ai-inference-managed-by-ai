package workflow

import (
	"fmt"
	"regexp"
	"strings"
)

var varRefPattern = regexp.MustCompile(`\$\{([a-zA-Z0-9_.]+)\}`)

type DAGValidator struct{}

func NewDAGValidator() *DAGValidator {
	return &DAGValidator{}
}

func (v *DAGValidator) Validate(def *WorkflowDef) *ValidationResult {
	result := &ValidationResult{
		Valid:  true,
		Errors: []ValidationError{},
	}

	v.validateSteps(def, result)
	v.validateDependencies(def, result)
	v.validateVariableReferences(def, result)
	v.validateOutputReferences(def, result)

	return result
}

func (v *DAGValidator) validateSteps(def *WorkflowDef, result *ValidationResult) {
	if len(def.Steps) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "steps",
			Message: "no steps defined",
		})
		return
	}

	stepIDs := make(map[string]int)

	for i, step := range def.Steps {
		if step.ID == "" {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   fmt.Sprintf("steps[%d].id", i),
				Message: "step ID is empty",
			})
			continue
		}

		if existingIdx, exists := stepIDs[step.ID]; exists {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   fmt.Sprintf("steps[%d].id", i),
				Message: fmt.Sprintf("duplicate step ID '%s' (first defined at steps[%d])", step.ID, existingIdx),
			})
		}
		stepIDs[step.ID] = i

		if step.Type == "" {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   fmt.Sprintf("steps[%d].type", i),
				Message: "step type is empty",
			})
		}

		if step.OnFailure != "" {
			validOnFailure := map[string]bool{"abort": true, "continue": true, "retry": true}
			if !validOnFailure[step.OnFailure] {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Field:   fmt.Sprintf("steps[%d].on_failure", i),
					Message: fmt.Sprintf("invalid on_failure value '%s', must be abort, continue, or retry", step.OnFailure),
				})
			}
		}
	}
}

func (v *DAGValidator) validateDependencies(def *WorkflowDef, result *ValidationResult) {
	stepIDs := make(map[string]bool)
	for _, step := range def.Steps {
		stepIDs[step.ID] = true
	}

	for _, step := range def.Steps {
		for _, dep := range step.DependsOn {
			if !stepIDs[dep] {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Field:   fmt.Sprintf("steps.%s.depends_on", step.ID),
					Message: fmt.Sprintf("dependency '%s' references non-existent step", dep),
				})
			}

			if dep == step.ID {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Field:   fmt.Sprintf("steps.%s.depends_on", step.ID),
					Message: "step cannot depend on itself",
				})
			}
		}
	}

	if err := v.detectCycles(def); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, ValidationError{
			Field:   "steps",
			Message: err.Error(),
		})
	}
}

func (v *DAGValidator) detectCycles(def *WorkflowDef) error {
	visited := make(map[string]bool)
	recStack := make(map[string]bool)
	stepMap := make(map[string]*WorkflowStep)

	for i := range def.Steps {
		stepMap[def.Steps[i].ID] = &def.Steps[i]
	}

	var cyclePath []string

	var dfs func(stepID string) bool
	dfs = func(stepID string) bool {
		visited[stepID] = true
		recStack[stepID] = true
		cyclePath = append(cyclePath, stepID)

		step, exists := stepMap[stepID]
		if !exists {
			return false
		}

		for _, dep := range step.DependsOn {
			if !visited[dep] {
				if dfs(dep) {
					return true
				}
			} else if recStack[dep] {
				cyclePath = append(cyclePath, dep)
				return true
			}
		}

		recStack[stepID] = false
		cyclePath = cyclePath[:len(cyclePath)-1]
		return false
	}

	for _, step := range def.Steps {
		if !visited[step.ID] {
			cyclePath = []string{}
			if dfs(step.ID) {
				if len(cyclePath) >= 2 {
					cycleStart := cyclePath[len(cyclePath)-1]
					var cleanPath []string
					for _, id := range cyclePath {
						if id == cycleStart && len(cleanPath) > 0 {
							break
						}
						cleanPath = append(cleanPath, id)
					}
					return fmt.Errorf("circular dependency detected: %s", strings.Join(cleanPath, " -> "))
				}
				return fmt.Errorf("circular dependency detected involving: %s", step.ID)
			}
		}
	}

	return nil
}

func (v *DAGValidator) validateVariableReferences(def *WorkflowDef, result *ValidationResult) {
	stepIDs := make(map[string]bool)
	for _, step := range def.Steps {
		stepIDs[step.ID] = true
	}

	for _, step := range def.Steps {
		refs := extractVariableRefs(step.Input)
		for _, ref := range refs {
			if err := v.validateVarRef(ref, stepIDs); err != nil {
				result.Valid = false
				result.Errors = append(result.Errors, ValidationError{
					Field:   fmt.Sprintf("steps.%s.input", step.ID),
					Message: err.Error(),
				})
			}
		}
	}
}

func (v *DAGValidator) validateOutputReferences(def *WorkflowDef, result *ValidationResult) {
	stepIDs := make(map[string]bool)
	for _, step := range def.Steps {
		stepIDs[step.ID] = true
	}

	refs := extractVariableRefs(def.Output)
	for _, ref := range refs {
		if err := v.validateVarRef(ref, stepIDs); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   "output",
				Message: err.Error(),
			})
		}
	}
}

func (v *DAGValidator) validateVarRef(ref string, stepIDs map[string]bool) error {
	parts := strings.Split(ref, ".")
	if len(parts) < 2 {
		return fmt.Errorf("invalid variable reference '%s'", ref)
	}

	switch parts[0] {
	case "input":
		return nil
	case "config":
		return nil
	case "steps":
		if len(parts) < 2 {
			return fmt.Errorf("invalid steps reference '%s', expected format: steps.{step_id}.{field}", ref)
		}
		stepID := parts[1]
		if !stepIDs[stepID] {
			return fmt.Errorf("variable reference '%s' references non-existent step '%s'", ref, stepID)
		}
		return nil
	default:
		return fmt.Errorf("unknown variable reference source '%s' in '${%s}'", parts[0], ref)
	}
}

func extractVariableRefs(data any) []string {
	var refs []string

	switch v := data.(type) {
	case string:
		matches := varRefPattern.FindAllStringSubmatch(v, -1)
		for _, match := range matches {
			if len(match) > 1 {
				refs = append(refs, match[1])
			}
		}
	case map[string]any:
		for _, val := range v {
			refs = append(refs, extractVariableRefs(val)...)
		}
	case []any:
		for _, val := range v {
			refs = append(refs, extractVariableRefs(val)...)
		}
	}

	return refs
}

func TopologicalSort(def *WorkflowDef) ([]*WorkflowStep, error) {
	stepMap := make(map[string]*WorkflowStep)
	inDegree := make(map[string]int)

	for i := range def.Steps {
		step := &def.Steps[i]
		stepMap[step.ID] = step
		inDegree[step.ID] = 0
	}

	for _, step := range def.Steps {
		inDegree[step.ID] = len(step.DependsOn)
	}

	var queue []string
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	var sorted []*WorkflowStep
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		sorted = append(sorted, stepMap[id])

		for _, step := range def.Steps {
			for _, dep := range step.DependsOn {
				if dep == id {
					inDegree[step.ID]--
					if inDegree[step.ID] == 0 {
						queue = append(queue, step.ID)
					}
				}
			}
		}
	}

	if len(sorted) != len(def.Steps) {
		return nil, fmt.Errorf("cycle detected in workflow steps")
	}

	return sorted, nil
}
