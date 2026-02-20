package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/spf13/cobra"
)

func NewSkillCommand(root *RootCommand) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Skill knowledge base management",
		Long: `Manage the AIMA skill knowledge base.

Skills are structured best-practice documents loaded into the Agent's
system prompt to guide its decisions.`,
	}

	cmd.AddCommand(NewSkillListCommand(root))
	cmd.AddCommand(NewSkillGetCommand(root))
	cmd.AddCommand(NewSkillAddCommand(root))
	cmd.AddCommand(NewSkillRemoveCommand(root))
	cmd.AddCommand(NewSkillSearchCommand(root))

	return cmd
}

func NewSkillListCommand(root *RootCommand) *cobra.Command {
	var (
		category    string
		source      string
		enabledOnly bool
	)

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List skills",
		Long:    `List all skills in the knowledge base.`,
		Example: `  # List all skills
  aima skill list

  # List only enabled setup skills
  aima skill list --category setup --enabled`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillList(cmd.Context(), root, category, source, enabledOnly)
		},
	}

	cmd.Flags().StringVarP(&category, "category", "c", "", "Filter by category (setup, troubleshoot, optimize, manage)")
	cmd.Flags().StringVarP(&source, "source", "s", "", "Filter by source (builtin, user, community)")
	cmd.Flags().BoolVarP(&enabledOnly, "enabled", "e", false, "Only show enabled skills")

	return cmd
}

func runSkillList(ctx context.Context, root *RootCommand, category, source string, enabledOnly bool) error {
	gw := root.Gateway()
	opts := root.OutputOptions()

	input := map[string]any{}
	if category != "" {
		input["category"] = category
	}
	if source != "" {
		input["source"] = source
	}
	if enabledOnly {
		input["enabled_only"] = true
	}

	req := &gateway.Request{
		Type:  gateway.TypeQuery,
		Unit:  "skill.list",
		Input: input,
	}

	resp := gw.Handle(ctx, req)
	if !resp.Success {
		PrintError(fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message), opts)
		return fmt.Errorf("list skills failed: %s", resp.Error.Message)
	}

	return PrintOutput(resp.Data, opts)
}

func NewSkillGetCommand(root *RootCommand) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <skill-id>",
		Short: "Get skill details",
		Long:  `Get detailed information about a specific skill, including its full content.`,
		Example: `  # Get skill details
  aima skill get setup-llm`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillGet(cmd.Context(), root, args[0])
		},
	}

	return cmd
}

func runSkillGet(ctx context.Context, root *RootCommand, skillID string) error {
	gw := root.Gateway()
	opts := root.OutputOptions()

	req := &gateway.Request{
		Type: gateway.TypeQuery,
		Unit: "skill.get",
		Input: map[string]any{
			"skill_id": skillID,
		},
	}

	resp := gw.Handle(ctx, req)
	if !resp.Success {
		PrintError(fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message), opts)
		return fmt.Errorf("get skill failed: %s", resp.Error.Message)
	}

	return PrintOutput(resp.Data, opts)
}

func NewSkillAddCommand(root *RootCommand) *cobra.Command {
	var (
		filePath string
		source   string
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new skill",
		Long: `Add a new skill from a Markdown file with YAML front-matter.

The skill file format:
  ---
  id: my-skill
  name: "My Skill"
  category: manage
  description: "Description of the skill"
  trigger:
    keywords: ["keyword1", "keyword2"]
  priority: 5
  enabled: true
  source: user
  ---

  # Skill Content

  Markdown body with instructions...`,
		Example: `  # Add a skill from file
  aima skill add --file my-skill.md

  # Add a community skill
  aima skill add --file community-skill.md --source community`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillAdd(cmd.Context(), root, filePath, source)
		},
	}

	cmd.Flags().StringVarP(&filePath, "file", "f", "", "Skill file path (Markdown with YAML front-matter)")
	cmd.Flags().StringVarP(&source, "source", "s", "user", "Skill source (user, community)")
	cmd.MarkFlagRequired("file")

	return cmd
}

func runSkillAdd(ctx context.Context, root *RootCommand, filePath, source string) error {
	gw := root.Gateway()
	opts := root.OutputOptions()

	// Read file content
	content, err := readFileContent(filePath)
	if err != nil {
		PrintError(fmt.Errorf("read file: %w", err), opts)
		return fmt.Errorf("read file: %w", err)
	}

	req := &gateway.Request{
		Type: gateway.TypeCommand,
		Unit: "skill.add",
		Input: map[string]any{
			"content": content,
			"source":  source,
		},
	}

	resp := gw.Handle(ctx, req)
	if !resp.Success {
		PrintError(fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message), opts)
		return fmt.Errorf("add skill failed: %s", resp.Error.Message)
	}

	PrintSuccess("Skill added successfully", opts)
	return PrintOutput(resp.Data, opts)
}

func NewSkillRemoveCommand(root *RootCommand) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <skill-id>",
		Short: "Remove a user skill",
		Long:  `Remove a user skill. Builtin skills cannot be removed.`,
		Example: `  # Remove a skill
  aima skill remove my-skill-id`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillRemove(cmd.Context(), root, args[0])
		},
	}

	return cmd
}

func runSkillRemove(ctx context.Context, root *RootCommand, skillID string) error {
	gw := root.Gateway()
	opts := root.OutputOptions()

	req := &gateway.Request{
		Type: gateway.TypeCommand,
		Unit: "skill.remove",
		Input: map[string]any{
			"skill_id": skillID,
		},
	}

	resp := gw.Handle(ctx, req)
	if !resp.Success {
		PrintError(fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message), opts)
		return fmt.Errorf("remove skill failed: %s", resp.Error.Message)
	}

	PrintSuccess(fmt.Sprintf("Skill %s removed", skillID), opts)
	return nil
}

func NewSkillSearchCommand(root *RootCommand) *cobra.Command {
	var category string

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search skills",
		Long:  `Search skills by text query matching name, description, and content.`,
		Example: `  # Search for GPU-related skills
  aima skill search gpu

  # Search within a specific category
  aima skill search "performance" --category optimize`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSkillSearch(cmd.Context(), root, args[0], category)
		},
	}

	cmd.Flags().StringVarP(&category, "category", "c", "", "Filter by category")

	return cmd
}

func runSkillSearch(ctx context.Context, root *RootCommand, query, category string) error {
	gw := root.Gateway()
	opts := root.OutputOptions()

	input := map[string]any{"query": query}
	if category != "" {
		input["category"] = category
	}

	req := &gateway.Request{
		Type:  gateway.TypeQuery,
		Unit:  "skill.search",
		Input: input,
	}

	resp := gw.Handle(ctx, req)
	if !resp.Success {
		PrintError(fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message), opts)
		return fmt.Errorf("search skills failed: %s", resp.Error.Message)
	}

	return PrintOutput(resp.Data, opts)
}

// readFileContent reads file content as a string.
func readFileContent(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
