package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jguan/ai-inference-managed-by-ai/pkg/gateway"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewExecCommand(root *RootCommand) *cobra.Command {
	var inputJSON string

	cmd := &cobra.Command{
		Use:   "exec <unit> [flags]",
		Short: "Execute an atomic unit",
		Long: `Execute any atomic unit (command or query) directly.

The unit name should be in the format "domain.action", for example:
  - model.pull
  - model.list
  - inference.chat
  - device.detect

Input parameters can be provided via flags or using --input with JSON.`,
		Example: `  # Execute model.pull with flags
  aima exec model.pull --source ollama --repo llama3.2

  # Execute with JSON input
  aima exec model.pull --input '{"source":"ollama","repo":"llama3.2"}'

  # Execute model.list
  aima exec model.list --type llm

  # Execute with JSON output
  aima exec model.list --output json`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			unitName := args[0]
			return runExec(cmd.Context(), root, unitName, inputJSON, cmd)
		},
	}

	cmd.Flags().StringVarP(&inputJSON, "input", "i", "", "JSON input for the unit")

	return cmd
}

func runExec(ctx context.Context, root *RootCommand, unitName string, inputJSON string, cmd *cobra.Command) error {
	gw := root.Gateway()
	opts := root.OutputOptions()

	input := make(map[string]any)

	if inputJSON != "" {
		if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
			return fmt.Errorf("parse input JSON: %w", err)
		}
	}

	flags := cmd.Flags()
	flagMap := extractFlags(flags)

	for k, v := range flagMap {
		if k != "input" && k != "output" && k != "quiet" && k != "config" && k != "help" {
			if v != "" && v != nil {
				input[k] = v
			}
		}
	}

	reqType := gateway.TypeCommand
	if root.Registry().GetQuery(unitName) != nil {
		reqType = gateway.TypeQuery
	}

	req := &gateway.Request{
		Type:  reqType,
		Unit:  unitName,
		Input: input,
	}

	resp := gw.Handle(ctx, req)

	if !resp.Success {
		PrintError(fmt.Errorf("%s: %s", resp.Error.Code, resp.Error.Message), opts)
		return fmt.Errorf("execution failed: %s", resp.Error.Message)
	}

	return PrintOutput(resp.Data, opts)
}

func extractFlags(flags *pflag.FlagSet) map[string]any {
	result := make(map[string]any)

	flags.VisitAll(func(f *pflag.Flag) {
		if f.Changed {
			result[f.Name] = f.Value.String()
		}
	})

	return result
}
