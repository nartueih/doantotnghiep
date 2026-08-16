package licenses

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"license-manager/backend/internal/modules/auth"
	"license-manager/backend/internal/modules/software"
	"license-manager/backend/internal/platform/securevalue"
)

func TestCreateResponseDoesNotExposeLicenseKey(t *testing.T) {
	router, token, productID := newLicenseHTTPTestRouter(t, auth.RoleAdmin)
	body := `{
		"software_product_id":"` + productID + `",
		"name":"Photoshop Teams",
		"license_type":"subscription",
		"assignment_type":"user",
		"seat_count":10,
		"license_key":"TOP-SECRET-KEY-1234",
		"expires_at":"2099-01-01"
	}`
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/licenses", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", response.Code, response.Body.String())
	}
	if strings.Contains(response.Body.String(), "TOP-SECRET-KEY-1234") || strings.Contains(response.Body.String(), "encrypted_key") {
		t.Fatalf("response exposed protected key material: %s", response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"key_hint":"****1234"`) {
		t.Fatalf("response did not contain expected key hint: %s", response.Body.String())
	}
}

func TestEmployeeCannotListLicenses(t *testing.T) {
	router, token, _ := newLicenseHTTPTestRouter(t, auth.RoleEmployee)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/licenses", nil)
	request.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", response.Code, response.Body.String())
	}
}

func newLicenseHTTPTestRouter(t *testing.T, role string) (http.Handler, string, string) {
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
	pair, err := tokens.IssuePair(auth.User{ID: "user-id", Email: "user@example.com", Role: role, Status: auth.StatusActive})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	softwareRepository := software.NewMemoryRepository()
	product, err := software.NewService(softwareRepository).Create(context.Background(), software.Input{
		Name: "Adobe Photoshop", Publisher: "Adobe", Version: "2026",
	})
	if err != nil {
		t.Fatalf("create software product: %v", err)
	}
	cipher, err := securevalue.NewCipher("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")
	if err != nil {
		t.Fatalf("create cipher: %v", err)
	}
	authService := auth.NewService(auth.NewMemoryRepository(nil), auth.NewPasswordHasher(4), tokens)
	authHandler := auth.NewHTTPHandler(authService, tokens)
	handler := NewHTTPHandler(NewService(NewMemoryRepository(), softwareRepository, cipher), authHandler)
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api/v1"))
	return router, pair.AccessToken, product.ID
}
