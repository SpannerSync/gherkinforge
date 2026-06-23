// Package run implements the `gforge run` command.
// It walks a directory tree, reads the tier tag from each .feature file,
// and prints a routing table mapping each file to its appropriate test runner.
package run

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/cucumber/gherkin/go/v26"
	"github.com/spf13/cobra"
)

// tierRunner maps a tier tag to its canonical test runner.
var tierRunner = map[string]string{
	"@business": "godog",
	"@contract": "pact-go",
	"@nfr":      "k6",
	"@draft":    "lint-only",
}

// validTiers is the set of recognised tier tags for routing purposes.
var validTiers = map[string]bool{
	"@business": true,
	"@contract": true,
	"@nfr":      true,
	"@draft":    true,
}

// NewCommand returns the cobra command for `gforge run <directory>`.
func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "run <directory>",
		Short: "Print a tier-to-runner routing table for .feature files",
		Long: `run walks the given directory, reads the tier tag from each .feature file,
and prints which test runner should execute it:

  @business → godog
  @contract → pact-go
  @nfr      → k6
  @draft    → lint-only
  no tier   → unknown (run gforge lint first)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunDir(args[0], cmd.OutOrStdout())
		},
	}
}

// RunDir walks root and writes a routing table to out.
func RunDir(root string, out io.Writer) error {
	w := tabwriter.NewWriter(out, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "FILE\tTIER\tRUNNER")

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".feature") {
			return nil
		}
		rel, _ := filepath.Rel(root, path)

		tier, parseErr := extractTier(path)
		if parseErr != nil {
			fmt.Fprintf(w, "%s\t%s\t%s\n", rel, "", "PARSE-ERROR")
			return nil
		}

		runner := runnerFor(tier)
		fmt.Fprintf(w, "%s\t%s\t%s\n", rel, tier, runner)
		return nil
	})

	w.Flush()
	return err
}

// extractTier opens path, parses it with the Gherkin library, and returns
// the first recognised tier tag found at feature level.
// Returns ("", nil) when the file has no tier tag.
// Returns ("", err) when the file cannot be parsed.
func extractTier(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var idSeq int
	newID := func() string { idSeq++; return strconv.Itoa(idSeq) }
	doc, err := gherkin.ParseGherkinDocument(f, newID)
	if err != nil {
		return "", err
	}
	if doc.Feature == nil {
		return "", nil
	}
	for _, tag := range doc.Feature.Tags {
		if validTiers[tag.Name] {
			return tag.Name, nil
		}
	}
	return "", nil
}

// runnerFor returns the test runner for the given tier tag.
func runnerFor(tier string) string {
	if r, ok := tierRunner[tier]; ok {
		return r
	}
	return "unknown (run gforge lint first)"
}
