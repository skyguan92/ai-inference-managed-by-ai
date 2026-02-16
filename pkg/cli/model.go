package cli

import (
	"context"
	"fmt"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/spf13/cobra"
)

func NewModelCommand(root *RootCommand) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "model",
		Short: "Model management commands",
		Long: `Manage AI models in the AIMA infrastructure.

This includes pulling models from registries, listing available models,
and managing model lifecycle.`,
	}

	cmd.AddCommand(NewModelPullCommand(root))
	cmd.AddCommand(NewModelListCommand(root))
	cmd.AddCommand(NewModelGetCommand(root))
	cmd.AddCommand(NewModelDeleteCommand(root))
	cmd.AddCommand(NewModelCreateCommand(root))

	return cmd
}

func NewModelPullCommand(root *RootCommand) *cobra.Command {
	var (
		source string
		repo   string
		tag    string
	)

	cmd := &cobra.Command{
		Use:   "pull [repo]",
		Short: "Pull a model from a registry",
		Long: `Pull a model from a model registry (Ollama, HuggingFace, ModelScope).

If source is not specified, the default from config is used.`,
		Example: `  # Pull from Ollama (default)
  aima model pull llama3.2

  # Pull with explicit source
  aima model pull llama3.2 --source ollama

  # Pull with tag
  aima model pull llama3.2 --tag latest

  # Pull from HuggingFace
  aima model pull meta-llama/Llama-3-8B --source huggingface`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				repo = args[0]
			}
			return runModelPull(cmd.Context(), root, source, repo, tag)
		},
	}

	cmd.Flags().StringVarP(&source, "source", "s", "", "Model source (ollama, huggingface, modelscope)")
	cmd.Flags().StringVar(&tag, "tag", "", "Model tag or version")

	return cmd
}

func runModelPull(ctx context.Context, root *RootCommand, source, repo, tag string) error {
	if repo == "" {
		return fmt.Errorf("repo is required")
	}

	if source == "" {
		source = root.Config().Model.DefaultSource
	}

	gw := root.Gateway()
	opts := root.OutputOptions()

	req := &gateway.Request{
		Type: gateway.TypeCommand,
		Unit: "model.pull",
		Input: map[string]any{
			"source": source,
			"repo":   repo,
			"tag":    tag,
		},
	}

	resp := gw.Handle(ctx, req)

	if !resp.Success {
		PrintError(fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message), opts)
		return fmt.Errorf("pull model failed: %s", resp.Error.Message)
	}

	return PrintOutput(resp.Data, opts)
}

func NewModelListCommand(root *RootCommand) *cobra.Command {
	var (
		modelType string
		status    string
		format    string
		limit     int
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List models",
		Long:    `List all models in the registry.`,
		Example: `  # List all models
  aima model list

  # List only LLM models
  aima model list --type llm

  # List with JSON output
  aima model list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runModelList(cmd.Context(), root, modelType, status, format, limit)
		},
	}

	cmd.Flags().StringVarP(&modelType, "type", "t", "", "Filter by model type (llm, vlm, asr, tts, embedding)")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status (pending, ready, error)")
	cmd.Flags().StringVar(&format, "format", "", "Filter by format (gguf, safetensors, onnx)")
	cmd.Flags().IntVar(&limit, "limit", 100, "Maximum number of results")

	return cmd
}

func runModelList(ctx context.Context, root *RootCommand, modelType, status, format string, limit int) error {
	gw := root.Gateway()
	opts := root.OutputOptions()

	input := make(map[string]any)
	if modelType != "" {
		input["type"] = modelType
	}
	if status != "" {
		input["status"] = status
	}
	if format != "" {
		input["format"] = format
	}
	if limit > 0 {
		input["limit"] = limit
	}

	req := &gateway.Request{
		Type:  gateway.TypeQuery,
		Unit:  "model.list",
		Input: input,
	}

	resp := gw.Handle(ctx, req)

	if !resp.Success {
		PrintError(fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message), opts)
		return fmt.Errorf("list models failed: %s", resp.Error.Message)
	}

	return PrintOutput(resp.Data, opts)
}

func NewModelGetCommand(root *RootCommand) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <model_id>",
		Short: "Get model details",
		Long:  `Get detailed information about a specific model.`,
		Args:  cobra.ExactArgs(1),
		Example: `  # Get model details
  aima model get llama3.2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runModelGet(cmd.Context(), root, args[0])
		},
	}

	return cmd
}

