package licenserequests

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"license-manager/backend/internal/modules/assignments"
	"license-manager/backend/internal/modules/auth"
	"license-manager/backend/internal/modules/devices"
	"license-manager/backend/internal/modules/licenses"
	"license-manager/backend/internal/modules/notifications"
	"license-manager/backend/internal/modules/software"
	"license-manager/backend/internal/platform/securevalue"
)

func TestCreateNormalizesRequestAndRejectsDuplicatePending(t *testing.T) {
	fixture := newRequestFixture(t, 2)
	created, err := fixture.service.Create(t.Context(), fixture.employee.ID, CreateInput{
		SoftwareProductID: "  " + fixture.product.ID + " ",
		Priority:          " high ",
		Reason:            "  Cần cho dự án thiết kế  ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.RequesterName != fixture.employee.FullName || created.SoftwareProductName != fixture.product.Name || created.Priority != PriorityHigh || created.Reason != "Cần cho dự án thiết kế" || created.Status != StatusPending {
		t.Fatalf("request was not normalized/enriched: %#v", created)
	}

	_, err = fixture.service.Create(t.Context(), fixture.employee.ID, validCreateInput(fixture))
	if !errors.Is(err, ErrPendingDuplicate) {
		t.Fatalf("expected pending duplicate, got %v", err)
	}
}

func TestCreateValidatesRequesterSoftwarePriorityAndReason(t *testing.T) {
	fixture := newRequestFixture(t, 2)
	tests := []struct {
		name     string
		userID   string
		input    CreateInput
		expected error
	}{
		{name: "missing requester", userID: "missing", input: validCreateInput(fixture), expected: ErrRequesterUnavailable},
		{name: "missing software", userID: fixture.employee.ID, input: CreateInput{SoftwareProductID: "missing", Priority: PriorityNormal, Reason: "Cần dùng"}, expected: ErrSoftwareNotFound},
		{name: "invalid priority", userID: fixture.employee.ID, input: CreateInput{SoftwareProductID: fixture.product.ID, Priority: "low", Reason: "Cần dùng"}, expected: ErrInvalidPriority},
		{name: "blank reason", userID: fixture.employee.ID, input: CreateInput{SoftwareProductID: fixture.product.ID, Priority: PriorityNormal, Reason: "  "}, expected: ErrInvalidData},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := fixture.service.Create(t.Context(), test.userID, test.input)
			if !errors.Is(err, test.expected) {
				t.Fatalf("expected %v, got %v", test.expected, err)
			}
		})
	}
}

func TestListMineAndCancelHideOtherUsersRequests(t *testing.T) {
	fixture := newRequestFixture(t, 2)
	owned, err := fixture.service.Create(t.Context(), fixture.employee.ID, validCreateInput(fixture))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.service.Create(t.Context(), fixture.otherEmployee.ID, validCreateInput(fixture)); err != nil {
		t.Fatal(err)
	}

	items, err := fixture.service.ListMine(t.Context(), fixture.employee.ID)
	if err != nil || len(items) != 1 || items[0].ID != owned.ID {
		t.Fatalf("unexpected own requests: %#v, %v", items, err)
	}
	if _, err := fixture.service.Cancel(t.Context(), fixture.otherEmployee.ID, owned.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected hidden ownership error, got %v", err)
	}
	cancelled, err := fixture.service.Cancel(t.Context(), fixture.employee.ID, owned.ID)
	if err != nil || cancelled.Status != StatusCancelled || cancelled.CancelledAt == nil {
		t.Fatalf("unexpected cancellation: %#v, %v", cancelled, err)
	}
	if _, err := fixture.service.Cancel(t.Context(), fixture.employee.ID, owned.ID); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected terminal state conflict, got %v", err)
	}
	if _, err := fixture.service.Create(t.Context(), fixture.employee.ID, validCreateInput(fixture)); err != nil {
		t.Fatalf("expected resubmission after cancellation: %v", err)
	}
}

