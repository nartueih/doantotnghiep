package assignments

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"license-manager/backend/internal/modules/auth"
)

func TestAdminCanCreateAssignment(t *testing.T) {
	fixture := newAssignmentFixture(t, 1, "user")
	router, token := newAssignmentHTTPRouter(t, fixture, fixture.admin)
	body := `{"license_id":"` + fixture.license.ID + `","user_id":"` + fixture.employee1.ID + `"}`
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/license-assignments", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", response.Code, response.Body.String())
	}
}

func TestEmployeeCannotListAssignments(t *testing.T) {
	fixture := newAssignmentFixture(t, 1, "user")
	router, token := newAssignmentHTTPRouter(t, fixture, fixture.employee1)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/license-assignments", nil)
	request.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", response.Code, response.Body.String())
	}
}

func newAssignmentHTTPRouter(t *testing.T, fixture assignmentFixture, caller auth.User) (http.Handler, string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	tokens, err := auth.NewTokenManager(
		"test-secret-with-at-least-thirty-two-characters", "test-issuer", 15*time.Minute, 24*time.Hour,
	)
	if err != nil {
		t.Fatalf("create token manager: %v", err)
	}
	pair, err := tokens.IssuePair(caller)
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	users := auth.NewMemoryRepository([]auth.User{fixture.admin, fixture.employee1, fixture.employee2})
	authHandler := auth.NewHTTPHandler(auth.NewService(users, auth.NewPasswordHasher(4), tokens), tokens)
	handler := NewHTTPHandler(fixture.service, authHandler)
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api/v1"))
	return router, pair.AccessToken
}
