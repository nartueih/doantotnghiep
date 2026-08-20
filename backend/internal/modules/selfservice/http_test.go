package selfservice

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"license-manager/backend/internal/modules/audit"
	"license-manager/backend/internal/modules/auth"
)

func TestQueryCannotOverrideAuthenticatedUser(t *testing.T) {
	router, token, _ := newSelfServiceHTTPTestRouter(t)
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
	router, _, _ := newSelfServiceHTTPTestRouter(t)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me/licenses", nil)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", response.Code, response.Body.String())
	}
}

func TestEmployeeRevealKeyCreatesAuditLog(t *testing.T) {
	router, token, auditService := newSelfServiceHTTPTestRouter(t)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me/licenses/assignment-user/key", nil)
	request.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "DIRECT-KEY") {
		t.Fatalf("expected key response, got %d: %s", response.Code, response.Body.String())
	}
	items, err := auditService.List(context.Background(), audit.Filter{Action: audit.ActionViewKey})
	if err != nil || len(items) != 1 {
		t.Fatalf("expected one view_key audit log, got %#v (error: %v)", items, err)
	}
	if items[0].EntityID != "license-user" || items[0].Metadata["assignment_id"] != "assignment-user" {
		t.Fatalf("unexpected audit log: %#v", items[0])
	}
	if strings.Contains(response.Body.String(), "license-other") {
		t.Fatalf("response exposed another license: %s", response.Body.String())
	}
}

func TestEmployeeCannotRevealAnotherUsersKey(t *testing.T) {
	router, token, _ := newSelfServiceHTTPTestRouter(t)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me/licenses/assignment-other/key", nil)
	request.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", response.Code, response.Body.String())
	}
}

func newSelfServiceHTTPTestRouter(t *testing.T) (http.Handler, string, *audit.Service) {
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
	auditService := audit.NewService(audit.NewMemoryRepository())
	handler := NewHTTPHandler(newSelfServiceTestService(t), authHandler, auditService)
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api/v1"))
	return router, pair.AccessToken, auditService
}
