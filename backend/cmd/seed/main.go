package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"license-manager/backend/internal/bootstrap"
	"license-manager/backend/internal/modules/auth"
	"license-manager/backend/internal/platform/database"
	"license-manager/backend/migrations"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := run(ctx, os.Getenv, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, getenv func(string) string, output io.Writer) error {
	databaseURL := getenv("DATABASE_URL")
	if databaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	adminEmail := getenv("DEV_ADMIN_EMAIL")
	if adminEmail == "" {
		return errors.New("DEV_ADMIN_EMAIL is required")
	}
	adminPassword := getenv("DEV_ADMIN_PASSWORD")
	if adminPassword == "" {
		return errors.New("DEV_ADMIN_PASSWORD is required")
	}

	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := migrations.RequireCurrent(ctx, pool); err != nil {
		return err
	}

	result, err := bootstrap.EnsureAdmin(
		ctx,
		auth.NewPostgresRepository(pool),
		auth.NewPasswordHasher(12),
		bootstrap.AdminInput{
			Email: adminEmail, Password: adminPassword,
			FullName: "Development Admin", EmployeeCode: "DEV-ADMIN",
		},
	)
	if err != nil {
		return err
	}
	state := "already exists"
	if result.Created {
		state = "created"
	}
	_, err = fmt.Fprintf(output, "Admin %s: %s (%s)\n", state, result.User.Email, result.User.EmployeeCode)
	return err
}
