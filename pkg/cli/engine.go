package cli

import (
	"context"
	"fmt"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/spf13/cobra"
)

func NewEngineCommand(root *RootCommand) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "engine",
		Short: "Engine management commands",
		Long: `Manage inference engines in the AIMA infrastructure.

This includes starting, stopping, and listing inference engines.`,
	}

	cmd.AddCommand(NewEngineStartCommand(root))
	cmd.AddCommand(NewEngineStopCommand(root))
	cmd.AddCommand(NewEngineListCommand(root))

	return cmd
}

func NewEngineStartCommand(root *RootCommand) *cobra.Command {
	var config string

	cmd := &cobra.Command{
		Use:   "start <engine>",
		Short: "Start an inference engine",
		Long: `Start an inference engine by name.

Supported engines include ollama, vllm, and others.`,
		Example: `  # Start Ollama engine
  aima engine start ollama

  # Start with config
  aima engine start ollama --config /path/to/config.json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEngineStart(cmd.Context(), root, args[0], config)
		},
	}

	cmd.Flags().StringVarP(&config, "config", "c", "", "Engine configuration file")

	return cmd
}

func runEngineStart(ctx context.Context, root *RootCommand, engineName, configPath string) error {
	gw := root.Gateway()
	opts := root.OutputOptions()

	input := map[string]any{
		"name": engineName,
	}
	if configPath != "" {
		input["config"] = configPath
	}

	req := &gateway.Request{
		Type:  gateway.TypeCommand,
		Unit:  "engine.start",
		Input: input,
	}

	resp := gw.Handle(ctx, req)

	if !resp.Success {
		PrintError(fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message), opts)
		return fmt.Errorf("start engine failed: %s", resp.Error.Message)
	}

	PrintSuccess(fmt.Sprintf("Engine %s started successfully", engineName), opts)
	return PrintOutput(resp.Data, opts)
}

func NewEngineStopCommand(root *RootCommand) *cobra.Command {
	var force bool
	var timeout int

	cmd := &cobra.Command{
		Use:   "stop <engine>",
		Short: "Stop an inference engine",
		Long: `Stop a running inference engine.

By default, performs a graceful shutdown. Use --force for immediate termination.`,
		Example: `  # Stop Ollama engine gracefully
  aima engine stop ollama

  # Force stop
  aima engine stop ollama --force

  # Stop with timeout
  aima engine stop ollama --timeout 30`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEngineStop(cmd.Context(), root, args[0], force, timeout)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Force stop the engine")
	cmd.Flags().IntVar(&timeout, "timeout", 30, "Shutdown timeout in seconds")

	return cmd
}

func runEngineStop(ctx context.Context, root *RootCommand, engineName string, force bool, timeout int) error {
	gw := root.Gateway()
	opts := root.OutputOptions()

	input := map[string]any{
		"name":    engineName,
		"force":   force,
		"timeout": timeout,
	}

	req := &gateway.Request{
		Type:  gateway.TypeCommand,
		Unit:  "engine.stop",
		Input: input,
	}

	resp := gw.Handle(ctx, req)

	if !resp.Success {
		PrintError(fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message), opts)
		return fmt.Errorf("stop engine failed: %s", resp.Error.Message)
	}

	PrintSuccess(fmt.Sprintf("Engine %s stopped successfully", engineName), opts)
	return nil
}

func NewEngineListCommand(root *RootCommand) *cobra.Command {
	var engineType string
	var status string

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List inference engines",
		Long:    `List all registered inference engines and their status.`,
		Example: `  # List all engines
  aima engine list

  # List only running engines
  aima engine list --status running

  # List with JSON output
  aima engine list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEngineList(cmd.Context(), root, engineType, status)
		},
	}

	cmd.Flags().StringVarP(&engineType, "type", "t", "", "Filter by engine type (ollama, vllm)")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status (running, stopped, error)")

	return cmd
}

func runEngineList(ctx context.Context, root *RootCommand, engineType, status string) error {
	gw := root.Gateway()
	opts := root.OutputOptions()

	input := map[string]any{}
	if engineType != "" {
		input["type"] = engineType
	}
	if status != "" {
		input["status"] = status
	}

	req := &gateway.Request{
		Type:  gateway.TypeQuery,
		Unit:  "engine.list",
		Input: input,
	}

	resp := gw.Handle(ctx, req)

	if !resp.Success {
		PrintError(fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message), opts)
		return fmt.Errorf("list engines failed: %s", resp.Error.Message)
	}

	return PrintOutput(resp.Data, opts)
}
