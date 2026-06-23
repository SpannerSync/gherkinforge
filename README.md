# GherkinForge

**Dual-Audience Gherkin framework for Go** — by [Rajavardhan Reddy Bathini](https://github.com/SpannerSync)

[![CI](https://github.com/spannersync/gherkinforge/actions/workflows/ci.yml/badge.svg)](https://github.com/spannersync/gherkinforge/actions)
[![Go 1.23](https://img.shields.io/badge/go-1.23-blue)](https://go.dev/doc/go1.23)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/spannersync/gherkinforge.svg)](https://pkg.go.dev/github.com/spannersync/gherkinforge)

---

## The Problem That Pushed Me to Build This

I was building a multi-tenant B2B SaaS product in Go. We had adopted BDD seriously — hundreds of Gherkin scenarios, a disciplined RED-before-GREEN workflow, Godog wired into CI.

Then we started using AI coding agents to help implement step definitions and domain code.

**The AI kept doing the same things wrong, every time:**

- Generating `float64` fields for monetary values — silently, confidently, incorrectly
- Importing `database/sql` directly into domain aggregates, destroying the hexagonal boundary we had carefully maintained
- Reading a step like `Given I click the Submit button` and generating a brittle, UI-coupled backend test
- Producing step definitions that bypassed the real service and called mocks directly — tests that always passed and proved nothing

We were spending more time fixing AI hallucinations than writing features.

The root cause was not the AI. **The root cause was the specification itself.** Our Gherkin was written for one audience — humans — and it left too much for the AI to guess.

---

## The Idea: Feature Files That Speak Two Languages

What if a `.feature` file could simultaneously be:

1. **Plain English** that a product owner or stakeholder can read and approve
2. **A precise technical contract** that an AI coding agent can parse without guessing

The dual-audience concept is not new in documentation. What GherkinForge attempts is to apply it specifically to Gherkin + AI code generation — using DataTables and DocStrings as the bridge.

**DataTable column headers become Go struct field names.**
**DocString JSON defines the exact command payload schema.**

The AI has nothing to hallucinate from.

```gherkin
@business
Feature: Order Management

  Background:
    Given the system contains the following valid product inventory:
      | product_id | sku      | unit_pence | stock |
      | P-1001     | WIDGET-X | 2999       | 50    |

  Scenario: Successfully creating an order emits a domain event
    Given a customer aggregate initialized with ID "CUST-998"
    When the customer submits a create order command with the following payload:
      """json
      {"customer_id":"CUST-998","items":[{"product_id":"P-1001","quantity":2}]}
      """
    Then the order aggregate should be successfully created
    And an "order.created" domain event is published to the broker
    And the total order value in pence should be 5998
```

The `5998` is 2 × 2999. Verifiable in the spec. An AI mutant that corrupts the calculation cannot accidentally satisfy that assertion.

---

## What GherkinForge Is (and Is Not)

**It is an experiment and a starting point** — not a finished product.

It packages three things that emerged from solving the problem above:

### 1. A Four-Tier Specification Model

```
features/
├── business/      @business  → godog + hand-written fakes
├── contract/      @contract  → pact-go consumer/provider verification
├── nfr/           @nfr       → k6 load tests + Go benchmarks + fuzz
└── draft/         @draft     → lint-only (work-in-progress, no runner yet)
```

Each tier maps to exactly one test runner. The tag is enforced by the linter — a file without a tier tag, or with two, is rejected at CI time.

`@business` and `@contract` files are also required to contain at least one DataTable or DocString. This is the structural anchor that prevents the AI from guessing schema from prose alone.

### 2. A CLI Tool (`gforge`)

```bash
# Lint feature files against dual-audience rules
gforge lint features/

# Print the tier-to-runner routing table for a directory
gforge run features/

# Scaffold hexagonal Go skeleton from a feature file
gforge scaffold --feature features/business/create_order.feature --out pkg/context/order
```

**`gforge lint`** parses the Gherkin AST and checks for:
- Missing or duplicate tier tags
- `@business` / `@contract` files without a DataTable or DocString anchor
- Forbidden implementation symbols in step text (`SELECT`, `/api/`, `.handler`, etc.)
- UI/DOM vocabulary in `@business` steps (`click`, `button`, `xpath`, `browser`, etc.) — ZERO TRUST Pillar 2

**`gforge run`** walks the directory, reads the tier tag from each `.feature` file, and prints the routing table:

```
FILE                                TIER        RUNNER
features/business/create_order.feature   @business   godog
features/contract/invoice_api.feature    @contract   pact-go
features/nfr/order_throughput.feature    @nfr        k6
features/draft/returns.feature           @draft      lint-only
```

Files with no tier tag show `unknown (run gforge lint first)`. Parse errors show `PARSE-ERROR` and the walk continues.

**`.gforge.yml` — project-level vocabulary rules**

Projects with domain-specific terminology can extend or override the built-in forbidden-word list without editing source code:

```yaml
# .gforge.yml  (place in project root or any ancestor directory)
lint:
  deny_terms:
    - "use_case_ref"    # add custom forbidden substrings
    - "repo_impl"
  allow_terms:
    - "/api/"           # suppress a built-in forbidden term for this project
```

`deny_terms` are appended to the built-in list. `allow_terms` remove matching entries from both the built-in list and any `deny_terms`. The package-level defaults are never mutated — each lint run builds its own slice.

### 3. AI Coding Agent Rules

`.cursor/rules/bdd-generation.mdc` — a constraint file for AI agents that enforces the generation sequence, the hexagonal boundary, `int64` precision for money, and a self-verify checklist before the task is declared complete.

The rules are written as **affirmative positive constraints**, not negations. The reasoning: telling an LLM "never use `database/sql`" places heavy attention on that token, which can increase the probability of hallucination. "Restrict imports exclusively to stdlib primitives" focuses the agent on the correct outcome instead.

---

## Four Guardrails We Tried

These came from asking: *what is the worst that can happen, and can we make it a pipeline failure instead of a code review comment?*

**Guardrail 1 — Compiler-level architectural boundaries**
`depguard` configuration that fails CI if domain code imports infrastructure packages. The prompt rule is not trusted alone.

**Guardrail 2 — AST-level Gherkin validation**
`gforge lint` parses the feature file abstract syntax tree. UI vocabulary in a `@business` step fails the lint gate before the AI reads the file.

**Guardrail 3 — Unconditional transaction rollback**
`@integration` godog scenarios run inside a SQL transaction that rolls back after every scenario regardless of pass/fail. No inter-scenario state contamination is possible.

**Guardrail 4 — Mutation testing**
After tests pass, a mutator introduces deliberate bugs. If the suite still passes with the mutant, the specification was not strong enough. Mathematical invariants in `Then` clauses are the primary mutation killers.

---

## Quick Start

```bash
go install github.com/spannersync/gherkinforge/cmd/gforge@latest

git clone https://github.com/SpannerSync/gherkinforge.git
cd gherkinforge
go mod tidy
go test -race -run TestFeatures ./tests/...
```

Expected:
```
3 scenarios (3 passed)
17 steps (17 passed)
--- PASS: TestFeatures
```

Lint the pilot feature files:
```bash
gforge lint features/
# ✓ No violations found.
```

Print the routing table:
```bash
gforge run features/
# FILE                                          TIER        RUNNER
# features/business/create_order.feature        @business   godog
# features/integration/order_persistence.feature @integration unknown (run gforge lint first)
# features/nfr/order_throughput.feature          @nfr        k6
```

---

## Using in Your Project

```bash
go get github.com/spannersync/gherkinforge@latest

# lint
go run github.com/spannersync/gherkinforge/cmd/gforge lint your-features/

# routing table
go run github.com/spannersync/gherkinforge/cmd/gforge run your-features/

# scaffold a new bounded context
go run github.com/spannersync/gherkinforge/cmd/gforge scaffold \
  --feature your-features/business/my_feature.feature \
  --out pkg/context/mycontext
```

Copy the Cursor rules into your project:
```bash
cp .cursor/rules/bdd-generation.mdc your-project/.cursor/rules/
cp .cursor/rules/translate-legacy-gherkin.mdc your-project/.cursor/rules/
```
The Full Pipeline (End-to-End)

New feature requirement
        ↓
1. Write @business feature file (dual-audience: DataTable + DocString JSON)
        ↓
2. gforge lint — catches UI words, missing tier tag, malformed DataTables
        ↓
3. gforge scaffold --mode goa-design — emits design/design.go from feature spec
        ↓
4. goa gen — generates HTTP/gRPC transport, OpenAPI spec (zero-drift)
        ↓
5. Cursor + bdd-generation.mdc — implements domain aggregate, ports, adapters
        ↓
6. @integration tests — each scenario wrapped in pgxephemeraltest.TxFactory.Tx()
   auto-rolls back, parallel-safe, 2ms per test
        ↓
7. go-mutesting — mutation gate blocks green CI if spec has no numeric invariants
        ↓
8. depguard — compiler blocks domain importing infrastructure packages
        ↓
SHIP ✓
---

## What We Are Not Claiming

- This is not a proven methodology. It is an approach that helped on one project.
- The AI rule files are heuristics. They reduce hallucinations — they do not eliminate them.
- The `gforge lint` word ban is an opinionated starting point. Your team's domain vocabulary may need different rules.
- Mutation testing integration is incomplete — the Makefile target documents the workflow but the CI score gate needs calibration per project.

---

## What Would Make This Better

This was built to solve a specific problem. There are almost certainly better approaches, edge cases we missed, and rules that are wrong for certain domains.

**We would genuinely like to know:**

- Does the four-tier tag model (@business / @contract / @nfr / @draft) map to how your team thinks about test layers?
- Are there Gherkin anti-patterns we missed that should be in the linter?
- Is the affirmative constraint framing for AI rules actually measurably better, or is this premature optimisation?
- What hexagonal architecture patterns in Go do you use that are not covered by the scaffold generator?
- If you tried the Translation Engine on a legacy feature file, what did it get wrong?

Open an issue, start a discussion, or open a PR. The framework is more useful as a community-shaped tool than as one team's internal convention made public.

---

## Documentation

Full documentation with citations: [github.com/SpannerSync/gherkinforge/wiki](https://github.com/SpannerSync/gherkinforge/wiki)

| Page | |
|---|---|
| [Getting Started](https://github.com/SpannerSync/gherkinforge/wiki/Getting-Started) | Install, lint, run, scaffold |
| [Four-Tier Specification Model](https://github.com/SpannerSync/gherkinforge/wiki/Four-Tier-Specification-Model) | @business / @contract / @nfr / @draft with examples |
| [Hexagonal Architecture](https://github.com/SpannerSync/gherkinforge/wiki/Hexagonal-Architecture) | Layer rules and port patterns |
| [Zero Trust Pillars](https://github.com/SpannerSync/gherkinforge/wiki/Zero-Trust-Pillars) | Four guardrails with code |
| [AI Generation Rules](https://github.com/SpannerSync/gherkinforge/wiki/AI-Generation-Rules) | Both `.mdc` files explained |
| [gforge CLI](https://github.com/SpannerSync/gherkinforge/wiki/Gforge-CLI) | lint · run · scaffold — full command reference |
| [Project Config (.gforge.yml)](https://github.com/SpannerSync/gherkinforge/wiki/Project-Config) | deny_terms / allow_terms vocabulary overrides |
| [References](https://github.com/SpannerSync/gherkinforge/wiki/References) | Citations for every rule |

---

## License

MIT — see [LICENSE](LICENSE).
