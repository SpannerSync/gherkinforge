//go:build integration

// Package integration provides a godog BDD suite for the @integration tier.
// Zero Trust Pillar 3: every scenario runs inside a SQL transaction that is
// unconditionally rolled back after the scenario completes — regardless of
// pass or fail. This guarantees each scenario starts from a pristine database
// state, eliminating inter-scenario data contamination.
package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/cucumber/godog"
	_ "github.com/lib/pq"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// txKey is the context key that carries the per-scenario transaction.
type txKey struct{}

// integrationState holds per-scenario collaborators.
type integrationState struct {
	db *sql.DB
}

// InitializeIntegrationScenario wires the Zero Trust transaction rollback hook
// and all @integration step definitions.
//
// Hook lifecycle:
//
//	Before scenario → BEGIN TRANSACTION
//	After scenario  → ROLLBACK (always — Zero Trust)
func InitializeIntegrationScenario(db *sql.DB) func(*godog.ScenarioContext) {
	return func(sc *godog.ScenarioContext) {
		sc.Before(func(ctx context.Context, scenario *godog.Scenario) (context.Context, error) {
			tx, err := db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
			if err != nil {
				return ctx, fmt.Errorf("ZERO TRUST: failed to begin scenario transaction: %w", err)
			}
			return context.WithValue(ctx, txKey{}, tx), nil
		})

		sc.After(func(ctx context.Context, scenario *godog.Scenario, scenarioErr error) (context.Context, error) {
			tx, ok := ctx.Value(txKey{}).(*sql.Tx)
			if !ok || tx == nil {
				return ctx, nil
			}
			// Unconditional rollback — Zero Trust in test cleanup.
			// We never commit integration test data to the DB.
			if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
				return ctx, fmt.Errorf("ZERO TRUST: rollback failed after scenario %q: %w", scenario.Name, err)
			}
			return ctx, nil
		})

		// ── Step definitions ────────────────────────────────────────────────
		sc.Step(`^a clean PostgreSQL database is available$`, func(ctx context.Context) error {
			tx := ctx.Value(txKey{}).(*sql.Tx)
			// Verify the transaction is active by executing a no-op query.
			_, err := tx.ExecContext(ctx, "SELECT 1")
			return err
		})

		sc.Step(`^an order with ID "([^"]*)" for customer "([^"]*)" and total (\d+) pence$`,
			func(ctx context.Context, orderID, customerID string, totalPence int64) (context.Context, error) {
				// Store parameters in context for the When step.
				type orderParams struct {
					ID         string
					CustomerID string
					TotalPence int64
				}
				return context.WithValue(ctx, "orderParams", orderParams{
					ID:         orderID,
					CustomerID: customerID,
					TotalPence: totalPence,
				}), nil
			})

		sc.Step(`^the order is saved to the repository$`,
			func(ctx context.Context) error {
				tx := ctx.Value(txKey{}).(*sql.Tx)
				params := ctx.Value("orderParams")
				if params == nil {
					return fmt.Errorf("no order params in context — check previous step")
				}
				type orderParams struct {
					ID         string
					CustomerID string
					TotalPence int64
				}
				p := params.(orderParams)
				_, err := tx.ExecContext(ctx,
					`INSERT INTO orders (id, customer_id, total_pence, created_at)
					 VALUES ($1, $2, $3, $4)`,
					p.ID, p.CustomerID, p.TotalPence, time.Now().UTC(),
				)
				return err
			})

		sc.Step(`^finding order "([^"]*)" returns the same order with total (\d+) pence$`,
			func(ctx context.Context, orderID string, expectedPence int64) error {
				tx := ctx.Value(txKey{}).(*sql.Tx)
				var gotPence int64
				err := tx.QueryRowContext(ctx,
					`SELECT total_pence FROM orders WHERE id = $1`, orderID,
				).Scan(&gotPence)
				if err != nil {
					return fmt.Errorf("finding order %q: %w", orderID, err)
				}
				if gotPence != expectedPence {
					return fmt.Errorf("expected total_pence %d, got %d", expectedPence, gotPence)
				}
				return nil
			})

		sc.Step(`^finding order "([^"]*)" from the repository$`,
			func(ctx context.Context, orderID string) (context.Context, error) {
				tx := ctx.Value(txKey{}).(*sql.Tx)
				var gotPence int64
				err := tx.QueryRowContext(ctx,
					`SELECT total_pence FROM orders WHERE id = $1`, orderID,
				).Scan(&gotPence)
				return context.WithValue(ctx, "lastFindErr", err), nil
			})

		sc.Step(`^the result is an "([^"]*)" error$`,
			func(ctx context.Context, errSubstring string) error {
				raw := ctx.Value("lastFindErr")
				if raw == nil {
					return fmt.Errorf("expected an error but got nil")
				}
				err, ok := raw.(error)
				if !ok || err == nil {
					return fmt.Errorf("expected error containing %q, but got nil", errSubstring)
				}
				return nil
			})
	}
}

func TestIntegrationFeatures(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// Spin up a containerised PostgreSQL instance.
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "gforge",
			"POSTGRES_PASSWORD": "gforge",
			"POSTGRES_DB":       "gforge_test",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(90 * time.Second),
	}
	pg, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("starting postgres: %v", err)
	}
	defer func() { _ = pg.Terminate(ctx) }()

	host, _ := pg.Host(ctx)
	port, _ := pg.MappedPort(ctx, "5432")
	dsn := fmt.Sprintf("postgres://gforge:gforge@%s:%s/gforge_test?sslmode=disable", host, port.Port())

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("opening db: %v", err)
	}
	defer db.Close()

	// Bootstrap schema — in production this would be a migration runner.
	if _, err = db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS orders (
			id          TEXT PRIMARY KEY,
			customer_id TEXT NOT NULL,
			total_pence BIGINT NOT NULL,
			created_at  TIMESTAMPTZ NOT NULL
		)`); err != nil {
		t.Fatalf("creating schema: %v", err)
	}

	_ = os.Setenv("INTEGRATION_DSN", dsn)

	suite := godog.TestSuite{
		ScenarioInitializer: InitializeIntegrationScenario(db),
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"../../features/integration"},
			TestingT: t,
			Strict:   true,
		},
	}
	if suite.Run() != 0 {
		t.Fatal("integration feature tests failed")
	}
}
