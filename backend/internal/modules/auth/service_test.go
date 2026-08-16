package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

const testJWTSecret = "test-secret-with-at-least-thirty-two-characters"

func TestLoginReturnsTokenPair(t *testing.T) {
	service, tokens := newTestService(t, StatusActive)

	result, err := service.Login(context.Background(), " ADMIN@EXAMPLE.COM ", "correct-password")
	if err != nil {
		t.Fatalf("expected successful login, got %v", err)
	}
	if result.Tokens.AccessToken == "" || result.Tokens.RefreshToken == "" {
		t.Fatal("expected both access and refresh tokens")
	}
	claims, err := tokens.ParseAccess(result.Tokens.AccessToken)
	if err != nil {
		t.Fatalf("expected a valid access token, got %v", err)
	}
	if claims.Subject != result.User.ID || claims.Role != RoleAdmin {
		t.Fatalf("unexpected claims: subject=%q role=%q", claims.Subject, claims.Role)
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	service, _ := newTestService(t, StatusActive)

	_, err := service.Login(context.Background(), "admin@example.com", "wrong-password")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLoginRejectsLockedAccount(t *testing.T) {
	service, _ := newTestService(t, StatusLocked)

	_, err := service.Login(context.Background(), "admin@example.com", "correct-password")
	if !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("expected ErrAccountLocked, got %v", err)
	}
}

func TestRefreshTokenIsRotatedAndCannotBeReused(t *testing.T) {
	service, _ := newTestService(t, StatusActive)
	ctx := context.Background()

	login, err := service.Login(ctx, "admin@example.com", "correct-password")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	refreshed, err := service.Refresh(ctx, login.Tokens.RefreshToken)
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	if refreshed.Tokens.RefreshToken == login.Tokens.RefreshToken {
		t.Fatal("expected refresh token rotation")
	}

	_, err = service.Refresh(ctx, login.Tokens.RefreshToken)
	if !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("expected reused refresh token to be rejected, got %v", err)
	}
}

func TestLogoutRevokesRefreshToken(t *testing.T) {
	service, _ := newTestService(t, StatusActive)
	ctx := context.Background()
	login, err := service.Login(ctx, "admin@example.com", "correct-password")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	if err := service.Logout(ctx, login.Tokens.RefreshToken); err != nil {
		t.Fatalf("logout failed: %v", err)
	}
	_, err = service.Refresh(ctx, login.Tokens.RefreshToken)
	if !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("expected revoked refresh token to be rejected, got %v", err)
	}
}

func newTestService(t *testing.T, status string) (*Service, *TokenManager) {
	t.Helper()
	hasher := NewPasswordHasher(4)
	passwordHash, err := hasher.Hash("correct-password")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	repository := NewMemoryRepository([]User{{
		ID:           "00000000-0000-0000-0000-000000000001",
		Email:        "admin@example.com",
		PasswordHash: passwordHash,
		FullName:     "Admin User",
		EmployeeCode: "ADMIN-001",
		Role:         RoleAdmin,
		Status:       status,
		CreatedAt:    time.Now(),
	}})
	tokens, err := NewTokenManager(testJWTSecret, "test-issuer", 15*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatalf("create token manager: %v", err)
	}
	return NewService(repository, hasher, tokens), tokens
}
