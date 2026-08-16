package licenses

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"license-manager/backend/internal/modules/audit"
	"license-manager/backend/internal/modules/auth"
	"license-manager/backend/internal/modules/software"
	"license-manager/backend/internal/platform/securevalue"
)

func TestCreateResponseDoesNotExposeLicenseKey(t *testing.T) {
	router, token, productID, _ := newLicenseHTTPTestRouter(t, auth.RoleAdmin)
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
	router, token, _, _ := newLicenseHTTPTestRouter(t, auth.RoleEmployee)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/licenses", nil)
	request.Header.Set("Authorization", "Bearer "+token)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d: %s", response.Code, response.Body.String())
	}
}

func TestRevealKeyCreatesAuditLog(t *testing.T) {
	router, token, productID, auditService := newLicenseHTTPTestRouter(t, auth.RoleAdmin)
	createResponse := httptest.NewRecorder()
	createRequest := httptest.NewRequest(http.MethodPost, "/api/v1/licenses", bytes.NewBufferString(`{
		"software_product_id":"`+productID+`",
		"name":"Audited License",
		"license_type":"subscription",
		"assignment_type":"user",
		"seat_count":1,
		"license_key":"AUDIT-SECRET-1234",
		"expires_at":"2099-01-01"
	}`))
	createRequest.Header.Set("Content-Type", "application/json")
	createRequest.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("create license: %d: %s", createResponse.Code, createResponse.Body.String())
	}
	var created License
	if err := json.Unmarshal(createResponse.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created license: %v", err)
	}

	revealResponse := httptest.NewRecorder()
	revealRequest := httptest.NewRequest(http.MethodGet, "/api/v1/licenses/"+created.ID+"/key", nil)
	revealRequest.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(revealResponse, revealRequest)
	if revealResponse.Code != http.StatusOK || !strings.Contains(revealResponse.Body.String(), "AUDIT-SECRET-1234") {
		t.Fatalf("reveal key: %d: %s", revealResponse.Code, revealResponse.Body.String())
	}

	items, err := auditService.List(context.Background(), audit.Filter{Action: audit.ActionViewKey})
	if err != nil || len(items) != 1 {
		t.Fatalf("expected one view_key audit log, got %d (error: %v)", len(items), err)
	}
	encodedMetadata, _ := json.Marshal(items[0].Metadata)
	if strings.Contains(string(encodedMetadata), "AUDIT-SECRET-1234") {
		t.Fatalf("audit metadata exposed license key: %s", encodedMetadata)
	}
}

func newLicenseHTTPTestRouter(t *testing.T, role string) (http.Handler, string, string, *audit.Service) {
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
	auditService := audit.NewService(audit.NewMemoryRepository())
	handler := NewHTTPHandler(NewService(NewMemoryRepository(), softwareRepository, cipher), authHandler, auditService)
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api/v1"))
	return router, pair.AccessToken, product.ID, auditService
}