func TestRejectRequiresReasonAndResponseThenNotifiesRequester(t *testing.T) {
	fixture := newRequestFixture(t, 2)
	item, _ := fixture.service.Create(t.Context(), fixture.employee.ID, validCreateInput(fixture))
	if _, err := fixture.service.Reject(t.Context(), fixture.admin.ID, item.ID, RejectInput{DecisionReason: "invalid", ResponseNote: "Không cấp"}); !errors.Is(err, ErrInvalidDecision) {
		t.Fatalf("expected invalid decision, got %v", err)
	}
	if _, err := fixture.service.Reject(t.Context(), fixture.admin.ID, item.ID, RejectInput{DecisionReason: DecisionOutOfStock, ResponseNote: "  "}); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("expected required response, got %v", err)
	}

	rejected, err := fixture.service.Reject(t.Context(), fixture.admin.ID, item.ID, RejectInput{
		DecisionReason: DecisionOutOfStock,
		ResponseNote:   "  Hiện tại đã tạm hết license, vui lòng gửi lại sau.  ",
	})
	if err != nil || rejected.Status != StatusRejected || rejected.ReviewedByName != fixture.admin.FullName || rejected.ResponseNote != "Hiện tại đã tạm hết license, vui lòng gửi lại sau." {
		t.Fatalf("unexpected rejection: %#v, %v", rejected, err)
	}
	notices, _ := fixture.notificationService.List(t.Context(), fixture.employee.ID)
	if notices.UnreadCount != 1 || notices.Items[0].Type != notifications.TypeLicenseRequestRejected || notices.Items[0].EntityID != item.ID {
		t.Fatalf("missing rejection notification: %#v", notices)
	}
	if _, err := fixture.service.Reject(t.Context(), fixture.admin.ID, item.ID, RejectInput{DecisionReason: DecisionOther, ResponseNote: "Lần hai"}); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected terminal state conflict, got %v", err)
	}
}

func TestApproveCreatesAssignmentAndNotification(t *testing.T) {
	fixture := newRequestFixture(t, 2)
	item, _ := fixture.service.Create(t.Context(), fixture.employee.ID, validCreateInput(fixture))
	approved, err := fixture.service.Approve(t.Context(), fixture.admin.ID, item.ID, ApproveInput{
		LicenseID:    fixture.license.ID,
		ResponseNote: "  Đã cấp quyền sử dụng.  ",
	})
	if err != nil || approved.Status != StatusApproved || approved.AssignmentID == "" || approved.SelectedLicenseID != fixture.license.ID || approved.ResponseNote != "Đã cấp quyền sử dụng." {
		t.Fatalf("unexpected approval: %#v, %v", approved, err)
	}
	assignmentItems, _ := fixture.assignmentService.List(t.Context())
	if len(assignmentItems) != 1 || assignmentItems[0].UserID != fixture.employee.ID || assignmentItems[0].LicenseID != fixture.license.ID {
		t.Fatalf("unexpected assignments: %#v", assignmentItems)
	}
	notices, _ := fixture.notificationService.List(t.Context(), fixture.employee.ID)
	if notices.UnreadCount != 1 || notices.Items[0].Type != notifications.TypeLicenseRequestApproved || notices.Items[0].EntityID != item.ID {
		t.Fatalf("missing approval notification: %#v", notices)
	}
}

func TestApproveRejectsLicenseForDifferentSoftwareAndLeavesPending(t *testing.T) {
	fixture := newRequestFixture(t, 2)
	item, _ := fixture.service.Create(t.Context(), fixture.employee.ID, validCreateInput(fixture))
	otherProduct, err := software.NewService(fixture.softwareRepository).Create(t.Context(), software.Input{Name: "Office", Publisher: "Microsoft"})
	if err != nil {
		t.Fatal(err)
	}
	otherLicense, err := fixture.licenseService.Create(t.Context(), licenses.Input{
		SoftwareProductID: otherProduct.ID, Name: "Office Business", LicenseType: licenses.TypeSubscription,
		AssignmentType: licenses.AssignmentUser, SeatCount: 2, ExpiresAt: "2099-01-01",
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = fixture.service.Approve(t.Context(), fixture.admin.ID, item.ID, ApproveInput{LicenseID: otherLicense.ID})
	if !errors.Is(err, ErrLicenseProductMismatch) {
		t.Fatalf("expected product mismatch, got %v", err)
	}
	stored, _ := fixture.repository.FindByID(t.Context(), item.ID)
	assignmentsList, _ := fixture.assignmentService.List(t.Context())
	if stored.Status != StatusPending || len(assignmentsList) != 0 {
		t.Fatalf("failed approval changed state: request=%#v assignments=%#v", stored, assignmentsList)
	}
}

func TestApproveWithExhaustedLicenseLeavesRequestPending(t *testing.T) {
	fixture := newRequestFixture(t, 1)
	item, _ := fixture.service.Create(t.Context(), fixture.employee.ID, validCreateInput(fixture))
	if _, err := fixture.assignmentService.Create(t.Context(), fixture.admin.ID, assignments.CreateInput{LicenseID: fixture.license.ID, UserID: fixture.otherEmployee.ID}); err != nil {
		t.Fatal(err)
	}

	_, err := fixture.service.Approve(t.Context(), fixture.admin.ID, item.ID, ApproveInput{LicenseID: fixture.license.ID})
	if !errors.Is(err, assignments.ErrNoAvailableSeats) {
		t.Fatalf("expected no available seats, got %v", err)
	}
	stored, _ := fixture.repository.FindByID(t.Context(), item.ID)
	if stored.Status != StatusPending {
		t.Fatalf("failed approval changed request: %#v", stored)
	}
}

func TestConcurrentApprovalCreatesExactlyOneAssignment(t *testing.T) {
	fixture := newRequestFixture(t, 2)
	item, _ := fixture.service.Create(t.Context(), fixture.employee.ID, validCreateInput(fixture))

	start := make(chan struct{})
	errorsChannel := make(chan error, 2)
	var waitGroup sync.WaitGroup
	for range 2 {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			<-start
			_, err := fixture.service.Approve(context.Background(), fixture.admin.ID, item.ID, ApproveInput{LicenseID: fixture.license.ID})
			errorsChannel <- err
		}()
	}
	close(start)
	waitGroup.Wait()
	close(errorsChannel)

	successes, conflicts := 0, 0
	for err := range errorsChannel {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrInvalidState):
			conflicts++
		default:
			t.Fatalf("unexpected approval error: %v", err)
		}
	}
	assignmentItems, _ := fixture.assignmentService.List(t.Context())
	if successes != 1 || conflicts != 1 || len(assignmentItems) != 1 {
		t.Fatalf("successes=%d conflicts=%d assignments=%d", successes, conflicts, len(assignmentItems))
	}
}

