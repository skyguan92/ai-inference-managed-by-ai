package integration

import (
	"context"
	"testing"

	registrypkg "github.com/jguan/ai-inference-managed-by-ai/pkg/registry"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/unit"
	"github.com/jguan/ai-inference-managed-by-ai/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestRegistry(t *testing.T) *unit.Registry {
	registry := unit.NewRegistry()
	err := registrypkg.RegisterAll(registry)
	require.NoError(t, err)
	return registry
}

func createTestWorkflowEngine(t *testing.T) (*unit.Registry, *workflow.WorkflowEngine) {
	registry := createTestRegistry(t)
	store := workflow.NewInMemoryWorkflowStore()
	engine := workflow.NewWorkflowEngine(registry, store, nil)
	return registry, engine
}

func TestWorkflowIntegrationSimple(t *testing.T) {
	_, engine := createTestWorkflowEngine(t)

	def := &workflow.WorkflowDef{
		Name: "simple_test",
		Steps: []workflow.WorkflowStep{
			{
				ID:    "step1",
				Type:  "model.list",
				Input: map[string]any{},
			},
		},
	}

	result, err := engine.Execute(context.Background(), def, nil)
	require.NoError(t, err)
	assert.Equal(t, workflow.ExecutionStatusCompleted, result.Status)
	assert.NotNil(t, result.StepResults["step1"])
	assert.Equal(t, workflow.ExecutionStatusCompleted, result.StepResults["step1"].Status)
}

func TestWorkflowIntegrationWithDependencies(t *testing.T) {
	_, engine := createTestWorkflowEngine(t)

	def := &workflow.WorkflowDef{
		Name: "dependency_test",
		Steps: []workflow.WorkflowStep{
			{
				ID:    "list_models",
				Type:  "model.list",
				Input: map[string]any{},
			},
			{
				ID:        "list_engines",
				Type:      "engine.list",
				Input:     map[string]any{},
				DependsOn: []string{"list_models"},
			},
		},
	}

	result, err := engine.Execute(context.Background(), def, nil)
	require.NoError(t, err)
	assert.Equal(t, workflow.ExecutionStatusCompleted, result.Status)
	assert.NotNil(t, result.StepResults["list_models"])
	assert.NotNil(t, result.StepResults["list_engines"])
}

func TestWorkflowIntegrationMultipleDomains(t *testing.T) {
	_, engine := createTestWorkflowEngine(t)

	def := &workflow.WorkflowDef{
		Name: "multi_domain_test",
		Steps: []workflow.WorkflowStep{
			{
				ID:    "list_models",
				Type:  "model.list",
				Input: map[string]any{},
			},
			{
				ID:    "list_engines",
				Type:  "engine.list",
				Input: map[string]any{},
			},
			{
				ID:    "service_list",
				Type:  "service.list",
				Input: map[string]any{},
			},
			{
				ID:    "app_list",
				Type:  "app.list",
				Input: map[string]any{},
			},
		},
	}

	result, err := engine.Execute(context.Background(), def, nil)
	require.NoError(t, err)
	assert.Equal(t, workflow.ExecutionStatusCompleted, result.Status)
	assert.Len(t, result.StepResults, 4)
}

func TestWorkflowIntegrationWithInput(t *testing.T) {
	_, engine := createTestWorkflowEngine(t)

	def := &workflow.WorkflowDef{
		Name: "input_test",
		Steps: []workflow.WorkflowStep{
			{
				ID:   "list_models",
				Type: "model.list",
				Input: map[string]any{
					"filter": "${input.filter}",
				},
			},
		},
	}

	input := map[string]any{
		"filter": "",
	}

	result, err := engine.Execute(context.Background(), def, input)
	require.NoError(t, err)
	assert.Equal(t, workflow.ExecutionStatusCompleted, result.Status)
}

