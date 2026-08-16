package users

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"license-manager/backend/internal/modules/auth"
)

func TestEmployeeCannotListUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	hasher := auth.NewPasswordHasher(4)
	repository := auth.NewMemoryRepository(nil)
	tokens, err := auth.NewTokenManager(
		"test-secret-with-at-least-thirty-two-characters",
		"test-issuer",
		15*time.Minute,
		24*time.Hour,
	)
	if err != nil {
		t.Fatalf("create token manager: %v", err)
	}
	employee := auth.User{
		ID:     "00000000-0000-0000-0000-000000000002",
		Email:  "employee@example.com",
		Role:   auth.RoleEmployee,
		Status: auth.StatusActive,
	}
	pair, err := tokens.IssuePair(employee)
	if err != nil {
		t.Fatalf("issue employee token: %v", err)
	}

	authService := auth.NewService(repository, hasher, tokens)
	authHandler := auth.NewHTTPHandler(authService, tokens)
	usersHandler := NewHTTPHandler(NewService(repository, hasher), authHandler)
	router := gin.New()
	usersHandler.RegisterRoutes(router.Group("/api/v1"))

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	request.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", response.Code, response.Body.String())
	}
}
