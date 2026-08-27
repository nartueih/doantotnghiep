package migrations

import (
	"strings"
	"testing"
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
