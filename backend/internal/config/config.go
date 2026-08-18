package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	AppEnv               string
	HTTPAddress          string
	StorageDriver        string
	DatabaseURL          string
	ShutdownTimeout      time.Duration
	JWTSecret            string
	JWTIssuer            string
	AccessTokenTTL       time.Duration
	RefreshTokenTTL      time.Duration
	DevAdminEmail        string
	DevAdminPassword     string
	SeedDemoData         bool
	LicenseEncryptionKey string
}

func Load() (Config, error) {
	shutdownTimeout, err := time.ParseDuration(valueOrDefault("SHUTDOWN_TIMEOUT", "10s"))
	if err != nil {
		return Config{}, errors.New("SHUTDOWN_TIMEOUT must be a valid duration")
	}

	accessTTL, err := time.ParseDuration(valueOrDefault("ACCESS_TOKEN_TTL", "15m"))
	if err != nil {
		return Config{}, errors.New("ACCESS_TOKEN_TTL must be a valid duration")
	}
	refreshTTL, err := time.ParseDuration(valueOrDefault("REFRESH_TOKEN_TTL", "168h"))
	if err != nil {
		return Config{}, errors.New("REFRESH_TOKEN_TTL must be a valid duration")
	}

	appEnv := valueOrDefault("APP_ENV", "development")
	storageDriver := valueOrDefault("STORAGE_DRIVER", "postgres")
	databaseURL := os.Getenv("DATABASE_URL")
	if storageDriver != "memory" && storageDriver != "postgres" {
		return Config{}, fmt.Errorf("unsupported STORAGE_DRIVER %q", storageDriver)
	}
	if storageDriver == "postgres" && databaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required when STORAGE_DRIVER=postgres")
	}
	if storageDriver == "memory" && appEnv == "production" {
		return Config{}, errors.New("STORAGE_DRIVER=memory is not allowed in production")
	}
	seedDemoData, err := strconv.ParseBool(valueOrDefault("SEED_DEMO_DATA", "true"))
	if err != nil {
		return Config{}, errors.New("SEED_DEMO_DATA must be true or false")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if len(jwtSecret) < 32 {
		return Config{}, errors.New("JWT_SECRET must contain at least 32 characters")
	}

	licenseEncryptionKey := os.Getenv("LICENSE_ENCRYPTION_KEY")
	if licenseEncryptionKey == "" && storageDriver == "memory" && appEnv == "development" {
		licenseEncryptionKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	}
	decodedKey, err := base64.StdEncoding.DecodeString(licenseEncryptionKey)
	if err != nil || len(decodedKey) != 32 {
		return Config{}, errors.New("LICENSE_ENCRYPTION_KEY must be a base64-encoded 32-byte key")
	}

	return Config{
		AppEnv:               appEnv,
		HTTPAddress:          valueOrDefault("HTTP_ADDRESS", ":8080"),
		StorageDriver:        storageDriver,
		DatabaseURL:          databaseURL,
		ShutdownTimeout:      shutdownTimeout,
		JWTSecret:            jwtSecret,
		JWTIssuer:            valueOrDefault("JWT_ISSUER", "license-manager"),
		AccessTokenTTL:       accessTTL,
		RefreshTokenTTL:      refreshTTL,
		DevAdminEmail:        valueOrDefault("DEV_ADMIN_EMAIL", "admin@local.test"),
		DevAdminPassword:     valueOrDefault("DEV_ADMIN_PASSWORD", "ChangeMe123!"),
		SeedDemoData:         seedDemoData,
		LicenseEncryptionKey: licenseEncryptionKey,
	}, nil
}

func valueOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
