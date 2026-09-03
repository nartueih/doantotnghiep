package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"license-manager/backend/internal/bootstrap"
	"license-manager/backend/internal/developmentseed"
	"license-manager/backend/internal/modules/assignments"
	"license-manager/backend/internal/modules/auth"
	"license-manager/backend/internal/modules/departments"
	"license-manager/backend/internal/modules/devices"
	"license-manager/backend/internal/modules/licenses"
	"license-manager/backend/internal/modules/software"
	"license-manager/backend/internal/modules/users"
	"license-manager/backend/internal/platform/database"
	"license-manager/backend/internal/platform/securevalue"
	"license-manager/backend/migrations"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
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
	seedDemo, encryptionKey, err := demoSeedConfiguration(getenv)
	if err != nil {
		return err
	}

	pool, err := database.Open(ctx, databaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := migrations.RequireCurrent(ctx, pool); err != nil {
		return err
	}

	authRepository := auth.NewPostgresRepository(pool)
	passwordHasher := auth.NewPasswordHasher(12)
	result, err := bootstrap.EnsureAdmin(
		ctx,
		authRepository,
		passwordHasher,
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
	if _, err := fmt.Fprintf(output, "Admin %s: %s (%s)\n", state, result.User.Email, result.User.EmployeeCode); err != nil {
		return err
	}
	if !seedDemo {
		return nil
	}

	licenseCipher, err := securevalue.NewCipher(encryptionKey)
	if err != nil {
		return fmt.Errorf("initialize license encryption: %w", err)
	}
	departmentRepository := departments.NewPostgresRepository(pool)
	softwareRepository := software.NewPostgresRepository(pool)
	licenseRepository := licenses.NewPostgresRepository(pool)
	deviceRepository := devices.NewPostgresRepository(pool)
	assignmentRepository := assignments.NewPostgresRepository(pool)

	return seedDemoData(
		ctx,
		output,
		authRepository,
		developmentseed.Services{
			Departments: departments.NewService(departmentRepository),
			Users:       users.NewService(authRepository, passwordHasher, departmentRepository),
			Software:    software.NewService(softwareRepository),
			Licenses:    licenses.NewService(licenseRepository, softwareRepository, licenseCipher),
			Devices:     devices.NewService(deviceRepository, authRepository),
			Assignments: assignments.NewService(assignmentRepository, licenseRepository, authRepository, deviceRepository),
		},
		result.User.ID,
		time.Now().UTC(),
		developmentseed.Seed,
	)
}

func demoSeedConfiguration(getenv func(string) string) (bool, string, error) {
	rawEnabled := getenv("SEED_DEMO_DATA")
	if rawEnabled == "" {
		return false, "", nil
	}
	enabled, err := strconv.ParseBool(rawEnabled)
	if err != nil {
		return false, "", errors.New("SEED_DEMO_DATA must be true or false")
	}
	if !enabled {
		return false, "", nil
	}
	encryptionKey := getenv("LICENSE_ENCRYPTION_KEY")
	if _, err := securevalue.NewCipher(encryptionKey); err != nil {
		return false, "", errors.New("LICENSE_ENCRYPTION_KEY must be a base64-encoded 32-byte key")
	}
	return true, encryptionKey, nil
}

type demoUserFinder interface {
	FindByEmail(context.Context, string) (auth.User, error)
}

type demoSeedFunc func(
	context.Context,
	developmentseed.Services,
	string,
	time.Time,
) (developmentseed.Result, error)

func seedDemoData(
	ctx context.Context,
	output io.Writer,
	users demoUserFinder,
	services developmentseed.Services,
	actorID string,
	now time.Time,
	seed demoSeedFunc,
) error {
	const markerEmail = "anh.nguyen@local.test"
	if _, err := users.FindByEmail(ctx, markerEmail); err != nil && !errors.Is(err, auth.ErrUserNotFound) {
		return fmt.Errorf("check existing demo data: %w", err)
	}

	result, err := seed(ctx, services, actorID, now)
	if err != nil {
		return fmt.Errorf("seed demo data: %w", err)
	}
	_, err = fmt.Fprintf(
		output,
		"Demo data synchronized: departments=%d users=%d software=%d licenses=%d devices=%d assignments=%d\n",
		result.Departments,
		result.Users,
		result.Software,
		result.Licenses,
		result.Devices,
		result.Assignments,
	)
	return err
}
