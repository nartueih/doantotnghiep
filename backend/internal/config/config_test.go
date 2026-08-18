package config

import "testing"

func TestLoadRequiresDatabaseURLForPostgres(t *testing.T) {
	t.Setenv("STORAGE_DRIVER", "postgres")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "01234567890123456789012345678901")
	t.Setenv("LICENSE_ENCRYPTION_KEY", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")

	_, err := Load()
	if err == nil {
		t.Fatal("expected an error when DATABASE_URL is empty")
	}
}

func TestLoadUsesDefaults(t *testing.T) {
	t.Setenv("STORAGE_DRIVER", "memory")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("JWT_SECRET", "01234567890123456789012345678901")
	t.Setenv("LICENSE_ENCRYPTION_KEY", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("HTTP_ADDRESS", "")
	t.Setenv("SHUTDOWN_TIMEOUT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected valid configuration, got %v", err)
	}

	if cfg.AppEnv != "development" {
		t.Fatalf("expected development environment, got %q", cfg.AppEnv)
	}
	if cfg.HTTPAddress != ":8080" {
		t.Fatalf("expected :8080 address, got %q", cfg.HTTPAddress)
	}
	if cfg.AccessTokenTTL.String() != "15m0s" {
		t.Fatalf("expected 15m access token TTL, got %s", cfg.AccessTokenTTL)
	}
	if !cfg.SeedDemoData {
		t.Fatal("expected demo seed data to be enabled by default")
	}
}

func TestLoadCanDisableDemoSeedData(t *testing.T) {
	t.Setenv("STORAGE_DRIVER", "memory")
	t.Setenv("JWT_SECRET", "01234567890123456789012345678901")
	t.Setenv("SEED_DEMO_DATA", "false")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected valid configuration, got %v", err)
	}
	if cfg.SeedDemoData {
		t.Fatal("expected demo seed data to be disabled")
	}
}

func TestLoadRejectsInvalidDemoSeedData(t *testing.T) {
	t.Setenv("STORAGE_DRIVER", "memory")
	t.Setenv("JWT_SECRET", "01234567890123456789012345678901")
	t.Setenv("SEED_DEMO_DATA", "sometimes")

	_, err := Load()
	if err == nil {
		t.Fatal("expected invalid SEED_DEMO_DATA to be rejected")
	}
}

func TestLoadRejectsMemoryStorageInProduction(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("STORAGE_DRIVER", "memory")
	t.Setenv("JWT_SECRET", "01234567890123456789012345678901")
	t.Setenv("LICENSE_ENCRYPTION_KEY", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")

	_, err := Load()
	if err == nil {
		t.Fatal("expected memory storage to be rejected in production")
	}
}
