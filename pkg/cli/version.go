package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewVersionCommand(root *RootCommand) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  "Display the version, build date, and git commit of AIMA.",
		Run: func(cmd *cobra.Command, args []string) {
			printVersion(root.OutputOptions())
		},
	}

	return cmd
}

func printVersion(opts *OutputOptions) {
	versionInfo := map[string]string{
		"version":   cliVersion,
		"buildDate": cliBuildDate,
		"gitCommit": cliGitCommit,
	}

	if opts.Format == OutputJSON || opts.Format == OutputYAML {
		PrintOutput(versionInfo, opts)
	} else {
		fmt.Fprintf(opts.Writer, "AIMA version %s\n", cliVersion)
		fmt.Fprintf(opts.Writer, "  Commit: %s\n", cliGitCommit)
		fmt.Fprintf(opts.Writer, "  Built:  %s\n", cliBuildDate)
	}
}
