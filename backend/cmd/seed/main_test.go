package main

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"license-manager/backend/internal/developmentseed"
	"license-manager/backend/internal/modules/auth"
)

func TestRunRequiresDatabaseAndAdminCredentials(t *testing.T) {
	tests := []struct {
		name     string
		values   map[string]string
		expected string
	}{
		{name: "database", values: map[string]string{}, expected: "DATABASE_URL"},
		{name: "email", values: map[string]string{"DATABASE_URL": "postgres://unused"}, expected: "DEV_ADMIN_EMAIL"},
		{name: "password", values: map[string]string{
			"DATABASE_URL": "postgres://unused", "DEV_ADMIN_EMAIL": "admin@local.test",
		}, expected: "DEV_ADMIN_PASSWORD"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := run(context.Background(), func(key string) string { return test.values[key] }, io.Discard)
			if err == nil || !strings.Contains(err.Error(), test.expected) {
				t.Fatalf("expected %s error, got %v", test.expected, err)
			}
		})
	}
}

func TestDemoSeedConfigurationRequiresValidFlagAndEncryptionKey(t *testing.T) {
	validKey := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	tests := []struct {
		name        string
		values      map[string]string
		wantEnabled bool
		wantError   string
	}{
		{name: "disabled by default", values: map[string]string{}},
		{name: "explicitly disabled", values: map[string]string{"SEED_DEMO_DATA": "false"}},
		{name: "invalid flag", values: map[string]string{"SEED_DEMO_DATA": "sometimes"}, wantError: "SEED_DEMO_DATA"},
		{name: "missing key", values: map[string]string{"SEED_DEMO_DATA": "true"}, wantError: "LICENSE_ENCRYPTION_KEY"},
		{name: "enabled", values: map[string]string{
			"SEED_DEMO_DATA": "true", "LICENSE_ENCRYPTION_KEY": validKey,
		}, wantEnabled: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			enabled, _, err := demoSeedConfiguration(func(key string) string { return test.values[key] })
			if test.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantError) {
					t.Fatalf("expected %s error, got %v", test.wantError, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("load demo seed configuration: %v", err)
			}
			if enabled != test.wantEnabled {
				t.Fatalf("enabled=%v, want %v", enabled, test.wantEnabled)
			}
		})
	}
}

func TestSeedDemoDataCreatesDatasetWhenMarkerUserDoesNotExist(t *testing.T) {
	called := false
	var output strings.Builder
	err := seedDemoData(
		context.Background(),
		&output,
		stubDemoUserFinder{err: auth.ErrUserNotFound},
		developmentseed.Services{},
		"admin-id",
		time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC),
		func(_ context.Context, _ developmentseed.Services, actorID string, _ time.Time) (developmentseed.Result, error) {
			called = true
			if actorID != "admin-id" {
				t.Fatalf("actorID=%q", actorID)
			}
			return developmentseed.Result{
				Departments: 3, Users: 5, Software: 6, Licenses: 6, Devices: 6, Assignments: 14,
			}, nil
		},
	)

	if err != nil {
		t.Fatalf("seed demo data: %v", err)
	}
	if !called {
		t.Fatal("expected demo seeder to be called")
	}
	if !strings.Contains(output.String(), "assignments=14") {
		t.Fatalf("unexpected output: %q", output.String())
	}
}

func TestSeedDemoDataResumesExistingDataset(t *testing.T) {
	called := false
	var output strings.Builder
	err := seedDemoData(
		context.Background(),
		&output,
		stubDemoUserFinder{user: auth.User{Email: "anh.nguyen@local.test"}},
		developmentseed.Services{},
		"admin-id",
		time.Now(),
		func(context.Context, developmentseed.Services, string, time.Time) (developmentseed.Result, error) {
			called = true
			return developmentseed.Result{Assignments: 14}, nil
		},
	)

	if err != nil {
		t.Fatalf("skip existing demo data: %v", err)
	}
	if !called {
		t.Fatal("demo seeder must resume when marker user exists")
	}
	if !strings.Contains(output.String(), "assignments=14") {
		t.Fatalf("unexpected output: %q", output.String())
	}
}

type stubDemoUserFinder struct {
	user auth.User
	err  error
}

func (f stubDemoUserFinder) FindByEmail(context.Context, string) (auth.User, error) {
	return f.user, f.err
}
