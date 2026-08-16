package dashboard

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"license-manager/backend/internal/modules/auth"
	"license-manager/backend/internal/modules/devices"
	"license-manager/backend/internal/modules/licenses"
	"license-manager/backend/internal/modules/software"
)

func TestITManagerCanViewDashboardSummary(t *testing.T) {
	router, token := newDashboardHTTPTestRouter(t, auth.RoleITManager)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/summary", nil)
	request.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", response.Code, response.Body.String())
	}
}

func TestEmployeeCannotViewDashboard(t *testing.T) {
	router, token := newDashboardHTTPTestRouter(t, auth.RoleEmployee)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/license-alerts?days=30", nil)
	request.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", response.Code, response.Body.String())
	}
}

func TestDashboardRejectsInvalidExpiryWindow(t *testing.T) {
	router, token := newDashboardHTTPTestRouter(t, auth.RoleAdmin)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/dashboard/license-alerts?days=45", nil)
	request.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", response.Code, response.Body.String())
	}
}

func newDashboardHTTPTestRouter(t *testing.T, role string) (http.Handler, string) {
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
		ID: "00000000-0000-0000-0000-000000000002", Email: "user@example.com", Role: role, Status: auth.StatusActive,
	})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	authService := auth.NewService(auth.NewMemoryRepository(nil), auth.NewPasswordHasher(4), tokens)
	authHandler := auth.NewHTTPHandler(authService, tokens)
	service := NewService(licenses.NewMemoryRepository(), devices.NewMemoryRepository(), software.NewMemoryRepository())
	handler := NewHTTPHandler(service, authHandler)
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api/v1"))
	return router, pair.AccessToken
}