func runModelGet(ctx context.Context, root *RootCommand, modelID string) error {
	gw := root.Gateway()
	opts := root.OutputOptions()

	req := &gateway.Request{
		Type: gateway.TypeQuery,
		Unit: "model.get",
		Input: map[string]any{
			"model_id": modelID,
		},
	}

	resp := gw.Handle(ctx, req)

	if !resp.Success {
		PrintError(fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message), opts)
		return fmt.Errorf("get model failed: %s", resp.Error.Message)
	}

	return PrintOutput(resp.Data, opts)
}

func NewModelDeleteCommand(root *RootCommand) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "delete <model_id>",
		Aliases: []string{"rm"},
		Short:   "Delete a model",
		Long:    `Delete a model from the registry.`,
		Args:    cobra.ExactArgs(1),
		Example: `  # Delete a model
  aima model delete llama3.2

  # Force delete
  aima model delete llama3.2 --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runModelDelete(cmd.Context(), root, args[0], force)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force delete even if model is in use")

	return cmd
}

func runModelDelete(ctx context.Context, root *RootCommand, modelID string, force bool) error {
	gw := root.Gateway()
	opts := root.OutputOptions()

	req := &gateway.Request{
		Type: gateway.TypeCommand,
		Unit: "model.delete",
		Input: map[string]any{
			"model_id": modelID,
			"force":    force,
		},
	}

	resp := gw.Handle(ctx, req)

	if !resp.Success {
		PrintError(fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message), opts)
		return fmt.Errorf("delete model failed: %s", resp.Error.Message)
	}

	PrintSuccess(fmt.Sprintf("Model %s deleted successfully", modelID), opts)
	return nil
}

func NewModelCreateCommand(root *RootCommand) *cobra.Command {
	var (
		name      string
		modelType string
		source    string
		format    string
		path      string
	)

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a model record",
		Long:  `Create a new model record in the registry.`,
		Args:  cobra.MaximumNArgs(1),
		Example: `  # Create a model record
  aima model create llama3 --type llm --format gguf

  # Create with path
  aima model create my-model --type llm --path /models/my-model`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				name = args[0]
			}
			return runModelCreate(cmd.Context(), root, name, modelType, source, format, path)
		},
	}

	cmd.Flags().StringVarP(&modelType, "type", "t", "llm", "Model type (llm, vlm, asr, tts, embedding)")
	cmd.Flags().StringVarP(&source, "source", "s", "", "Model source")
	cmd.Flags().StringVar(&format, "format", "gguf", "Model format (gguf, safetensors, onnx)")
	cmd.Flags().StringVar(&path, "path", "", "Local path to model files")

	return cmd
}

func runModelCreate(ctx context.Context, root *RootCommand, name, modelType, source, format, path string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}

	gw := root.Gateway()
	opts := root.OutputOptions()

	input := map[string]any{
		"name":   name,
		"type":   modelType,
		"format": format,
	}
	if source != "" {
		input["source"] = source
	}
	if path != "" {
		input["path"] = path
	}

	req := &gateway.Request{
		Type:  gateway.TypeCommand,
		Unit:  "model.create",
		Input: input,
	}

	resp := gw.Handle(ctx, req)

	if !resp.Success {
		PrintError(fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message), opts)
		return fmt.Errorf("create model failed: %s", resp.Error.Message)
	}

	return PrintOutput(resp.Data, opts)
}
