package migrations

import (
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAllReturnsOrderedUniqueMigrations(t *testing.T) {
	items := All()
	if len(items) != 4 {
		t.Fatalf("expected 4 migrations, got %d", len(items))
	}

	seen := make(map[int]bool, len(items))
	for index, item := range items {
		expectedVersion := index + 1
		if item.Version != expectedVersion {
			t.Fatalf("migration at index %d has version %d, expected %d", index, item.Version, expectedVersion)
		}
		if seen[item.Version] {
			t.Fatalf("duplicate migration version %d", item.Version)
		}
		seen[item.Version] = true
		if strings.TrimSpace(item.Name) == "" || strings.TrimSpace(item.SQL) == "" {
			t.Fatalf("migration %d has blank name or SQL", item.Version)
		}
	}

	if LatestVersion != items[len(items)-1].Version {
		t.Fatalf("latest version is %d, last migration is %d", LatestVersion, items[len(items)-1].Version)
	}
}

func TestParseVersionRejectsInvalidFilename(t *testing.T) {
	if _, err := parseVersion("license_requests.sql"); err == nil {
		t.Fatal("expected an invalid migration filename error")
	}
}

func TestUpStatusAndRequireCurrent(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		t.Fatalf("parse test database URL: %v", err)
	}
	if !strings.HasSuffix(config.ConnConfig.Database, "_test") {
		t.Fatalf("refusing to migrate non-test database %q", config.ConnConfig.Database)
	}

	pool, err := pgxpool.NewWithConfig(t.Context(), config)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := Up(t.Context(), pool); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}

	statuses, err := Status(t.Context(), pool)
	if err != nil {
		t.Fatalf("load migration status: %v", err)
	}
	if len(statuses) != LatestVersion {
		t.Fatalf("got %d migration statuses, want %d", len(statuses), LatestVersion)
	}
	for _, status := range statuses {
		if status.AppliedAt == nil {
			t.Fatalf("migration %03d is not applied", status.Version)
		}
	}

	if err := RequireCurrent(t.Context(), pool); err != nil {
		t.Fatalf("require current schema: %v", err)
	}
}

func TestRequireVersionRejectsMissingAndStaleSchemas(t *testing.T) {
	for _, current := range []int{0, 3} {
		err := requireVersion(current)
		if err == nil {
			t.Fatalf("version %d unexpectedly accepted", current)
		}
		if !strings.Contains(err.Error(), "expected 4") || !strings.Contains(err.Error(), "cmd/migrate up") {
			t.Fatalf("version %d returned unclear error: %v", current, err)
		}
	}
	if err := requireVersion(LatestVersion); err != nil {
		t.Fatalf("current schema rejected: %v", err)
	}
}