func TestWorkflowIntegrationWithOutput(t *testing.T) {
	_, engine := createTestWorkflowEngine(t)

	def := &workflow.WorkflowDef{
		Name: "output_test",
		Steps: []workflow.WorkflowStep{
			{
				ID:    "list_models",
				Type:  "model.list",
				Input: map[string]any{},
			},
		},
		Output: map[string]any{
			"models": "${steps.list_models}",
		},
	}

	result, err := engine.Execute(context.Background(), def, nil)
	require.NoError(t, err)
	assert.Equal(t, workflow.ExecutionStatusCompleted, result.Status)
	assert.NotNil(t, result.Output)
}

func TestWorkflowIntegrationValidation(t *testing.T) {
	_, engine := createTestWorkflowEngine(t)

	t.Run("empty steps", func(t *testing.T) {
		def := &workflow.WorkflowDef{
			Name:  "empty_steps",
			Steps: []workflow.WorkflowStep{},
		}

		_, err := engine.Execute(context.Background(), def, nil)
		assert.Error(t, err)
	})

	t.Run("circular dependency", func(t *testing.T) {
		def := &workflow.WorkflowDef{
			Name: "circular",
			Steps: []workflow.WorkflowStep{
				{ID: "a", Type: "model.list", Input: map[string]any{}, DependsOn: []string{"b"}},
				{ID: "b", Type: "model.list", Input: map[string]any{}, DependsOn: []string{"a"}},
			},
		}

		_, err := engine.Execute(context.Background(), def, nil)
		assert.Error(t, err)
	})

	t.Run("missing step id", func(t *testing.T) {
		def := &workflow.WorkflowDef{
			Name: "missing_id",
			Steps: []workflow.WorkflowStep{
				{ID: "", Type: "model.list", Input: map[string]any{}},
			},
		}

		_, err := engine.Execute(context.Background(), def, nil)
		assert.Error(t, err)
	})
}

func TestWorkflowIntegrationPersistence(t *testing.T) {
	_, engine := createTestWorkflowEngine(t)

	def := &workflow.WorkflowDef{
		Name: "persistence_test",
		Steps: []workflow.WorkflowStep{
			{
				ID:    "step1",
				Type:  "model.list",
				Input: map[string]any{},
			},
		},
	}

	result, err := engine.Execute(context.Background(), def, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, result.RunID)

	savedResult, err := engine.GetExecution(context.Background(), result.RunID)
	require.NoError(t, err)
	assert.NotNil(t, savedResult)
	assert.Equal(t, result.RunID, savedResult.RunID)
	assert.Equal(t, result.Status, savedResult.Status)
}

func TestWorkflowIntegrationRegisterAndGet(t *testing.T) {
	_, engine := createTestWorkflowEngine(t)

	def := &workflow.WorkflowDef{
		Name: "registered_workflow",
		Steps: []workflow.WorkflowStep{
			{
				ID:    "step1",
				Type:  "model.list",
				Input: map[string]any{},
			},
		},
	}

	err := engine.RegisterWorkflow(context.Background(), def)
	require.NoError(t, err)

	retrieved, err := engine.GetWorkflow(context.Background(), "registered_workflow")
	require.NoError(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "registered_workflow", retrieved.Name)
}

func TestWorkflowIntegrationListWorkflows(t *testing.T) {
	_, engine := createTestWorkflowEngine(t)

	def1 := &workflow.WorkflowDef{
		Name: "workflow1",
		Steps: []workflow.WorkflowStep{
			{ID: "step1", Type: "model.list", Input: map[string]any{}},
		},
	}
	def2 := &workflow.WorkflowDef{
		Name: "workflow2",
		Steps: []workflow.WorkflowStep{
			{ID: "step1", Type: "engine.list", Input: map[string]any{}},
		},
	}

	err := engine.RegisterWorkflow(context.Background(), def1)
	require.NoError(t, err)
	err = engine.RegisterWorkflow(context.Background(), def2)
	require.NoError(t, err)

	list, err := engine.ListWorkflows(context.Background())
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list), 2)
}

func TestWorkflowIntegrationDeleteWorkflow(t *testing.T) {
	_, engine := createTestWorkflowEngine(t)

	def := &workflow.WorkflowDef{
		Name: "to_delete",
		Steps: []workflow.WorkflowStep{
			{ID: "step1", Type: "model.list", Input: map[string]any{}},
		},
	}

	err := engine.RegisterWorkflow(context.Background(), def)
	require.NoError(t, err)

	err = engine.DeleteWorkflow(context.Background(), "to_delete")
	require.NoError(t, err)

	retrieved, err := engine.GetWorkflow(context.Background(), "to_delete")
	require.NoError(t, err)
	assert.Nil(t, retrieved)
}

