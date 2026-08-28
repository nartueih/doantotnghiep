package maintenancerequests

import (
	"errors"
	"strings"
	"testing"
	"time"

	"license-manager/backend/internal/modules/auth"
	"license-manager/backend/internal/modules/devices"
	"license-manager/backend/internal/modules/notifications"
	"license-manager/backend/internal/platform/database"
)

func TestCreateSnapshotsOwnedDeviceAndPreventsOpenDuplicate(t *testing.T) {
	fixture := newServiceFixture(t)

	created, err := fixture.service.Create(t.Context(), fixture.employee.ID, CreateInput{
		DeviceID:    "  " + fixture.device.ID + "  ",
		Category:    " hardware ",
		Priority:    " high ",
		Title:       "  Máy không khởi động  ",
		Description: "  Thiết bị không phản hồi khi nhấn nút nguồn.  ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.RequesterName != fixture.employee.FullName ||
		created.DeviceAssetCode != fixture.device.AssetCode ||
		created.DeviceSerialNumber != fixture.device.SerialNumber ||
		created.DeviceManufacturer != fixture.device.Manufacturer ||
		created.DeviceModel != fixture.device.Model ||
		created.Category != CategoryHardware || created.Priority != PriorityHigh ||
		created.Title != "Máy không khởi động" ||
		created.Description != "Thiết bị không phản hồi khi nhấn nút nguồn." ||
		created.Status != StatusPending {
		t.Fatalf("request was not normalized and enriched: %#v", created)
	}

	_, err = fixture.service.Create(t.Context(), fixture.employee.ID, validCreateInput(fixture.device.ID))
	if !errors.Is(err, ErrOpenDuplicate) {
		t.Fatalf("expected open duplicate, got %v", err)
	}
}

func TestServiceRejectsMalformedIDsAndOversizedTitle(t *testing.T) {
	fixture := newServiceFixture(t)
	input := validCreateInput("not-a-uuid")
	if _, err := fixture.service.Create(t.Context(), fixture.employee.ID, input); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("expected invalid device id, got %v", err)
	}
	input = validCreateInput(fixture.device.ID)
	input.Title = strings.Repeat("a", 201)
	if _, err := fixture.service.Create(t.Context(), fixture.employee.ID, input); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("expected oversized title rejection, got %v", err)
	}
	if _, err := fixture.service.Cancel(t.Context(), fixture.employee.ID, "not-a-uuid"); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("expected invalid request id, got %v", err)
	}
}

func TestListMineAndCancelProtectOwnershipAndReleaseDevice(t *testing.T) {
	fixture := newServiceFixture(t)
	owned, err := fixture.service.Create(t.Context(), fixture.employee.ID, validCreateInput(fixture.device.ID))
	if err != nil {
		t.Fatal(err)
	}

	listed, err := fixture.service.ListMine(t.Context(), fixture.employee.ID)
	if err != nil || listed.Total != 1 || listed.OpenCount != 1 || listed.Items[0].ID != owned.ID {
		t.Fatalf("unexpected own requests: %#v, %v", listed, err)
	}
	if _, err := fixture.service.Cancel(t.Context(), "53000000-0000-0000-0000-000000000099", owned.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected hidden ownership error, got %v", err)
	}
	cancelled, err := fixture.service.Cancel(t.Context(), fixture.employee.ID, owned.ID)
	if err != nil || cancelled.Status != StatusCancelled || cancelled.CancelledAt == nil {
		t.Fatalf("unexpected cancellation: %#v, %v", cancelled, err)
	}
	if _, err := fixture.service.Cancel(t.Context(), fixture.employee.ID, owned.ID); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected terminal state conflict, got %v", err)
	}
	if _, err := fixture.service.Create(t.Context(), fixture.employee.ID, validCreateInput(fixture.device.ID)); err != nil {
		t.Fatalf("expected new request after cancellation: %v", err)
	}
}

func TestAcceptAndCompleteStoreHandlerAndNotifyRequester(t *testing.T) {
	fixture := newServiceFixture(t)
	item, err := fixture.service.Create(t.Context(), fixture.employee.ID, validCreateInput(fixture.device.ID))
	if err != nil {
		t.Fatal(err)
	}

	accepted, err := fixture.service.Accept(t.Context(), fixture.admin.ID, item.ID)
	if err != nil || accepted.Status != StatusInProgress || accepted.AssignedTo != fixture.admin.ID || accepted.AcceptedAt == nil {
		t.Fatalf("unexpected acceptance: %#v, %v", accepted, err)
	}
	notices, err := fixture.notifications.List(t.Context(), fixture.employee.ID)
	if err != nil || notices.UnreadCount != 1 || notices.Items[0].Type != notifications.TypeMaintenanceAccepted || notices.Items[0].EntityID != item.ID {
		t.Fatalf("missing accepted notification: %#v, %v", notices, err)
	}

	if _, err := fixture.service.Complete(t.Context(), fixture.admin.ID, item.ID, CompleteInput{}); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("expected required completion note, got %v", err)
	}
	completed, err := fixture.service.Complete(t.Context(), fixture.admin.ID, item.ID, CompleteInput{ResponseNote: "  Đã thay bàn phím và kiểm tra hoạt động ổn định.  "})
	if err != nil || completed.Status != StatusCompleted || completed.CompletedAt == nil || completed.ResponseNote != "Đã thay bàn phím và kiểm tra hoạt động ổn định." {
		t.Fatalf("unexpected completion: %#v, %v", completed, err)
	}
	notices, _ = fixture.notifications.List(t.Context(), fixture.employee.ID)
	types := map[string]bool{}
	for _, notice := range notices.Items {
		types[notice.Type] = true
	}
	if notices.UnreadCount != 2 || !types[notifications.TypeMaintenanceAccepted] || !types[notifications.TypeMaintenanceCompleted] {
		t.Fatalf("missing completed notification: %#v", notices)
	}
	if _, err := fixture.service.Complete(t.Context(), fixture.admin.ID, item.ID, CompleteInput{ResponseNote: "Lần hai"}); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected terminal state conflict, got %v", err)
	}
}

