package licenserequests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"license-manager/backend/internal/modules/assignments"
	"license-manager/backend/internal/modules/audit"
	"license-manager/backend/internal/modules/auth"
	"license-manager/backend/internal/modules/licenses"
	"license-manager/backend/internal/modules/software"
)

func TestLicenseRequestRoutesRequireAuthentication(t *testing.T) {
	fixture := newRequestHTTPFixture(t)
	for _, path := range []string{"/api/v1/me/license-requests", "/api/v1/license-requests"} {
		response := httptest.NewRecorder()
		fixture.router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 for %s, got %d: %s", path, response.Code, response.Body.String())
		}
	}
}

func TestEmployeeCreatesAndListsOnlyOwnLicenseRequests(t *testing.T) {
	fixture := newRequestHTTPFixture(t)
	if _, err := fixture.service.Create(t.Context(), fixture.otherEmployee.ID, validCreateInput(fixture.requestFixture)); err != nil {
		t.Fatal(err)
	}
	body := `{"software_product_id":"` + fixture.product.ID + `","priority":"high","reason":"Cần Photoshop cho dự án"}`
	createResponse := performJSONRequest(fixture.router, http.MethodPost, "/api/v1/me/license-requests", fixture.employeeToken, body)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", createResponse.Code, createResponse.Body.String())
	}

	listResponse := performJSONRequest(fixture.router, http.MethodGet, "/api/v1/me/license-requests", fixture.employeeToken, "")
	if listResponse.Code != http.StatusOK || !strings.Contains(listResponse.Body.String(), `"total":1`) || strings.Contains(listResponse.Body.String(), fixture.otherEmployee.ID) {
		t.Fatalf("unexpected own list %d: %s", listResponse.Code, listResponse.Body.String())
	}
	auditItems, err := fixture.auditService.List(t.Context(), audit.Filter{Action: audit.ActionRequest})
	if err != nil || len(auditItems) != 1 || auditItems[0].EntityType != audit.EntityLicenseRequest {
		t.Fatalf("missing request audit: %#v, %v", auditItems, err)
	}
}

func TestRequestableSoftwareIsEmployeeOnly(t *testing.T) {
	fixture := newRequestHTTPFixture(t)
	employeeResponse := performJSONRequest(fixture.router, http.MethodGet, "/api/v1/me/requestable-software", fixture.employeeToken, "")
	if employeeResponse.Code != http.StatusOK || !strings.Contains(employeeResponse.Body.String(), fixture.product.Name) {
		t.Fatalf("expected software catalog, got %d: %s", employeeResponse.Code, employeeResponse.Body.String())
	}
	adminResponse := performJSONRequest(fixture.router, http.MethodGet, "/api/v1/me/requestable-software", fixture.adminToken, "")
	if adminResponse.Code != http.StatusForbidden {
		t.Fatalf("expected admin self-service forbidden, got %d: %s", adminResponse.Code, adminResponse.Body.String())
	}
}

func TestEmployeeCannotUseAdminRequestList(t *testing.T) {
	fixture := newRequestHTTPFixture(t)
	response := performJSONRequest(fixture.router, http.MethodGet, "/api/v1/license-requests", fixture.employeeToken, "")
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", response.Code, response.Body.String())
	}
}

func TestAdminApprovesRequestAndCreatesSafeAudit(t *testing.T) {
	fixture := newRequestHTTPFixture(t)
	item, _ := fixture.service.Create(t.Context(), fixture.employee.ID, validCreateInput(fixture.requestFixture))
	body := `{"license_id":"` + fixture.license.ID + `","response_note":"Đã cấp license"}`
	response := performJSONRequest(fixture.router, http.MethodPatch, "/api/v1/license-requests/"+item.ID+"/approve", fixture.adminToken, body)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"status":"approved"`) || !strings.Contains(response.Body.String(), `"assignment_id"`) {
		t.Fatalf("expected approved request, got %d: %s", response.Code, response.Body.String())
	}
	auditItems, err := fixture.auditService.List(t.Context(), audit.Filter{Action: audit.ActionApprove})
	if err != nil || len(auditItems) != 1 || auditItems[0].EntityID != item.ID {
		t.Fatalf("missing approval audit: %#v, %v", auditItems, err)
	}
	encoded, _ := json.Marshal(auditItems[0].Metadata)
	if strings.Contains(strings.ToLower(string(encoded)), "license_key") || strings.Contains(string(encoded), "Đã cấp license") {
		t.Fatalf("audit metadata contains sensitive/free-form data: %s", encoded)
	}
}

func TestAdminRejectsOutOfStockAndCreatesAudit(t *testing.T) {
	fixture := newRequestHTTPFixture(t)
	item, _ := fixture.service.Create(t.Context(), fixture.employee.ID, validCreateInput(fixture.requestFixture))
	body := `{"decision_reason":"out_of_stock","response_note":"Tạm hết license, vui lòng gửi lại sau"}`
	response := performJSONRequest(fixture.router, http.MethodPatch, "/api/v1/license-requests/"+item.ID+"/reject", fixture.adminToken, body)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"status":"rejected"`) || !strings.Contains(response.Body.String(), `"decision_reason":"out_of_stock"`) {
		t.Fatalf("expected rejected request, got %d: %s", response.Code, response.Body.String())
	}
	auditItems, err := fixture.auditService.List(t.Context(), audit.Filter{Action: audit.ActionReject})
	if err != nil || len(auditItems) != 1 || auditItems[0].Metadata["decision_reason"] != DecisionOutOfStock {
		t.Fatalf("missing rejection audit: %#v, %v", auditItems, err)
	}
}

