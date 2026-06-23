# GherkinForge

**Dual-Audience Gherkin framework for Go** — feature files that simultaneously serve as human-readable business requirements and deterministic anchors for AI-assisted hexagonal code generation.

[![CI](https://github.com/spannersync/gherkinforge/actions/workflows/ci.yml/badge.svg)](https://github.com/spannersync/gherkinforge/actions)
[![Go 1.23](https://img.shields.io/badge/go-1.23-blue)](https://go.dev/doc/go1.23)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

---

## The Dual-Audience Problem

Traditional Gherkin serves one audience: the product team. Technical specifications live elsewhere (tickets, ADRs, diagrams) and drift from the code.

GherkinForge resolves this by structuring every `.feature` file so it speaks to **two audiences simultaneously**:

| Audience | What they read | What they get |
|----------|---------------|---------------|
| Product owner / stakeholder | Plain English scenarios | Executable acceptance criteria |
| AI coding agent | DataTables + DocStrings | Deterministic type contracts that prevent hallucinated structs |

---

## Three-Tier Specification Model

```
features/
├── business/      @business   — godog + hand-written fakes (domain logic)
├── integration/   @integration — testcontainers-go (adapter persistence)
└── nfr/           @nfr        — Go benchmarks + fuzz (throughput/resilience)
```

Every `.feature` file carries exactly one tier tag. The `gforge lint` command enforces this.

---

## Hexagonal Architecture

```
pkg/context/<name>/
├── domain/        Pure Go — no HTTP, no SQL. Aggregates + Ports (interfaces).
├── usecases/      Orchestrates domain via ports. No framework coupling.
└── adapters/
    └── inmemory/  In-memory port implementations (swap for DB in production).
```

Domain code never imports infrastructure packages. All external dependencies flow inward through interfaces defined in `domain/ports.go`.

---

## Quick Start

### Prerequisites

- Go 1.23+
- [golangci-lint](https://golangci-lint.run/usage/install/)

### Install gforge CLI

```bash
go install github.com/spannersync/gherkinforge/cmd/gforge@latest
```

### Run the pilot BDD suite

```bash
git clone https://github.com/spannersync/gherkinforge
cd gherkinforge
go mod tidy
go test -race -run TestFeatures ./tests/...
```

Expected output:

```
Feature: Order Management
  Scenario: Successfully creating an order emits a domain event   ... passed
  Scenario: Order creation fails when no items are provided        ... passed

2 scenarios (2 passed)
--- PASS: TestFeatures
```

### Lint your feature files

```bash
gforge lint features/
# ✓ No violations found.
```

### Scaffold a new bounded context

```bash
gforge scaffold \
  --feature features/business/create_order.feature \
  --out pkg/context/shipment
```

This reads the feature file and generates:

```
pkg/context/shipment/
├── domain/
│   ├── shipment.go   (aggregate root)
│   └── ports.go      (repository + event interfaces)
├── usecases/
│   └── create_shipment.go
└── adapters/
    └── inmemory/
        └── repository.go
```

---

## The Golden Packet

`features/business/create_order.feature` is the pilot "Golden Packet" — a complete, self-documenting specification:

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

**Key design decisions:**
- `unit_pence` column: `int64` — no `float64` for money, ever.
- DocString JSON: defines the exact command payload struct; AI agents parse this, not narrative prose.
- `5998 pence` = 2 × 2999: mathematically verifiable in the spec itself.

---

## AI Agent Rules

`.cursor/rules/bdd-generation.mdc` constrains AI coding agents to:

1. Keep domain packages free of HTTP/SQL imports (hexagonal boundary).
2. Follow the scaffold sequence: domain → adapters → step defs → GREEN.
3. Derive Go types exclusively from DataTables and DocStrings — no hallucination.
4. Use `int64` pence for every monetary value.
5. Never declare complete until `go test -run TestFeatures` passes.

---

## Development

```bash
make ci          # lint-go + lint-features + test (mirrors GitHub Actions)
make bdd         # godog suite only
make scaffold-demo  # generate example skeleton into /tmp/gherkinforge-demo
```

Integration tests (requires Docker):

```bash
go test -tags=integration -race ./tests/integration/...
```

---

## Contributing

1. Fork the repository.
2. Write a failing `@business` feature file first.
3. Run `gforge lint features/` — must be clean.
4. Implement the domain, adapters, and step definitions.
5. Run `make ci` — must be green.
6. Open a pull request.

---

## License

MIT — see [LICENSE](LICENSE).
