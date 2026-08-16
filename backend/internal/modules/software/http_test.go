package software

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"license-manager/backend/internal/modules/auth"
)

func TestITManagerCanCreateSoftwareProduct(t *testing.T) {
	router, token := newSoftwareTestRouter(t, auth.RoleITManager)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/software",
		bytes.NewBufferString(`{"name":"Adobe Photoshop","publisher":"Adobe","version":"2026"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", response.Code, response.Body.String())
	}
}

func TestEmployeeCannotListSoftwareProducts(t *testing.T) {
	router, token := newSoftwareTestRouter(t, auth.RoleEmployee)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/software", nil)
	request.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", response.Code, response.Body.String())
	}
}

func newSoftwareTestRouter(t *testing.T, role string) (http.Handler, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	tokens, err := auth.NewTokenManager(
		"test-secret-with-at-least-thirty-two-characters",
		"test-issuer",
		15*time.Minute,
		24*time.Hour,
	)
	if err != nil {
		t.Fatalf("create token manager: %v", err)
	}
	pair, err := tokens.IssuePair(auth.User{
		ID:     "00000000-0000-0000-0000-000000000002",
		Email:  "user@example.com",
		Role:   role,
		Status: auth.StatusActive,
	})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	authRepository := auth.NewMemoryRepository(nil)
	authService := auth.NewService(authRepository, auth.NewPasswordHasher(4), tokens)
	authHandler := auth.NewHTTPHandler(authService, tokens)
	handler := NewHTTPHandler(NewService(NewMemoryRepository()), authHandler)
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api/v1"))
	return router, pair.AccessToken
}