func TestEmployeeCannotCancelAnotherUsersRequest(t *testing.T) {
	fixture := newRequestHTTPFixture(t)
	item, _ := fixture.service.Create(t.Context(), fixture.otherEmployee.ID, validCreateInput(fixture.requestFixture))
	response := performJSONRequest(fixture.router, http.MethodPatch, "/api/v1/me/license-requests/"+item.ID+"/cancel", fixture.employeeToken, "")
	if response.Code != http.StatusNotFound {
		t.Fatalf("expected hidden ownership 404, got %d: %s", response.Code, response.Body.String())
	}
}

func TestApprovalReturnsMachineReadableProductMismatch(t *testing.T) {
	fixture := newRequestHTTPFixture(t)
	item, _ := fixture.service.Create(t.Context(), fixture.employee.ID, validCreateInput(fixture.requestFixture))
	otherProduct, err := software.NewService(fixture.softwareRepository).Create(t.Context(), software.Input{Name: "Office", Publisher: "Microsoft"})
	if err != nil {
		t.Fatal(err)
	}
	otherLicense, err := fixture.licenseService.Create(t.Context(), licenses.Input{
		SoftwareProductID: otherProduct.ID, Name: "Office Business", LicenseType: licenses.TypeSubscription,
		AssignmentType: licenses.AssignmentUser, SeatCount: 1, ExpiresAt: "2099-01-01",
	})
	if err != nil {
		t.Fatal(err)
	}

	response := performJSONRequest(fixture.router, http.MethodPatch, "/api/v1/license-requests/"+item.ID+"/approve", fixture.adminToken, `{"license_id":"`+otherLicense.ID+`"}`)
	if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), `"code":"license_product_mismatch"`) {
		t.Fatalf("expected product mismatch 422 with code, got %d: %s", response.Code, response.Body.String())
	}
}

func TestApprovalReturnsMachineReadableNoAvailableSeats(t *testing.T) {
	fixture := newRequestHTTPFixtureWithSeats(t, 1)
	item, _ := fixture.service.Create(t.Context(), fixture.employee.ID, validCreateInput(fixture.requestFixture))
	if _, err := fixture.assignmentService.Create(t.Context(), fixture.admin.ID, assignments.CreateInput{
		LicenseID: fixture.license.ID, UserID: fixture.otherEmployee.ID,
	}); err != nil {
		t.Fatal(err)
	}

	response := performJSONRequest(fixture.router, http.MethodPatch, "/api/v1/license-requests/"+item.ID+"/approve", fixture.adminToken, `{"license_id":"`+fixture.license.ID+`"}`)
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), `"code":"no_available_seats"`) {
		t.Fatalf("expected no-seat 409 with code, got %d: %s", response.Code, response.Body.String())
	}
}

type requestHTTPFixture struct {
	requestFixture
	router        http.Handler
	employeeToken string
	adminToken    string
	auditService  *audit.Service
}

func newRequestHTTPFixture(t *testing.T) requestHTTPFixture {
	return newRequestHTTPFixtureWithSeats(t, 2)
}

func newRequestHTTPFixtureWithSeats(t *testing.T, seats int) requestHTTPFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)
	fixture := newRequestFixture(t, seats)
	tokens, err := auth.NewTokenManager(
		"test-secret-with-at-least-thirty-two-characters",
		"test-issuer",
		15*time.Minute,
		24*time.Hour,
	)
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
	return requestHTTPFixture{
		requestFixture: fixture, router: router, employeeToken: employeePair.AccessToken,
		adminToken: adminPair.AccessToken, auditService: auditService,
	}
}

func performJSONRequest(router http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+token)
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	router.ServeHTTP(response, request)
	return response
}
