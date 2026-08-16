package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"license-manager/backend/internal/config"
	"license-manager/backend/internal/httpapi"
	"license-manager/backend/internal/modules/assignments"
	"license-manager/backend/internal/modules/audit"
	"license-manager/backend/internal/modules/auth"
	"license-manager/backend/internal/modules/dashboard"
	"license-manager/backend/internal/modules/departments"
	"license-manager/backend/internal/modules/devices"
	"license-manager/backend/internal/modules/licenses"
	"license-manager/backend/internal/modules/software"
	"license-manager/backend/internal/modules/users"
	"license-manager/backend/internal/platform/database"
	"license-manager/backend/internal/platform/securevalue"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}
	licenseCipher, err := securevalue.NewCipher(cfg.LicenseEncryptionKey)
	if err != nil {
		logger.Error("cannot initialize license encryption", "error", err)
		os.Exit(1)
	}

	tokenManager, err := auth.NewTokenManager(
		cfg.JWTSecret,
		cfg.JWTIssuer,
		cfg.AccessTokenTTL,
		cfg.RefreshTokenTTL,
	)
	if err != nil {
		logger.Error("cannot initialize token manager", "error", err)
		os.Exit(1)
	}

	passwordHasher := auth.NewPasswordHasher(12)
	var authRepository auth.Repository
	var auditRepository audit.Repository
	var usersRepository users.Repository
	var departmentRepository departments.Repository
	var softwareRepository software.Repository
	var licenseRepository licenses.Repository
	var deviceRepository devices.Repository
	var assignmentRepository assignments.Repository
	var ping httpapi.PingFunc
	cleanup := func() {}

	switch cfg.StorageDriver {
	case "memory":
		passwordHash, hashErr := passwordHasher.Hash(cfg.DevAdminPassword)
		if hashErr != nil {
			logger.Error("cannot create development admin", "error", hashErr)
			os.Exit(1)
		}
		memoryRepository := auth.NewMemoryRepository([]auth.User{{
			ID:           "00000000-0000-0000-0000-000000000001",
			Email:        cfg.DevAdminEmail,
			PasswordHash: passwordHash,
			FullName:     "Development Admin",
			EmployeeCode: "DEV-ADMIN",
			Role:         auth.RoleAdmin,
			Status:       auth.StatusActive,
			CreatedAt:    time.Now().UTC(),
		}})
		authRepository = memoryRepository
		auditRepository = audit.NewMemoryRepository()
		usersRepository = memoryRepository
		departmentRepository = departments.NewMemoryRepository()
		softwareRepository = software.NewMemoryRepository()
		memoryLicenseRepository := licenses.NewMemoryRepository()
		licenseRepository = memoryLicenseRepository
		deviceRepository = devices.NewMemoryRepository()
		assignmentRepository = assignments.NewMemoryRepository(memoryLicenseRepository)
		ping = func(context.Context) error { return nil }
		logger.Warn("using temporary in-memory storage; data will be lost on shutdown", "admin_email", cfg.DevAdminEmail)
	case "postgres":
		startupCtx, cancelStartup := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelStartup()

		pool, openErr := database.Open(startupCtx, cfg.DatabaseURL)
		if openErr != nil {
			logger.Error("cannot connect to database", "error", openErr)
			os.Exit(1)
		}
		postgresRepository := auth.NewPostgresRepository(pool)
		authRepository = postgresRepository
		auditRepository = audit.NewPostgresRepository(pool)
		usersRepository = postgresRepository
		departmentRepository = departments.NewPostgresRepository(pool)
		softwareRepository = software.NewPostgresRepository(pool)
		licenseRepository = licenses.NewPostgresRepository(pool)
		deviceRepository = devices.NewPostgresRepository(pool)
		assignmentRepository = assignments.NewPostgresRepository(pool)
		ping = pool.Ping
		cleanup = pool.Close
	}
	defer cleanup()

	authService := auth.NewService(authRepository, passwordHasher, tokenManager)
	authHandler := auth.NewHTTPHandler(authService, tokenManager)
	auditService := audit.NewService(auditRepository, authRepository)
	auditHandler := audit.NewHTTPHandler(auditService, authHandler)
	dashboardService := dashboard.NewService(licenseRepository, deviceRepository, softwareRepository)
	dashboardHandler := dashboard.NewHTTPHandler(dashboardService, authHandler)
	usersService := users.NewService(usersRepository, passwordHasher, departmentRepository)
	usersHandler := users.NewHTTPHandler(usersService, authHandler, auditService)
	departmentService := departments.NewService(departmentRepository)
	departmentHandler := departments.NewHTTPHandler(departmentService, authHandler, auditService)
	softwareService := software.NewService(softwareRepository)
	softwareHandler := software.NewHTTPHandler(softwareService, authHandler, auditService)
	licenseService := licenses.NewService(licenseRepository, softwareRepository, licenseCipher)
	licenseHandler := licenses.NewHTTPHandler(licenseService, authHandler, auditService)
	deviceService := devices.NewService(deviceRepository, authRepository)
	deviceHandler := devices.NewHTTPHandler(deviceService, authHandler, auditService)
	assignmentService := assignments.NewService(assignmentRepository, licenseRepository, authRepository, deviceRepository)
	assignmentHandler := assignments.NewHTTPHandler(assignmentService, authHandler, auditService)

	server := &http.Server{
		Addr:              cfg.HTTPAddress,
		Handler:           httpapi.NewRouter(ping, authHandler, auditHandler, dashboardHandler, usersHandler, departmentHandler, softwareHandler, licenseHandler, deviceHandler, assignmentHandler),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("http server started", "address", cfg.HTTPAddress, "environment", cfg.AppEnv, "storage", cfg.StorageDriver)
		serverErrors <- server.ListenAndServe()
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-signals:
		logger.Info("shutdown signal received", "signal", sig.String())
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server stopped unexpectedly", "error", err)
			os.Exit(1)
		}
		return
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancelShutdown()
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	logger.Info("http server stopped")
}
