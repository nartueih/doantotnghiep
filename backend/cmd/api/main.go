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
	"license-manager/backend/internal/developmentseed"
	"license-manager/backend/internal/httpapi"
	"license-manager/backend/internal/modules/assignments"
	"license-manager/backend/internal/modules/audit"
	"license-manager/backend/internal/modules/auth"
	"license-manager/backend/internal/modules/dashboard"
	"license-manager/backend/internal/modules/departments"
	"license-manager/backend/internal/modules/devices"
	"license-manager/backend/internal/modules/licenserequests"
	"license-manager/backend/internal/modules/licenses"
	"license-manager/backend/internal/modules/notifications"
	"license-manager/backend/internal/modules/selfservice"
	"license-manager/backend/internal/modules/software"
	"license-manager/backend/internal/modules/users"
	"license-manager/backend/internal/platform/database"
	"license-manager/backend/internal/platform/securevalue"
)

const developmentAdminID = "00000000-0000-0000-0000-000000000001"

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
	var notificationRepository notifications.Repository
	var licenseRequestRepository licenserequests.Repository
	var transactionManager database.Transactor
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
			ID:           developmentAdminID,
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
		notificationRepository = notifications.NewMemoryRepository()
		licenseRequestRepository = licenserequests.NewMemoryRepository()
		transactionManager = database.DirectTransactor{}
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
		notificationRepository = notifications.NewPostgresRepository(pool)
		licenseRequestRepository = licenserequests.NewPostgresRepository(pool)
		transactionManager = database.NewPostgresTransactor(pool)
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
	selfService := selfservice.NewService(deviceRepository, assignmentRepository, licenseService, licenseService)
	selfServiceHandler := selfservice.NewHTTPHandler(selfService, authHandler, auditService)
	deviceService := devices.NewService(deviceRepository, authRepository)
	deviceHandler := devices.NewHTTPHandler(deviceService, authHandler, auditService)
	assignmentService := assignments.NewService(assignmentRepository, licenseRepository, authRepository, deviceRepository)
	assignmentHandler := assignments.NewHTTPHandler(assignmentService, authHandler, auditService)
	notificationService := notifications.NewService(notificationRepository)
	notificationHandler := notifications.NewHTTPHandler(notificationService, authHandler)
	licenseRequestService := licenserequests.NewService(
		licenseRequestRepository,
		softwareRepository,
		licenseRepository,
		authRepository,
		assignmentService,
		notificationService,
		transactionManager,
	)
	licenseRequestHandler := licenserequests.NewHTTPHandler(licenseRequestService, authHandler, auditService)

	if cfg.StorageDriver == "memory" && cfg.SeedDemoData {
		seedResult, seedErr := developmentseed.Seed(context.Background(), developmentseed.Services{
			Departments: departmentService,
			Users:       usersService,
			Software:    softwareService,
			Licenses:    licenseService,
			Devices:     deviceService,
			Assignments: assignmentService,
		}, developmentAdminID, time.Now())
		if seedErr != nil {
			logger.Error("cannot seed development demo data", "error", seedErr)
			os.Exit(1)
		}
		logger.Info(
			"development demo data seeded",
			"departments", seedResult.Departments,
			"users", seedResult.Users,
			"software", seedResult.Software,
			"licenses", seedResult.Licenses,
			"devices", seedResult.Devices,
			"assignments", seedResult.Assignments,
		)
	}

	server := &http.Server{
		Addr:              cfg.HTTPAddress,
		Handler:           httpapi.NewRouter(ping, authHandler, auditHandler, dashboardHandler, selfServiceHandler, notificationHandler, licenseRequestHandler, usersHandler, departmentHandler, softwareHandler, licenseHandler, deviceHandler, assignmentHandler),
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
