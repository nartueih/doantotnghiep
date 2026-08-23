package notifications

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"license-manager/backend/internal/modules/auth"
)

func TestNotificationRoutesRequireAuthentication(t *testing.T) {
	router, _, _, _ := newNotificationHTTPTestRouter(t, auth.RoleEmployee)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/v1/me/notifications", nil))

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", response.Code, response.Body.String())
	}
}

func TestNotificationRoutesOnlyAllowEmployees(t *testing.T) {
	router, token, _, _ := newNotificationHTTPTestRouter(t, auth.RoleAdmin)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me/notifications", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", response.Code, response.Body.String())
	}
}

func TestEmployeeListsOnlyOwnNotificationsWithUnreadCount(t *testing.T) {
	router, token, service, _ := newNotificationHTTPTestRouter(t, auth.RoleEmployee)
	_, _ = service.Create(t.Context(), CreateInput{
		UserID: "user-1", Type: TypeLicenseRequestApproved, Title: "Đã duyệt",
		Message: "Adobe đã được cấp", EntityType: EntityLicenseRequest, EntityID: "request-1",
	})
	_, _ = service.Create(t.Context(), CreateInput{
		UserID: "user-2", Type: TypeLicenseRequestRejected, Title: "Đã từ chối",
		Message: "Office tạm hết license", EntityType: EntityLicenseRequest, EntityID: "request-2",
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/me/notifications", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"unread_count":1`) || !strings.Contains(response.Body.String(), `"user_id":"user-1"`) || strings.Contains(response.Body.String(), `"user_id":"user-2"`) {
		t.Fatalf("response was not scoped to employee: %s", response.Body.String())
	}
}

func TestEmployeeCannotMarkAnotherUsersNotificationRead(t *testing.T) {
	router, token, service, _ := newNotificationHTTPTestRouter(t, auth.RoleEmployee)
	item, _ := service.Create(t.Context(), CreateInput{
		UserID: "user-2", Type: TypeLicenseRequestRejected, Title: "Đã từ chối",
		Message: "Office tạm hết license", EntityType: EntityLicenseRequest, EntityID: "request-2",
	})

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/me/notifications/"+item.ID+"/read", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", response.Code, response.Body.String())
	}
}

func TestEmployeeMarksOneAndAllNotificationsRead(t *testing.T) {
	router, token, service, _ := newNotificationHTTPTestRouter(t, auth.RoleEmployee)
	first, _ := service.Create(t.Context(), CreateInput{
		UserID: "user-1", Type: TypeLicenseRequestApproved, Title: "Đã duyệt",
		Message: "Adobe đã được cấp", EntityType: EntityLicenseRequest, EntityID: "request-1",
	})
	_, _ = service.Create(t.Context(), CreateInput{
		UserID: "user-1", Type: TypeLicenseRequestRejected, Title: "Đã từ chối",
		Message: "Office tạm hết license", EntityType: EntityLicenseRequest, EntityID: "request-2",
	})

	oneResponse := httptest.NewRecorder()
	oneRequest := httptest.NewRequest(http.MethodPatch, "/api/v1/me/notifications/"+first.ID+"/read", nil)
	oneRequest.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(oneResponse, oneRequest)
	if oneResponse.Code != http.StatusOK || !strings.Contains(oneResponse.Body.String(), `"read_at"`) {
		t.Fatalf("expected read notification, got %d: %s", oneResponse.Code, oneResponse.Body.String())
	}

	allResponse := httptest.NewRecorder()
	allRequest := httptest.NewRequest(http.MethodPatch, "/api/v1/me/notifications/read-all", nil)
	allRequest.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(allResponse, allRequest)
	if allResponse.Code != http.StatusOK || !strings.Contains(allResponse.Body.String(), `"updated":1`) {
		t.Fatalf("expected one remaining update, got %d: %s", allResponse.Code, allResponse.Body.String())
	}
}

func newNotificationHTTPTestRouter(t *testing.T, role string) (http.Handler, string, *Service, *MemoryRepository) {
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
	pair, err := tokens.IssuePair(auth.User{ID: "user-1", Email: "user@example.com", Role: role, Status: auth.StatusActive})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}
	authHandler := auth.NewHTTPHandler(auth.NewService(auth.NewMemoryRepository(nil), auth.NewPasswordHasher(4), tokens), tokens)
	repository := NewMemoryRepository()
	service := NewService(repository)
	handler := NewHTTPHandler(service, authHandler)
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api/v1"))
	return router, pair.AccessToken, service, repository
}
