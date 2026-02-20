package cli

import (
	"context"
	"fmt"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/spf13/cobra"
)

func NewCatalogCommand(root *RootCommand) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "catalog",
		Short: "Recipe catalog management commands",
		Long: `Manage the AIMA recipe catalog.

Recipes are hardware best-practice configurations that combine an inference
engine, model, and settings validated for specific hardware profiles.`,
	}

	cmd.AddCommand(NewCatalogListCommand(root))
	cmd.AddCommand(NewCatalogGetCommand(root))
	cmd.AddCommand(NewCatalogMatchCommand(root))
	cmd.AddCommand(NewCatalogApplyCommand(root))
	cmd.AddCommand(NewCatalogValidateCommand(root))

	return cmd
}

func NewCatalogListCommand(root *RootCommand) *cobra.Command {
	var (
		gpuVendor    string
		verifiedOnly bool
		tags         []string
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List available recipes",
		Long:    `List all available recipes in the catalog.`,
		Example: `  # List all recipes
  aima catalog list

  # List verified NVIDIA recipes
  aima catalog list --gpu-vendor nvidia --verified`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCatalogList(cmd.Context(), root, gpuVendor, verifiedOnly, tags)
		},
	}

	cmd.Flags().StringVar(&gpuVendor, "gpu-vendor", "", "Filter by GPU vendor (nvidia, amd, apple)")
	cmd.Flags().BoolVar(&verifiedOnly, "verified", false, "Only show verified recipes")
	cmd.Flags().StringSliceVar(&tags, "tags", nil, "Filter by tags")

	return cmd
}

func runCatalogList(ctx context.Context, root *RootCommand, gpuVendor string, verifiedOnly bool, tags []string) error {
	gw := root.Gateway()
	opts := root.OutputOptions()

	input := map[string]any{}
	if gpuVendor != "" {
		input["gpu_vendor"] = gpuVendor
	}
	if verifiedOnly {
		input["verified_only"] = true
	}
	if len(tags) > 0 {
		input["tags"] = tags
	}

	req := &gateway.Request{
		Type:  gateway.TypeQuery,
		Unit:  "catalog.list",
		Input: input,
	}

	resp := gw.Handle(ctx, req)
	if !resp.Success {
		PrintError(fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message), opts)
		return fmt.Errorf("list recipes failed: %s", resp.Error.Message)
	}

	return PrintOutput(resp.Data, opts)
}

func NewCatalogGetCommand(root *RootCommand) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <recipe-id>",
		Short: "Get recipe details",
		Long:  `Get detailed information about a specific recipe.`,
		Example: `  # Get recipe details
  aima catalog get my-recipe-id`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCatalogGet(cmd.Context(), root, args[0])
		},
	}

	return cmd
}

func runCatalogGet(ctx context.Context, root *RootCommand, recipeID string) error {
	gw := root.Gateway()
	opts := root.OutputOptions()

	req := &gateway.Request{
		Type: gateway.TypeQuery,
		Unit: "catalog.get",
		Input: map[string]any{
			"recipe_id": recipeID,
		},
	}

	resp := gw.Handle(ctx, req)
	if !resp.Success {
		PrintError(fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message), opts)
		return fmt.Errorf("get recipe failed: %s", resp.Error.Message)
	}

	return PrintOutput(resp.Data, opts)
}

func NewCatalogMatchCommand(root *RootCommand) *cobra.Command {
	var (
		gpuVendor string
		gpuModel  string
		vramGB    int
		os        string
		limit     int
	)

	cmd := &cobra.Command{
		Use:   "match",
		Short: "Find recipes matching your hardware",
		Long:  `Match recipes to your current or specified hardware profile.`,
		Example: `  # Match recipes to NVIDIA hardware
  aima catalog match --gpu-vendor nvidia --vram 24

  # Match with specific GPU model
  aima catalog match --gpu-vendor nvidia --gpu-model "RTX 4090"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCatalogMatch(cmd.Context(), root, gpuVendor, gpuModel, vramGB, os, limit)
		},
	}

	cmd.Flags().StringVar(&gpuVendor, "gpu-vendor", "", "GPU vendor (nvidia, amd, apple)")
	cmd.Flags().StringVar(&gpuModel, "gpu-model", "", "GPU model name")
	cmd.Flags().IntVar(&vramGB, "vram", 0, "VRAM in GB")
	cmd.Flags().StringVar(&os, "os", "", "Operating system (linux, darwin, windows)")
	cmd.Flags().IntVar(&limit, "limit", 5, "Maximum number of results")

	return cmd
}

func runCatalogMatch(ctx context.Context, root *RootCommand, gpuVendor, gpuModel string, vramGB int, osName string, limit int) error {
	gw := root.Gateway()
	opts := root.OutputOptions()

	input := map[string]any{"limit": limit}
	if gpuVendor != "" {
		input["gpu_vendor"] = gpuVendor
	}
	if gpuModel != "" {
		input["gpu_model"] = gpuModel
	}
	if vramGB > 0 {
		input["vram_gb"] = vramGB
	}
	if osName != "" {
		input["os"] = osName
	}

	req := &gateway.Request{
		Type:  gateway.TypeQuery,
		Unit:  "catalog.match",
		Input: input,
	}

	resp := gw.Handle(ctx, req)
	if !resp.Success {
		PrintError(fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message), opts)
		return fmt.Errorf("match recipes failed: %s", resp.Error.Message)
	}

	return PrintOutput(resp.Data, opts)
}

func NewCatalogApplyCommand(root *RootCommand) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply <recipe-id>",
		Short: "Apply a recipe to deploy engine and model",
		Long: `Apply a recipe to automatically deploy the inference engine and download the model.

This command will:
1. Pull the inference engine image
2. Download the specified model
3. Start the inference service`,
		Example: `  # Apply a recipe
  aima catalog apply my-recipe-id`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCatalogApply(cmd.Context(), root, args[0])
		},
	}

	return cmd
}

func runCatalogApply(ctx context.Context, root *RootCommand, recipeID string) error {
	gw := root.Gateway()
	opts := root.OutputOptions()

	req := &gateway.Request{
		Type: gateway.TypeCommand,
		Unit: "catalog.apply_recipe",
		Input: map[string]any{
			"recipe_id": recipeID,
		},
	}

	resp := gw.Handle(ctx, req)
	if !resp.Success {
		PrintError(fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message), opts)
		return fmt.Errorf("apply recipe failed: %s", resp.Error.Message)
	}

	PrintSuccess(fmt.Sprintf("Recipe %s applied successfully", recipeID), opts)
	return PrintOutput(resp.Data, opts)
}

func NewCatalogValidateCommand(root *RootCommand) *cobra.Command {
	var file string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate a recipe",
		Long:  `Validate a recipe definition for correctness.`,
		Example: `  # Validate a recipe from file
  aima catalog validate --file my-recipe.yaml`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCatalogValidate(cmd.Context(), root, file)
		},
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "Recipe file path (YAML)")
	_ = cmd.MarkFlagRequired("file")

	return cmd
}

func runCatalogValidate(ctx context.Context, root *RootCommand, filePath string) error {
	gw := root.Gateway()
	opts := root.OutputOptions()

	req := &gateway.Request{
		Type: gateway.TypeCommand,
		Unit: "catalog.validate_recipe",
		Input: map[string]any{
			"file": filePath,
		},
	}

	resp := gw.Handle(ctx, req)
	if !resp.Success {
		PrintError(fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message), opts)
		return fmt.Errorf("validate recipe failed: %s", resp.Error.Message)
	}

	PrintSuccess("Recipe is valid", opts)
	return PrintOutput(resp.Data, opts)
}
