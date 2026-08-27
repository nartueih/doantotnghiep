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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
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
