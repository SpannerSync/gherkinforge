// Package lint implements the `gforge lint` command.
// It validates that every .feature file in the target directory tree:
//
//  1. Has exactly one tier tag: @business, @integration, or @nfr.
//  2. @business files use at least one DataTable or DocString to anchor types.
//  3. No step text contains forbidden implementation symbols (SQL keywords,
//     HTTP paths, selector strings, handler names).
//  4. ZERO TRUST Pillar 2: @business steps must not contain UI/DOM language
//     (click, button, xpath, css selector, etc.) — these couple backend specs
//     to frontend rendering, producing brittle tests.
package lint

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/cucumber/gherkin/go/v26"
	messages "github.com/cucumber/messages/go/v21"
	"github.com/spf13/cobra"
)

var validTiers = map[string]bool{
	"@business":    true,
	"@integration": true,
	"@nfr":         true,
}

// forbiddenPatterns are substrings that must not appear in any step text
// regardless of tier. They indicate technical implementation leaking into specs.
var forbiddenPatterns = []string{
	"data-testid",
	"SELECT ", "INSERT ", "UPDATE ", "DELETE ",
	"/api/", "/v1/", "/v2/",
	".handler", ".service", "Handler{", "Service{",
}

// restrictedBusinessWords are terms that must not appear in @business tier step
// text. They couple the domain specification to UI rendering decisions, producing
// tests that break on cosmetic frontend changes.
//
// ZERO TRUST Pillar 2: the AI is not trusted to avoid these terms. The linter
// enforces the rule before the AI is allowed to read the feature file.
var restrictedBusinessWords = []string{
	"click",
	"button",
	"dropdown",
	"checkbox",
	"radio button",
	"scroll",
	"hover",
	"drag",
	"xpath",
	"css selector",
	"data-testid",
	"aria-label",
	"id=",
	"class=",
	"div",
	"span",
	"input field",
	"text box",
	"modal",
	"popup",
	"tab key",
	"keyboard",
	"mouse",
	"screen",
	"pixel",
	"viewport",
	"browser",
	"dom",
}

// Violation records a single lint failure.
type Violation struct {
	File    string
	Line    int
	Message string
}

func (v Violation) String() string {
	return fmt.Sprintf("%s:%d: %s", v.File, v.Line, v.Message)
}

// NewCommand returns the cobra command for `gforge lint <dir>`.
func NewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lint [directory]",
		Short: "Validate .feature files against dual-audience Gherkin rules",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			violations, err := LintDir(args[0])
			if err != nil {
				return err
			}
			if len(violations) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "✓ No violations found.")
				return nil
			}
			for _, v := range violations {
				fmt.Fprintln(cmd.ErrOrStderr(), v)
			}
			return fmt.Errorf("%d violation(s) found", len(violations))
		},
	}
	return cmd
}

// LintDir walks root recursively and lints every .feature file found.
func LintDir(root string) ([]Violation, error) {
	var all []Violation
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".feature") {
			return nil
		}
		vs, lintErr := LintFile(path)
		if lintErr != nil {
			return lintErr
		}
		all = append(all, vs...)
		return nil
	})
	return all, err
}

// LintFile parses a single .feature file and returns its violations.
func LintFile(path string) ([]Violation, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	var idSeq int
	newIDFunc := func() string { idSeq++; return strconv.Itoa(idSeq) }
	doc, err := gherkin.ParseGherkinDocument(f, newIDFunc)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if doc.Feature == nil {
		return nil, nil
	}

	var vs []Violation

	// Rule 1: exactly one tier tag at feature level.
	tierCount := 0
	for _, tag := range doc.Feature.Tags {
		if validTiers[tag.Name] {
			tierCount++
		}
	}
	if tierCount == 0 {
		vs = append(vs, Violation{
			File:    path,
			Line:    int(doc.Feature.Location.Line),
			Message: "missing tier tag — add @business, @integration, or @nfr",
		})
	}
	if tierCount > 1 {
		vs = append(vs, Violation{
			File:    path,
			Line:    int(doc.Feature.Location.Line),
			Message: "multiple tier tags — a feature file must belong to exactly one tier",
		})
	}

	// Determine which tier this file claims.
	tier := tierFor(doc.Feature.Tags)

	// Collect all scenarios (including outline and background).
	hasDataTableOrDocString := false
	for _, child := range doc.Feature.Children {
		var steps []*messages.Step
		switch {
		case child.Scenario != nil:
			steps = child.Scenario.Steps
		case child.Background != nil:
			steps = child.Background.Steps
		}

		for _, step := range steps {
			// Rule 2: @business files need at least one structured anchor.
			if step.DataTable != nil || step.DocString != nil {
				hasDataTableOrDocString = true
			}

			// Rule 3: no forbidden implementation symbols in any step text.
			for _, forbidden := range forbiddenPatterns {
				if strings.Contains(step.Text, forbidden) {
					vs = append(vs, Violation{
						File:    path,
						Line:    int(step.Location.Line),
						Message: fmt.Sprintf("forbidden symbol %q in step text — keep steps at business language level", forbidden),
					})
				}
			}

			// Zero Trust Pillar 2: @business steps must use domain language only.
			// UI/DOM vocabulary couples backend specs to frontend rendering.
			// Whole-word matching prevents false positives (e.g. "dom" inside "domain").
			if tier == "@business" {
				lower := strings.ToLower(step.Text)
				for _, word := range restrictedBusinessWords {
					pattern := `\b` + regexp.QuoteMeta(word) + `\b`
					matched, _ := regexp.MatchString(pattern, lower)
					if matched {
						vs = append(vs, Violation{
							File:    path,
							Line:    int(step.Location.Line),
							Message: fmt.Sprintf("ZERO TRUST VIOLATION: UI-specific term %q found in @business tier step — domain specs must be UI-agnostic", word),
						})
					}
				}
			}
		}
	}

	if tier == "@business" && !hasDataTableOrDocString {
		vs = append(vs, Violation{
			File:    path,
			Line:    int(doc.Feature.Location.Line),
			Message: "@business feature must use at least one DataTable or DocString to anchor parameter types",
		})
	}

	return vs, nil
}

func tierFor(tags []*messages.Tag) string {
	for _, t := range tags {
		if validTiers[t.Name] {
			return t.Name
		}
	}
	return ""
}

// ErrViolationsFound is returned when lint finds at least one violation.
var ErrViolationsFound = errors.New("lint violations found")
