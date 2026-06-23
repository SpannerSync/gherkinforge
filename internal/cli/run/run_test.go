package run

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFeature writes a .feature file into dir and returns the full path.
func writeFeature(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeFeature: %v", err)
	}
	return path
}

// runOutput calls RunDir and returns stdout as a string.
func runOutput(t *testing.T, dir string) string {
	t.Helper()
	var buf bytes.Buffer
	if err := RunDir(dir, &buf); err != nil {
		t.Fatalf("RunDir: %v", err)
	}
	return buf.String()
}

func TestRunDir_BusinessRoutes_ToGodog(t *testing.T) {
	dir := t.TempDir()
	writeFeature(t, dir, "orders.feature", `@business
Feature: Order Management
  Scenario: Create order
    Given a customer aggregate with ID "CUST-1"
    When an order is submitted with items:
      | product_id | quantity |
      | P-001      | 2        |
    Then the order is created
`)
	out := runOutput(t, dir)
	if !strings.Contains(out, "@business") {
		t.Errorf("expected @business in output, got:\n%s", out)
	}
	if !strings.Contains(out, "godog") {
		t.Errorf("expected godog runner in output, got:\n%s", out)
	}
}

func TestRunDir_ContractRoutes_ToPactGo(t *testing.T) {
	dir := t.TempDir()
	writeFeature(t, dir, "invoice_contract.feature", `@contract
Feature: Invoice API contract
  Scenario: Provider returns invoice
    Given the invoice provider is available
    When the consumer requests invoice INV-001 with details:
      | field  | value    |
      | id     | INV-001  |
    Then the response matches the agreed schema
`)
	out := runOutput(t, dir)
	if !strings.Contains(out, "@contract") {
		t.Errorf("expected @contract in output, got:\n%s", out)
	}
	if !strings.Contains(out, "pact-go") {
		t.Errorf("expected pact-go runner in output, got:\n%s", out)
	}
}

func TestRunDir_NfrRoutes_ToK6(t *testing.T) {
	dir := t.TempDir()
	writeFeature(t, dir, "throughput.feature", `@nfr
Feature: Order throughput
  Scenario: Handles peak load
    Given 500 concurrent users are active
    When each submits a request
    Then all responses arrive within 2 seconds
`)
	out := runOutput(t, dir)
	if !strings.Contains(out, "@nfr") {
		t.Errorf("expected @nfr in output, got:\n%s", out)
	}
	if !strings.Contains(out, "k6") {
		t.Errorf("expected k6 runner in output, got:\n%s", out)
	}
}

func TestRunDir_DraftRoutes_ToLintOnly(t *testing.T) {
	dir := t.TempDir()
	writeFeature(t, dir, "wip.feature", `@draft
Feature: Work in progress feature
  Scenario: Placeholder scenario
    Given something is being explored
    When the team reviews requirements
    Then acceptance criteria will be decided
`)
	out := runOutput(t, dir)
	if !strings.Contains(out, "@draft") {
		t.Errorf("expected @draft in output, got:\n%s", out)
	}
	if !strings.Contains(out, "lint-only") {
		t.Errorf("expected lint-only runner in output, got:\n%s", out)
	}
}

func TestRunDir_MissingTierRoutes_ToUnknown(t *testing.T) {
	dir := t.TempDir()
	writeFeature(t, dir, "notier.feature", `Feature: No tier tag
  Scenario: Plain scenario
    Given something exists
    When an action is taken
    Then a result is produced
`)
	out := runOutput(t, dir)
	if !strings.Contains(out, "unknown") {
		t.Errorf("expected unknown in output for file with no tier tag, got:\n%s", out)
	}
}

func TestRunDir_MultipleFiles_AllAppear(t *testing.T) {
	dir := t.TempDir()
	writeFeature(t, dir, "biz.feature", `@business
Feature: Business feature
  Scenario: Do something
    Given data exists:
      | key   | value |
      | hello | world |
    When processing occurs
    Then output is produced
`)
	writeFeature(t, dir, "perf.feature", `@nfr
Feature: Performance
  Scenario: Handles load
    Given the system is warmed up
    When 100 requests arrive
    Then all succeed
`)
	out := runOutput(t, dir)
	if !strings.Contains(out, "godog") {
		t.Errorf("expected godog in output, got:\n%s", out)
	}
	if !strings.Contains(out, "k6") {
		t.Errorf("expected k6 in output, got:\n%s", out)
	}
}

func TestRunDir_ParseError_ShowsParseError(t *testing.T) {
	dir := t.TempDir()
	// Write a file with invalid Gherkin syntax.
	path := filepath.Join(dir, "broken.feature")
	if err := os.WriteFile(path, []byte("this is not valid gherkin {{{{"), 0o644); err != nil {
		t.Fatalf("writing broken feature: %v", err)
	}
	out := runOutput(t, dir)
	if !strings.Contains(out, "PARSE-ERROR") {
		t.Errorf("expected PARSE-ERROR in output for broken feature, got:\n%s", out)
	}
}

func TestRunDir_HeaderAlwaysPresent(t *testing.T) {
	dir := t.TempDir()
	out := runOutput(t, dir)
	if !strings.Contains(out, "FILE") || !strings.Contains(out, "TIER") || !strings.Contains(out, "RUNNER") {
		t.Errorf("expected header row in output, got:\n%s", out)
	}
}
