package lint

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFeature writes content to a file called name inside dir and returns the path.
func writeFeature(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeFeature: %v", err)
	}
	return path
}

// writeConfig writes a .gforge.yml into dir.
func writeConfig(t *testing.T, dir, content string) {
	t.Helper()
	path := filepath.Join(dir, ".gforge.yml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writeConfig: %v", err)
	}
}

// hasViolationContaining returns true if any violation message contains substr.
func hasViolationContaining(vs []Violation, substr string) bool {
	for _, v := range vs {
		if strings.Contains(v.Message, substr) {
			return true
		}
	}
	return false
}

// ── LintFile direct tests ─────────────────────────────────────────────────────

func TestLintFile_DraftNoDataTable_Passes(t *testing.T) {
	dir := t.TempDir()
	path := writeFeature(t, dir, "draft.feature", `@draft
Feature: Draft work in progress
  Scenario: Placeholder
    Given something is being explored
    When the team discusses requirements
    Then acceptance criteria will be defined
`)
	vs, err := LintFile(path, forbiddenPatterns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vs) != 0 {
		t.Errorf("expected no violations for @draft without DataTable, got: %v", vs)
	}
}

func TestLintFile_NfrNoDataTable_Passes(t *testing.T) {
	dir := t.TempDir()
	path := writeFeature(t, dir, "nfr.feature", `@nfr
Feature: System throughput
  Scenario: Handles peak load
    Given 500 concurrent users are active
    When each submits a request
    Then all responses arrive within 2 seconds
`)
	vs, err := LintFile(path, forbiddenPatterns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vs) != 0 {
		t.Errorf("expected no violations for @nfr without DataTable, got: %v", vs)
	}
}

func TestLintFile_ContractNoDataTable_Fails(t *testing.T) {
	dir := t.TempDir()
	path := writeFeature(t, dir, "contract.feature", `@contract
Feature: Invoice API contract
  Scenario: Provider returns invoice
    Given the invoice provider is available
    When the consumer requests invoice INV-001
    Then the response matches the agreed schema
`)
	vs, err := LintFile(path, forbiddenPatterns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasViolationContaining(vs, "@contract feature must use at least one DataTable") {
		t.Errorf("expected DataTable violation for @contract without DataTable, got: %v", vs)
	}
}

func TestLintFile_BusinessNoDataTable_Fails(t *testing.T) {
	dir := t.TempDir()
	path := writeFeature(t, dir, "business.feature", `@business
Feature: Order submission
  Scenario: Customer submits order
    Given a customer with ID CUST-001
    When the customer submits an order
    Then the order is created successfully
`)
	vs, err := LintFile(path, forbiddenPatterns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasViolationContaining(vs, "@business feature must use at least one DataTable") {
		t.Errorf("expected DataTable violation for @business without DataTable, got: %v", vs)
	}
}

func TestLintFile_ForbiddenSymbol_Fails(t *testing.T) {
	dir := t.TempDir()
	path := writeFeature(t, dir, "bad.feature", `@nfr
Feature: Leaking impl
  Scenario: Should not contain SQL
    Given the system runs SELECT id FROM orders
    When nothing happens
    Then it should not be here
`)
	vs, err := LintFile(path, forbiddenPatterns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasViolationContaining(vs, "forbidden symbol") {
		t.Errorf("expected forbidden-symbol violation, got: %v", vs)
	}
}

func TestLintFile_MissingTier_Fails(t *testing.T) {
	dir := t.TempDir()
	path := writeFeature(t, dir, "notier.feature", `Feature: No tier tag
  Scenario: Plain scenario
    Given something
    When something else
    Then a result
`)
	vs, err := LintFile(path, forbiddenPatterns)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasViolationContaining(vs, "missing tier tag") {
		t.Errorf("expected missing-tier violation, got: %v", vs)
	}
}

// ── Config integration via LintDir ───────────────────────────────────────────

func TestLintDir_DenyTerms_AddsToForbidden(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `lint:
  deny_terms:
    - "use_case_ref"
`)
	// @nfr so no DataTable requirement; the custom term is the only trigger.
	writeFeature(t, dir, "nfr.feature", `@nfr
Feature: Forbidden term test
  Scenario: Custom deny term appears in step
    Given the system calls use_case_ref handler
    When the workflow completes
    Then the result is observed
`)
	vs, err := LintDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasViolationContaining(vs, `"use_case_ref"`) {
		t.Errorf("expected violation for custom deny_term, got: %v", vs)
	}
}

func TestLintDir_AllowTerms_SuppressesDenyTerm(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, `lint:
  deny_terms:
    - "use_case_ref"
  allow_terms:
    - "use_case_ref"
`)
	writeFeature(t, dir, "nfr.feature", `@nfr
Feature: Suppressed term test
  Scenario: Allowed term does not trigger violation
    Given the system calls use_case_ref handler
    When the workflow completes
    Then the result is observed
`)
	vs, err := LintDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hasViolationContaining(vs, `"use_case_ref"`) {
		t.Errorf("expected use_case_ref to be suppressed by allow_terms, got: %v", vs)
	}
}

func TestLintDir_AllowTerms_SuppressesBaseForbidden(t *testing.T) {
	dir := t.TempDir()
	// Allow a term from the package-level forbiddenPatterns.
	writeConfig(t, dir, `lint:
  allow_terms:
    - "/api/"
`)
	writeFeature(t, dir, "nfr.feature", `@nfr
Feature: Allowed base term
  Scenario: /api/ is explicitly allowed for this project
    Given the endpoint /api/ is reachable
    When a request arrives
    Then the service responds
`)
	vs, err := LintDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hasViolationContaining(vs, `"/api/"`) {
		t.Errorf("expected /api/ to be suppressed by allow_terms, got: %v", vs)
	}
}

func TestLintDir_NoConfig_UsesDefaultForbidden(t *testing.T) {
	dir := t.TempDir()
	// No .gforge.yml — default forbidden list still applies.
	writeFeature(t, dir, "nfr.feature", `@nfr
Feature: Default forbidden patterns
  Scenario: SQL leaks through
    Given the query runs SELECT id FROM orders
    When nothing happens
    Then it fails lint
`)
	vs, err := LintDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasViolationContaining(vs, "forbidden symbol") {
		t.Errorf("expected default forbidden-symbol violation, got: %v", vs)
	}
}
