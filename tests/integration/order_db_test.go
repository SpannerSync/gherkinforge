//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestOrderPersistenceRoundTrip validates that the order adapter correctly
// persists and retrieves an order from a real PostgreSQL instance.
// Run with: go test -tags=integration ./tests/integration/...
func TestOrderPersistenceRoundTrip(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

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
			WithStartupTimeout(60 * time.Second),
	}

	pg, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("starting postgres container: %v", err)
	}
	defer func() { _ = pg.Terminate(ctx) }()

	host, err := pg.Host(ctx)
	if err != nil {
		t.Fatalf("getting container host: %v", err)
	}
	port, err := pg.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("getting mapped port: %v", err)
	}

	t.Logf("PostgreSQL ready at %s:%s — wire a pgx adapter here", host, port.Port())

	// TODO: wire the pgx-backed OrderRepository adapter and run
	// Save/FindByID assertions against the live container.
	// This stub verifies that testcontainers-go spins up successfully.
}
