package selfservice

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"license-manager/backend/internal/modules/auth"
)

func TestQueryCannotOverrideAuthenticatedUser(t *testing.T) {
	router, token := newSelfServiceHTTPTestRouter(t)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me/devices?user_id=user-2", nil)
	request.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "USER-1-ASSET") || strings.Contains(response.Body.String(), "OTHER-ASSET") {
		t.Fatalf("response did not stay scoped to authenticated user: %s", response.Body.String())
	}
}

func TestSelfServiceRequiresAuthentication(t *testing.T) {
	router, _ := newSelfServiceHTTPTestRouter(t)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me/licenses", nil)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", response.Code, response.Body.String())
	}
}

func newSelfServiceHTTPTestRouter(t *testing.T) (http.Handler, string) {
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
		ID: "user-1", Email: "employee@example.com", Role: auth.RoleEmployee, Status: auth.StatusActive,
	})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	authService := auth.NewService(auth.NewMemoryRepository(nil), auth.NewPasswordHasher(4), tokens)
	authHandler := auth.NewHTTPHandler(authService, tokens)
	handler := NewHTTPHandler(newSelfServiceTestService(t), authHandler)
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api/v1"))
	return router, pair.AccessToken
}