func TestWorkflowIntegrationAllDomains(t *testing.T) {
	_, engine := createTestWorkflowEngine(t)

	domainTests := []struct {
		name  string
		steps []workflow.WorkflowStep
	}{
		{
			name: "model_domain",
			steps: []workflow.WorkflowStep{
				{ID: "list", Type: "model.list", Input: map[string]any{}},
			},
		},
		{
			name: "engine_domain",
			steps: []workflow.WorkflowStep{
				{ID: "list", Type: "engine.list", Input: map[string]any{}},
			},
		},
		{
			name: "service_domain",
			steps: []workflow.WorkflowStep{
				{ID: "list", Type: "service.list", Input: map[string]any{}},
			},
		},
		{
			name: "app_domain",
			steps: []workflow.WorkflowStep{
				{ID: "list", Type: "app.list", Input: map[string]any{}},
			},
		},
		{
			name: "pipeline_domain",
			steps: []workflow.WorkflowStep{
				{ID: "list", Type: "pipeline.list", Input: map[string]any{}},
			},
		},
		{
			name: "alert_domain",
			steps: []workflow.WorkflowStep{
				{ID: "list", Type: "alert.list_rules", Input: map[string]any{}},
			},
		},
		{
			name: "remote_domain",
			steps: []workflow.WorkflowStep{
				{ID: "status", Type: "remote.status", Input: map[string]any{}},
			},
		},
	}

	for _, tc := range domainTests {
		t.Run(tc.name, func(t *testing.T) {
			def := &workflow.WorkflowDef{
				Name:  tc.name + "_test",
				Steps: tc.steps,
			}

			result, err := engine.Execute(context.Background(), def, nil)
			require.NoError(t, err)
			assert.Equal(t, workflow.ExecutionStatusCompleted, result.Status)
		})
	}
}

func TestWorkflowIntegrationComplexWorkflow(t *testing.T) {
	_, engine := createTestWorkflowEngine(t)

	def := &workflow.WorkflowDef{
		Name:        "complex_workflow",
		Description: "A complex workflow testing multiple features",
		Steps: []workflow.WorkflowStep{
			{
				ID:    "list_models",
				Type:  "model.list",
				Input: map[string]any{},
			},
			{
				ID:    "list_engines",
				Type:  "engine.list",
				Input: map[string]any{},
			},
			{
				ID:        "list_services",
				Type:      "service.list",
				Input:     map[string]any{},
				DependsOn: []string{"list_models", "list_engines"},
			},
			{
				ID:        "list_apps",
				Type:      "app.list",
				Input:     map[string]any{},
				DependsOn: []string{"list_services"},
			},
		},
		Output: map[string]any{
			"models":   "${steps.list_models}",
			"engines":  "${steps.list_engines}",
			"services": "${steps.list_services}",
			"apps":     "${steps.list_apps}",
		},
	}

	result, err := engine.Execute(context.Background(), def, nil)
	require.NoError(t, err)
	assert.Equal(t, workflow.ExecutionStatusCompleted, result.Status)
	assert.Len(t, result.StepResults, 4)
	assert.NotNil(t, result.Output)
}

func TestWorkflowIntegrationExecutionHistory(t *testing.T) {
	_, engine := createTestWorkflowEngine(t)

	def := &workflow.WorkflowDef{
		Name: "history_test",
		Steps: []workflow.WorkflowStep{
			{ID: "step1", Type: "model.list", Input: map[string]any{}},
		},
	}

	for i := 0; i < 3; i++ {
		result, err := engine.Execute(context.Background(), def, nil)
		require.NoError(t, err)
		assert.Equal(t, workflow.ExecutionStatusCompleted, result.Status)
	}

	executions, err := engine.ListExecutions(context.Background(), "history_test", 10)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(executions), 3)
}
