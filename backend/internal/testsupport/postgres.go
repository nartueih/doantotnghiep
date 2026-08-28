package testsupport

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"license-manager/backend/internal/platform/database"
	"license-manager/backend/migrations"

	"github.com/jackc/pgx/v5/pgxpool"
)

const postgresTestAdvisoryLock int64 = 720240827

func OpenPostgres(t testing.TB) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse TEST_DATABASE_URL: %v", err)
	}
	if !strings.HasSuffix(config.ConnConfig.Database, "_test") {
		t.Fatalf("refusing to modify non-test database %q", config.ConnConfig.Database)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	acquirePostgresTestLock(t, ctx, databaseURL)

	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open PostgreSQL test database: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := migrations.Up(ctx, pool); err != nil {
		t.Fatalf("migrate PostgreSQL test database: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		TRUNCATE TABLE
			notifications,
			license_requests,
			audit_logs,
			license_assignments,
			refresh_tokens,
			licenses,
			devices,
			software_products,
			users,
			departments
		RESTART IDENTITY CASCADE
	`); err != nil {
		t.Fatalf("reset PostgreSQL test database: %v", err)
	}
	return pool
}

func acquirePostgresTestLock(t testing.TB, ctx context.Context, databaseURL string) {
	t.Helper()
	lockPool, err := database.Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open PostgreSQL test lock connection: %v", err)
	}
	connection, err := lockPool.Acquire(ctx)
	if err != nil {
		lockPool.Close()
		t.Fatalf("acquire PostgreSQL test lock connection: %v", err)
	}
	if _, err := connection.Exec(ctx, "SELECT pg_advisory_lock($1)", postgresTestAdvisoryLock); err != nil {
		connection.Release()
		lockPool.Close()
		t.Fatalf("acquire PostgreSQL test lock: %v", err)
	}

	t.Cleanup(func() {
		unlockContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := connection.Exec(unlockContext, "SELECT pg_advisory_unlock($1)", postgresTestAdvisoryLock); err != nil {
			t.Errorf("release PostgreSQL test lock: %v", err)
		}
		connection.Release()
		lockPool.Close()
	})
}
