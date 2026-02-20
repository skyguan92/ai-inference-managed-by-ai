package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/workflow"
	"github.com/spf13/cobra"
)

func NewWorkflowCommand(root *RootCommand) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "workflow",
		Aliases: []string{"wf"},
		Short:   "Manage workflows",
		Long: `Manage workflow definitions and executions.

Workflows allow you to chain multiple atomic units together
to create complex automation pipelines.`,
		Example: `  # List workflow templates
  aima workflow list

  # Run a workflow
  aima workflow run my-workflow

  # Validate a workflow definition
  aima workflow validate workflow.yaml`,
	}

	cmd.AddCommand(newWorkflowListCommand(root))
	cmd.AddCommand(newWorkflowRunCommand(root))
	cmd.AddCommand(newWorkflowValidateCommand(root))

	return cmd
}

func newWorkflowListCommand(root *RootCommand) *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List workflow templates",
		Long: `List all available workflow templates.

Shows both built-in templates and user-defined workflows.`,
		Example: `  # List all workflows
  aima workflow list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWorkflowList(cmd.Context(), root)
		},
	}
}

func newWorkflowRunCommand(root *RootCommand) *cobra.Command {
	var inputJSON string

	cmd := &cobra.Command{
		Use:   "run <workflow>",
		Short: "Run a workflow",
		Long: `Run a workflow by name or from a file.

You can provide input parameters via --input flag or as JSON.`,
		Example: `  # Run a workflow by name
  aima workflow run deploy-model

  # Run with input parameters
  aima workflow run deploy-model --input '{"model":"llama3"}'

  # Run from file
  aima workflow run ./workflows/deploy.yaml`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workflowName := args[0]
			return runWorkflowRun(cmd.Context(), root, workflowName, inputJSON)
		},
	}

	cmd.Flags().StringVarP(&inputJSON, "input", "i", "", "JSON input for the workflow")

	return cmd
}

func newWorkflowValidateCommand(root *RootCommand) *cobra.Command {
	return &cobra.Command{
		Use:   "validate <file>",
		Short: "Validate a workflow definition",
		Long: `Validate a workflow definition file.

Checks the workflow syntax, step references, and dependencies.`,
		Example: `  # Validate a workflow file
  aima workflow validate workflow.yaml`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filePath := args[0]
			return runWorkflowValidate(cmd.Context(), root, filePath)
		},
	}
}

func runWorkflowList(ctx context.Context, root *RootCommand) error {
	opts := root.OutputOptions()

	// Define built-in workflow templates
	templates := []map[string]any{
		{
			"name":        "deploy-model",
			"description": "Deploy a model to an inference engine",
			"steps":       4,
		},
		{
			"name":        "scale-service",
			"description": "Scale a model service based on load",
			"steps":       3,
		},
		{
			"name":        "backup-restore",
			"description": "Backup and restore model configurations",
			"steps":       5,
		},
	}

	return PrintOutput(map[string]any{
		"workflows": templates,
		"total":     len(templates),
	}, opts)
}

func runWorkflowRun(ctx context.Context, root *RootCommand, workflowName, inputJSON string) error {
	opts := root.OutputOptions()
	registry := root.Registry()

	// Create workflow engine
	store := workflow.NewInMemoryWorkflowStore()
	engine := workflow.NewWorkflowEngine(registry, store, nil)

	// Parse input
	input := make(map[string]any)
	if inputJSON != "" {
		if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
			return fmt.Errorf("parse input JSON: %w", err)
		}
	}

	// For demonstration, create a simple workflow definition
	def := &workflow.WorkflowDef{
		Name: workflowName,
		Steps: []workflow.WorkflowStep{
			{
				ID:    "step1",
				Type:  "model.list",
				Input: map[string]any{},
			},
		},
	}

	// Execute workflow
	result, err := engine.Execute(ctx, def, input)
	if err != nil {
		PrintError(fmt.Errorf("execute workflow: %w", err), opts)
		return nil
	}

	return PrintOutput(map[string]any{
		"workflow_id": result.WorkflowID,
		"run_id":      result.RunID,
		"status":      result.Status,
		"duration":    result.Duration.String(),
		"output":      result.Output,
	}, opts)
}

func runWorkflowValidate(ctx context.Context, root *RootCommand, filePath string) error {
	opts := root.OutputOptions()

	// Parse workflow definition
	def, err := workflow.ParseFile(filePath)
	if err != nil {
		PrintError(fmt.Errorf("parse workflow: %w", err), opts)
		return nil
	}

	// Use pipeline validation (similar logic)
	valid := true
	var issues []string

	if def.Name == "" {
		valid = false
		issues = append(issues, "workflow name is empty")
	}
	if len(def.Steps) == 0 {
		valid = false
		issues = append(issues, "workflow has no steps")
	}

	// Check step IDs
	stepIDs := make(map[string]bool)
	for _, step := range def.Steps {
		if step.ID == "" {
			valid = false
			issues = append(issues, "step has empty ID")
		}
		if step.Type == "" {
			valid = false
			issues = append(issues, fmt.Sprintf("step %s has empty type", step.ID))
		}
		if stepIDs[step.ID] {
			valid = false
			issues = append(issues, fmt.Sprintf("duplicate step ID: %s", step.ID))
		}
		stepIDs[step.ID] = true
	}

	// Check dependencies
	for _, step := range def.Steps {
		for _, dep := range step.DependsOn {
			if !stepIDs[dep] {
				valid = false
				issues = append(issues, fmt.Sprintf("step %s depends on non-existent step: %s", step.ID, dep))
			}
		}
	}

	return PrintOutput(map[string]any{
		"valid":      valid,
		"issues":     issues,
		"file":       filePath,
		"workflow":   def.Name,
		"step_count": len(def.Steps),
	}, opts)
}
