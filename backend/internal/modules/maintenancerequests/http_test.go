package maintenancerequests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"license-manager/backend/internal/modules/audit"
	"license-manager/backend/internal/modules/auth"

	"github.com/gin-gonic/gin"
)

func TestMaintenanceRoutesEnforceRolesAndRunEmployeeAdminWorkflow(t *testing.T) {
	fixture := newMaintenanceHTTPFixture(t)

	unauthorized := httptest.NewRecorder()
	fixture.router.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/api/v1/me/maintenance-requests", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", unauthorized.Code, unauthorized.Body.String())
	}
	employeeAdminList := maintenanceRequest(fixture.router, http.MethodGet, "/api/v1/maintenance-requests", fixture.employeeToken, "")
	if employeeAdminList.Code != http.StatusForbidden {
		t.Fatalf("expected employee admin-list 403, got %d: %s", employeeAdminList.Code, employeeAdminList.Body.String())
	}

	createBody := `{"device_id":"` + fixture.device.ID + `","category":"hardware","priority":"urgent","title":"Máy không khởi động","description":"Không phản hồi khi nhấn nút nguồn"}`
	createdResponse := maintenanceRequest(fixture.router, http.MethodPost, "/api/v1/me/maintenance-requests", fixture.employeeToken, createBody)
	if createdResponse.Code != http.StatusCreated || !strings.Contains(createdResponse.Body.String(), `"device_serial_number":"SN-ABC-001"`) {
		t.Fatalf("expected created snapshot, got %d: %s", createdResponse.Code, createdResponse.Body.String())
	}
	var created Request
	if err := json.Unmarshal(createdResponse.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}

	acceptedResponse := maintenanceRequest(fixture.router, http.MethodPost, "/api/v1/maintenance-requests/"+created.ID+"/accept", fixture.adminToken, "")
	if acceptedResponse.Code != http.StatusOK || !strings.Contains(acceptedResponse.Body.String(), `"status":"in_progress"`) {
		t.Fatalf("expected accepted request, got %d: %s", acceptedResponse.Code, acceptedResponse.Body.String())
	}
	completedResponse := maintenanceRequest(fixture.router, http.MethodPost, "/api/v1/maintenance-requests/"+created.ID+"/complete", fixture.adminToken, `{"response_note":"Đã thay bàn phím"}`)
	if completedResponse.Code != http.StatusOK || !strings.Contains(completedResponse.Body.String(), `"status":"completed"`) {
		t.Fatalf("expected completed request, got %d: %s", completedResponse.Code, completedResponse.Body.String())
	}

	auditItems, err := fixture.auditService.List(t.Context(), audit.Filter{EntityType: audit.EntityMaintenanceRequest})
	if err != nil || len(auditItems) != 3 {
		t.Fatalf("unexpected maintenance audit: %#v, %v", auditItems, err)
	}
	encoded, _ := json.Marshal(auditItems)
	if strings.Contains(string(encoded), "Không phản hồi khi nhấn nút nguồn") || strings.Contains(string(encoded), "Đã thay bàn phím") {
		t.Fatalf("audit contains free-form maintenance text: %s", encoded)
	}
}

type maintenanceHTTPFixture struct {
	serviceFixture
	router        http.Handler
	employeeToken string
	adminToken    string
	auditService  *audit.Service
}

func newMaintenanceHTTPFixture(t *testing.T) maintenanceHTTPFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	fixture := newServiceFixture(t)
	tokens, err := auth.NewTokenManager("test-secret-with-at-least-thirty-two-characters", "test-issuer", 15*time.Minute, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	employeePair, err := tokens.IssuePair(fixture.employee)
	if err != nil {
		t.Fatal(err)
	}
	adminPair, err := tokens.IssuePair(fixture.admin)
	if err != nil {
		t.Fatal(err)
	}
	authHandler := auth.NewHTTPHandler(auth.NewService(fixture.userRepository, auth.NewPasswordHasher(4), tokens), tokens)
	auditService := audit.NewService(audit.NewMemoryRepository(), fixture.userRepository)
	handler := NewHTTPHandler(fixture.service, authHandler, auditService)
	router := gin.New()
	handler.RegisterRoutes(router.Group("/api/v1"))
	return maintenanceHTTPFixture{serviceFixture: fixture, router: router, employeeToken: employeePair.AccessToken, adminToken: adminPair.AccessToken, auditService: auditService}
}

func maintenanceRequest(router http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+token)
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	router.ServeHTTP(response, request)
	return response
}