type requestFixture struct {
	service             *Service
	repository          *MemoryRepository
	softwareRepository  *software.MemoryRepository
	licenseService      *licenses.Service
	assignmentService   *assignments.Service
	notificationService *notifications.Service
	userRepository      *auth.MemoryRepository
	product             software.Product
	license             licenses.License
	admin               auth.User
	employee            auth.User
	otherEmployee       auth.User
}

func newRequestFixture(t *testing.T, seats int) requestFixture {
	t.Helper()
	admin := auth.User{ID: "admin-id", Email: "admin@example.com", FullName: "Admin User", Role: auth.RoleAdmin, Status: auth.StatusActive}
	employee := auth.User{ID: "employee-1", Email: "employee@example.com", FullName: "Employee One", Role: auth.RoleEmployee, Status: auth.StatusActive}
	otherEmployee := auth.User{ID: "employee-2", Email: "other@example.com", FullName: "Employee Two", Role: auth.RoleEmployee, Status: auth.StatusActive}
	userRepository := auth.NewMemoryRepository([]auth.User{admin, employee, otherEmployee})

	softwareRepository := software.NewMemoryRepository()
	product, err := software.NewService(softwareRepository).Create(t.Context(), software.Input{Name: "Adobe Photoshop", Publisher: "Adobe"})
	if err != nil {
		t.Fatal(err)
	}
	cipher, err := securevalue.NewCipher("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")
	if err != nil {
		t.Fatal(err)
	}
	licenseRepository := licenses.NewMemoryRepository()
	licenseService := licenses.NewService(licenseRepository, softwareRepository, cipher)
	license, err := licenseService.Create(t.Context(), licenses.Input{
		SoftwareProductID: product.ID, Name: "Adobe Business", LicenseType: licenses.TypeSubscription,
		AssignmentType: licenses.AssignmentUser, SeatCount: seats, ExpiresAt: "2099-01-01",
	})
	if err != nil {
		t.Fatal(err)
	}
	assignmentRepository := assignments.NewMemoryRepository(licenseRepository)
	assignmentService := assignments.NewService(assignmentRepository, licenseRepository, userRepository, devices.NewMemoryRepository())
	notificationService := notifications.NewService(notifications.NewMemoryRepository())
	requestRepository := NewMemoryRepository()
	service := NewService(requestRepository, softwareRepository, licenseRepository, userRepository, assignmentService, notificationService)
	fixedNow := time.Date(2026, time.August, 23, 10, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return fixedNow }

	return requestFixture{
		service: service, repository: requestRepository, softwareRepository: softwareRepository,
		licenseService: licenseService, assignmentService: assignmentService, notificationService: notificationService,
		userRepository: userRepository,
		product:        product, license: license, admin: admin, employee: employee, otherEmployee: otherEmployee,
	}
}

func validCreateInput(fixture requestFixture) CreateInput {
	return CreateInput{SoftwareProductID: fixture.product.ID, Priority: PriorityNormal, Reason: "Cần dùng cho công việc"}
}
