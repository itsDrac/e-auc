package tests

import (
	"context"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool" // Import pgxpool
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// SetupTestDB spins up a real Postgres container and returns a pgx connection pool
func SetupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	ctx := context.Background()

	// 1. Start the Container
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		t.Fatalf("failed to start container: %s", err)
	}

	// 2. Get Connection String
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %s", err)
	}

	// 3. Run Migrations
	// Note: 'golang-migrate' uses its own internal drivers, so passing the standard
	// postgres connection string here works fine even if your app uses pgx.
	m, err := migrate.New("file://../../migrations", connStr)
	if err != nil {
		t.Fatalf("failed to create migrate instance: %s", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("failed to run migrations: %s", err)
	}

	// 4. Open Connection with PGX (The Key Change)
	// We use pgxpool to simulate a real production environment connection
	dbPool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		t.Fatalf("failed to create pgx pool: %s", err)
	}

	// Optional: Ping to ensure connection is actually alive
	if err := dbPool.Ping(ctx); err != nil {
		t.Fatalf("failed to ping database: %s", err)
	}

	// 5. Cleanup Closure
	cleanup := func() {
		dbPool.Close() // Close the pgx pool
		m.Close()      // Close the migration source
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err)
		}
	}

	return dbPool, cleanup
}
