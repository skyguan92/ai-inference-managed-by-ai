package main

import (
	"github.com/jguan/ai-inference-managed-by-ai/pkg/cli"
)

var (
	version   = "dev"
	buildDate = "unknown"
	gitCommit = "unknown"
)

func main() {
	cli.SetVersion(version, buildDate, gitCommit)
	cli.Execute()
}
