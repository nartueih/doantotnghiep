package main

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestRunRejectsMissingDatabaseURL(t *testing.T) {
	err := run(context.Background(), []string{"status"}, func(string) string { return "" }, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("expected DATABASE_URL error, got %v", err)
	}
}

func TestRunRejectsUnknownCommandBeforeOpeningDatabase(t *testing.T) {
	err := run(context.Background(), []string{"down"}, func(string) string {
		return "postgres://unused"
	}, io.Discard)
	if err == nil || !errors.Is(err, errUsage) {
		t.Fatalf("expected usage error, got %v", err)
	}
}
