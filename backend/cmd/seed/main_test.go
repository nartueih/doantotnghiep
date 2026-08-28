package main

import (
	"context"
	"io"
	"strings"
	"testing"
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
