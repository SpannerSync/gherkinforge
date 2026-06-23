package main

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spannersync/gherkinforge/internal/cli/lint"
	"github.com/spannersync/gherkinforge/internal/cli/scaffold"
)

func main() {
	root := &cobra.Command{
		Use:   "gforge",
		Short: "GherkinForge — dual-audience Gherkin scaffolding and linting tool",
		Long: `GherkinForge enforces the Dual-Audience Gherkin pattern:
feature files simultaneously serve as human-readable business requirements
and as deterministic anchors for AI-assisted hexagonal Go code generation.`,
	}

	root.AddCommand(scaffold.NewCommand())
	root.AddCommand(lint.NewCommand())

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
