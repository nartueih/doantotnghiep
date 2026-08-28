package maintenancerequests

import (
	"errors"
	"testing"
	"time"

	"license-manager/backend/internal/testsupport"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	maintenanceEmployeeID = "51000000-0000-0000-0000-000000000001"
	maintenanceAdminID    = "51000000-0000-0000-0000-000000000002"
	maintenanceDeviceID   = "52000000-0000-0000-0000-000000000001"
)

func TestPostgresRepositoryPersistsSearchesAndTransitions(t *testing.T) {
	pool := testsupport.OpenPostgres(t)
	seedMaintenanceReferences(t, pool)
	repository := NewPostgresRepository(pool)
	createdAt := time.Date(2026, time.August, 28, 9, 0, 0, 0, time.UTC)
	request := postgresPendingRequest("50000000-0000-0000-0000-000000000001", createdAt)

	created, err := repository.Create(t.Context(), request)
	if err != nil || created.DeviceSerialNumber != "SN-MAINT-001" {
		t.Fatalf("created=%#v err=%v", created, err)
	}
	duplicate := postgresPendingRequest("50000000-0000-0000-0000-000000000002", createdAt.Add(time.Second))
	if _, err := repository.Create(t.Context(), duplicate); !errors.Is(err, ErrOpenDuplicate) {
		t.Fatalf("expected open duplicate, got %v", err)
	}
	items, err := repository.List(t.Context(), Filter{Status: StatusPending, Category: CategoryHardware, Search: "sn-maint"})
	if err != nil || len(items) != 1 || items[0].ID != request.ID {
		t.Fatalf("items=%#v err=%v", items, err)
	}

	acceptedAt := createdAt.Add(time.Hour)
	accepted, err := repository.Accept(t.Context(), AcceptUpdate{RequestID: request.ID, ActorID: maintenanceAdminID, ActorName: "Admin User", AcceptedAt: acceptedAt})
	if err != nil || accepted.Status != StatusInProgress || accepted.AcceptedAt == nil {
		t.Fatalf("accepted=%#v err=%v", accepted, err)
	}
	completedAt := acceptedAt.Add(time.Hour)
	completed, err := repository.Complete(t.Context(), CompleteUpdate{
		RequestID: request.ID, ActorID: maintenanceAdminID, ActorName: "Admin User",
		ResponseNote: "Đã thay bàn phím", CompletedAt: completedAt,
	})
	if err != nil || completed.Status != StatusCompleted || completed.ResponseNote != "Đã thay bàn phím" || completed.CompletedAt == nil {
		t.Fatalf("completed=%#v err=%v", completed, err)
	}
	if _, err := repository.Accept(t.Context(), AcceptUpdate{RequestID: request.ID, ActorID: maintenanceAdminID, ActorName: "Admin User", AcceptedAt: completedAt}); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected terminal conflict, got %v", err)
	}
}

func TestPostgresCreateRechecksCurrentDeviceOwnership(t *testing.T) {
	pool := testsupport.OpenPostgres(t)
	seedMaintenanceReferences(t, pool)
	repository := NewPostgresRepository(pool)
	if _, err := pool.Exec(t.Context(), `UPDATE devices SET assigned_user_id = $2 WHERE id = $1`, maintenanceDeviceID, maintenanceAdminID); err != nil {
		t.Fatal(err)
	}
	_, err := repository.Create(t.Context(), postgresPendingRequest("50000000-0000-0000-0000-000000000003", time.Now().UTC()))
	if !errors.Is(err, ErrDeviceNotFound) {
		t.Fatalf("expected device ownership conflict, got %v", err)
	}
}

func postgresPendingRequest(id string, createdAt time.Time) Request {
	return Request{
		ID: id, RequesterID: maintenanceEmployeeID, RequesterName: "Employee One",
		DeviceID: maintenanceDeviceID, DeviceAssetCode: "LT-MAINT-001", DeviceSerialNumber: "SN-MAINT-001",
		DeviceName: "ThinkPad T14", DeviceType: "laptop", DeviceManufacturer: "Lenovo", DeviceModel: "T14 Gen 5",
		DevicePurchasedAt: "2026-01-02", DeviceWarrantyExpiresAt: "2029-01-02",
		Category: CategoryHardware, Priority: PriorityHigh, Title: "Máy không khởi động",
		Description: "Thiết bị không phản hồi khi nhấn nút nguồn.", Status: StatusPending,
		LastActorID: maintenanceEmployeeID, LastActorName: "Employee One", CreatedAt: createdAt, UpdatedAt: createdAt,
	}
}

func seedMaintenanceReferences(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(t.Context(), `
		INSERT INTO users (id, email, password_hash, full_name, employee_code, role, status)
		VALUES
			($1, 'maintenance-employee@local.test', 'hash', 'Employee One', 'MAINT-EMP', 'employee', 'active'),
			($2, 'maintenance-admin@local.test', 'hash', 'Admin User', 'MAINT-ADMIN', 'admin', 'active')
	`, maintenanceEmployeeID, maintenanceAdminID); err != nil {
		t.Fatalf("seed maintenance users: %v", err)
	}
	if _, err := pool.Exec(t.Context(), `
		INSERT INTO devices (
			id, assigned_user_id, asset_code, serial_number, name, device_type,
			manufacturer, model, status, purchased_at, warranty_expires_at
		) VALUES ($1, $2, 'LT-MAINT-001', 'SN-MAINT-001', 'ThinkPad T14', 'laptop',
			'Lenovo', 'T14 Gen 5', 'assigned', DATE '2026-01-02', DATE '2029-01-02')
	`, maintenanceDeviceID, maintenanceEmployeeID); err != nil {
		t.Fatalf("seed maintenance device: %v", err)
	}
}