func TestRejectPendingRequestRequiresNoteAndNotifiesRequester(t *testing.T) {
	fixture := newServiceFixture(t)
	item, err := fixture.service.Create(t.Context(), fixture.employee.ID, validCreateInput(fixture.device.ID))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.service.Reject(t.Context(), fixture.admin.ID, item.ID, RejectInput{}); !errors.Is(err, ErrInvalidData) {
		t.Fatalf("expected required rejection note, got %v", err)
	}
	rejected, err := fixture.service.Reject(t.Context(), fixture.admin.ID, item.ID, RejectInput{ResponseNote: "  Thiết bị không thuộc phạm vi hỗ trợ.  "})
	if err != nil || rejected.Status != StatusRejected || rejected.RejectedAt == nil || rejected.ResponseNote != "Thiết bị không thuộc phạm vi hỗ trợ." {
		t.Fatalf("unexpected rejection: %#v, %v", rejected, err)
	}
	notices, _ := fixture.notifications.List(t.Context(), fixture.employee.ID)
	if notices.UnreadCount != 1 || notices.Items[0].Type != notifications.TypeMaintenanceRejected {
		t.Fatalf("missing rejected notification: %#v", notices)
	}
}

func TestListAdminFiltersAndSearchesDeviceDetails(t *testing.T) {
	fixture := newServiceFixture(t)
	created, err := fixture.service.Create(t.Context(), fixture.employee.ID, validCreateInput(fixture.device.ID))
	if err != nil {
		t.Fatal(err)
	}

	for _, filter := range []Filter{
		{Status: StatusPending},
		{Priority: PriorityNormal},
		{Category: CategoryHardware},
		{Search: "employee one"},
		{Search: "lt-001"},
		{Search: "sn-abc-001"},
		{Search: "thinkpad"},
		{Search: "máy chạy chậm"},
	} {
		items, err := fixture.service.ListAdmin(t.Context(), filter)
		if err != nil || len(items) != 1 || items[0].ID != created.ID {
			t.Fatalf("filter %#v returned %#v, %v", filter, items, err)
		}
	}
	items, err := fixture.service.ListAdmin(t.Context(), Filter{Search: "không tồn tại"})
	if err != nil || len(items) != 0 {
		t.Fatalf("unexpected unmatched result: %#v, %v", items, err)
	}
	if _, err := fixture.service.ListAdmin(t.Context(), Filter{Category: "invalid"}); !errors.Is(err, ErrInvalidCategory) {
		t.Fatalf("expected invalid category, got %v", err)
	}
}

type serviceFixture struct {
	service        *Service
	repository     *MemoryRepository
	notifications  *notifications.Service
	userRepository *auth.MemoryRepository
	employee       auth.User
	admin          auth.User
	device         devices.Device
}

func newServiceFixture(t *testing.T) serviceFixture {
	t.Helper()
	employee := auth.User{ID: "53000000-0000-0000-0000-000000000001", Email: "employee@example.com", FullName: "Employee One", Role: auth.RoleEmployee, Status: auth.StatusActive}
	admin := auth.User{ID: "53000000-0000-0000-0000-000000000002", Email: "admin@example.com", FullName: "Admin User", Role: auth.RoleAdmin, Status: auth.StatusActive}
	users := auth.NewMemoryRepository([]auth.User{employee, admin})
	deviceRepository := devices.NewMemoryRepository()
	device := devices.Device{
		ID: "54000000-0000-0000-0000-000000000001", AssignedUserID: employee.ID, AssignedUserName: employee.FullName,
		AssetCode: "LT-001", SerialNumber: "SN-ABC-001", Name: "ThinkPad T14",
		DeviceType: "laptop", Manufacturer: "Lenovo", Model: "T14 Gen 5",
		Status: devices.StatusAssigned, PurchasedAt: "2026-01-02", WarrantyExpiresAt: "2029-01-02",
		CreatedAt: time.Date(2026, time.January, 2, 8, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, time.January, 2, 8, 0, 0, 0, time.UTC),
	}
	if _, err := deviceRepository.Create(t.Context(), device); err != nil {
		t.Fatal(err)
	}
	notificationService := notifications.NewService(notifications.NewMemoryRepository())
	repository := NewMemoryRepository()
	service := NewService(repository, deviceRepository, users, notificationService, database.DirectTransactor{})
	service.now = func() time.Time { return time.Date(2026, time.August, 28, 9, 0, 0, 0, time.UTC) }
	return serviceFixture{service: service, repository: repository, notifications: notificationService, userRepository: users, employee: employee, admin: admin, device: device}
}

func validCreateInput(deviceID string) CreateInput {
	return CreateInput{DeviceID: deviceID, Category: CategoryHardware, Priority: PriorityNormal, Title: "Máy chạy chậm", Description: "Máy thường xuyên bị treo khi làm việc."}
}
