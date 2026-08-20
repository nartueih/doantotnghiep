package users

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"license-manager/backend/internal/modules/auth"
)

func TestITManagerCanListButCannotCreateUsers(t *testing.T) {
	router, token := newUserHTTPTestRouter(t, auth.RoleITManager)

	listResponse := httptest.NewRecorder()
	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	listRequest.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected list status 200, got %d: %s", listResponse.Code, listResponse.Body.String())
	}

	createResponse := httptest.NewRecorder()
	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString(`{
		"email":"new@example.com","password":"StrongPass123","full_name":"New User",
		"employee_code":"NEW-001","role":"employee"
	}`))
	createRequest.Header.Set("Content-Type", "application/json")
	createRequest.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusForbidden {
		t.Fatalf("expected create status 403, got %d: %s", createResponse.Code, createResponse.Body.String())
	}
}

func TestEmployeeCannotListUsers(t *testing.T) {
	router, token := newUserHTTPTestRouter(t, auth.RoleEmployee)

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", response.Code, response.Body.String())
	}
}

func newUserHTTPTestRouter(t *testing.T, role string) (http.Handler, string) {
	t.Helper()
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
	caller := auth.User{
		ID:     "00000000-0000-0000-0000-000000000002",
		Email:  "caller@example.com",
		Role:   role,
		Status: auth.StatusActive,
	}
	pair, err := tokens.IssuePair(caller)
	if err != nil {
		t.Fatalf("issue caller token: %v", err)
	}

	authService := auth.NewService(repository, hasher, tokens)
	authHandler := auth.NewHTTPHandler(authService, tokens)
	usersHandler := NewHTTPHandler(NewService(repository, hasher), authHandler)
	router := gin.New()
	usersHandler.RegisterRoutes(router.Group("/api/v1"))

	return router, pair.AccessToken
}
