package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"license-manager/backend/internal/modules/auth"
	"license-manager/backend/internal/modules/devices"
	"license-manager/backend/internal/modules/maintenancerequests"
	"license-manager/backend/internal/modules/notifications"
	"license-manager/backend/internal/platform/database"

	"github.com/gin-gonic/gin"
)

func TestLiveHealth(t *testing.T) {
	router := NewRouter(func(context.Context) error { return nil }, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health/live", nil)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, response.Code)
	}
}

func TestReadyHealthWhenDatabaseIsUnavailable(t *testing.T) {
	router := NewRouter(func(context.Context) error { return errors.New("database unavailable") }, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health/ready", nil)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, response.Code)
	}
}

func TestRouterRegistersMaintenanceRoutes(t *testing.T) {
	tokens, err := auth.NewTokenManager("test-secret-with-at-least-thirty-two-characters", "test-issuer", 15*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	users := auth.NewMemoryRepository(nil)
	authHandler := auth.NewHTTPHandler(auth.NewService(users, auth.NewPasswordHasher(4), tokens), tokens)
	service := maintenancerequests.NewService(
		maintenancerequests.NewMemoryRepository(), devices.NewMemoryRepository(), users,
		notifications.NewService(notifications.NewMemoryRepository()), database.DirectTransactor{},
	)
	handler := maintenancerequests.NewHTTPHandler(service, authHandler)
	router := NewRouter(func(context.Context) error { return nil }, authHandler, nil, nil, nil, nil, nil, handler, nil, nil, nil, nil, nil, nil)

	wanted := map[string]bool{
		"GET /api/v1/me/maintenance-requests":             false,
		"POST /api/v1/me/maintenance-requests":            false,
		"POST /api/v1/me/maintenance-requests/:id/cancel": false,
		"GET /api/v1/maintenance-requests":                false,
		"POST /api/v1/maintenance-requests/:id/accept":    false,
		"POST /api/v1/maintenance-requests/:id/complete":  false,
		"POST /api/v1/maintenance-requests/:id/reject":    false,
	}
	for _, route := range router.(*gin.Engine).Routes() {
		key := route.Method + " " + route.Path
		if _, exists := wanted[key]; exists {
			wanted[key] = true
		}
	}
	for route, registered := range wanted {
		if !registered {
			t.Fatalf("maintenance route %s is not registered", route)
		}
	}
}
