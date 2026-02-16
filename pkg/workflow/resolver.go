package workflow

import (
	"fmt"
	"regexp"
	"strings"
)

var varPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

type VariableResolver struct{}

func NewVariableResolver() *VariableResolver {
	return &VariableResolver{}
}

func (r *VariableResolver) Resolve(value any, ctx *ExecutionContext) (any, error) {
	switch v := value.(type) {
	case string:
		return r.resolveString(v, ctx)
	case map[string]any:
		return r.resolveMap(v, ctx)
	case []any:
		return r.resolveArray(v, ctx)
	default:
		return value, nil
	}
}

func (r *VariableResolver) resolveString(s string, ctx *ExecutionContext) (any, error) {
	matches := varPattern.FindAllStringSubmatchIndex(s, -1)

	if len(matches) == 0 {
		return s, nil
	}

	if len(matches) == 1 && matches[0][0] == 0 && matches[0][1] == len(s) {
		ref := s[matches[0][2]:matches[0][3]]
		return r.resolveReference(ref, ctx)
	}

	var result strings.Builder
	lastIdx := 0
	for _, match := range matches {
		result.WriteString(s[lastIdx:match[0]])
		ref := s[match[2]:match[3]]
		resolved, err := r.resolveReference(ref, ctx)
		if err != nil {
			return nil, err
		}
		result.WriteString(fmt.Sprintf("%v", resolved))
		lastIdx = match[1]
	}
	result.WriteString(s[lastIdx:])

	return result.String(), nil
}

func (r *VariableResolver) resolveMap(m map[string]any, ctx *ExecutionContext) (map[string]any, error) {
	result := make(map[string]any, len(m))
	for key, value := range m {
		resolved, err := r.Resolve(value, ctx)
		if err != nil {
			return nil, fmt.Errorf("resolving key '%s': %w", key, err)
		}
		result[key] = resolved
	}
	return result, nil
}

func (r *VariableResolver) resolveArray(arr []any, ctx *ExecutionContext) ([]any, error) {
	result := make([]any, len(arr))
	for i, value := range arr {
		resolved, err := r.Resolve(value, ctx)
		if err != nil {
			return nil, fmt.Errorf("resolving array index %d: %w", i, err)
		}
		result[i] = resolved
	}
	return result, nil
}

func (r *VariableResolver) resolveReference(ref string, ctx *ExecutionContext) (any, error) {
	parts := strings.Split(ref, ".")
	if len(parts) < 1 {
		return nil, fmt.Errorf("invalid variable reference: ${%s}", ref)
	}

	switch parts[0] {
	case "input":
		return r.resolveFromMap(ctx.Input, parts[1:], "input")
	case "config":
		return r.resolveFromMap(ctx.Config, parts[1:], "config")
	case "steps":
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid steps reference: ${%s}, expected steps.{step_id}.{field}", ref)
		}
		stepID := parts[1]
		stepOutput, exists := ctx.Steps[stepID]
		if !exists {
			return nil, fmt.Errorf("step '%s' not found in context", stepID)
		}
		if len(parts) < 3 {
			return stepOutput, nil
		}
		return r.resolveFromMap(stepOutput, parts[2:], "steps."+stepID)
	default:
		return nil, fmt.Errorf("unknown variable source: %s", parts[0])
	}
}

func (r *VariableResolver) resolveFromMap(data map[string]any, path []string, source string) (any, error) {
	if len(path) == 0 {
		return data, nil
	}

	current := any(data)
	for i, key := range path {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("cannot access key '%s' on non-object value in ${%s.%s}", key, source, strings.Join(path[:i+1], "."))
		}

		val, exists := m[key]
		if !exists {
			return nil, fmt.Errorf("key '%s' not found in ${%s.%s}", key, source, strings.Join(path[:i+1], "."))
		}
		current = val
	}

	return current, nil
}

func (r *VariableResolver) ResolveStepInput(step *WorkflowStep, ctx *ExecutionContext) (map[string]any, error) {
	resolved, err := r.Resolve(step.Input, ctx)
	if err != nil {
		return nil, fmt.Errorf("resolving step '%s' input: %w", step.ID, err)
	}
	return resolved.(map[string]any), nil
}

func (r *VariableResolver) ResolveOutput(outputDef map[string]any, ctx *ExecutionContext) (map[string]any, error) {
	resolved, err := r.Resolve(outputDef, ctx)
	if err != nil {
		return nil, err
	}
	if m, ok := resolved.(map[string]any); ok {
		return m, nil
	}
	return nil, fmt.Errorf("output resolution did not produce a map")
}
