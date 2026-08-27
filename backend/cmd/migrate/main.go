package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"license-manager/backend/internal/platform/database"
	"license-manager/backend/migrations"
)

var errUsage = errors.New("usage: go run ./cmd/migrate <up|status>")

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := run(ctx, os.Args[1:], os.Getenv, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(
	ctx context.Context,
	args []string,
	getenv func(string) string,
	output io.Writer,
) error {
	if len(args) != 1 || (args[0] != "up" && args[0] != "status") {
		return errUsage
	}

	databaseURL := getenv("DATABASE_URL")
	if databaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}

	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if args[0] == "up" {
		if err := migrations.Up(ctx, pool); err != nil {
			return err
		}
	}

	statuses, err := migrations.Status(ctx, pool)
	if err != nil {
		return err
	}
	for _, status := range statuses {
		state := "pending"
		if status.AppliedAt != nil {
			state = "applied " + status.AppliedAt.UTC().Format(time.RFC3339)
		}
		if _, err := fmt.Fprintf(output, "%03d %-48s %s\n", status.Version, status.Name, state); err != nil {
			return fmt.Errorf("write migration status: %w", err)
		}
	}
	return nil
}
