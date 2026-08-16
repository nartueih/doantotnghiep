package devices

import (
	"context"
	"errors"
	"testing"
	"time"

	"license-manager/backend/internal/modules/auth"
)

func TestCreateNormalizesDeviceAndRejectsDuplicateAssetCode(t *testing.T) {
	service, _, _ := newDeviceTestService()
	ctx := context.Background()
	created, err := service.Create(ctx, validDeviceInput())
	if err != nil {
		t.Fatalf("create device: %v", err)
	}
	if created.AssetCode != "LAP-001" || created.DeviceType != "laptop" || created.Status != StatusAvailable {
		t.Fatalf("unexpected created device: %#v", created)
	}

	duplicate := validDeviceInput()
	duplicate.AssetCode = "lap-001"
	duplicate.SerialNumber = "OTHER-SERIAL"
	_, err = service.Create(ctx, duplicate)
	if !errors.Is(err, ErrAssetCodeExists) {
		t.Fatalf("expected ErrAssetCodeExists, got %v", err)
	}
}

func TestAssignAndUnassignDevice(t *testing.T) {
	service, _, user := newDeviceTestService()
	ctx := context.Background()
	device, err := service.Create(ctx, validDeviceInput())
	if err != nil {
		t.Fatalf("create device: %v", err)
	}

	assigned, err := service.Assign(ctx, device.ID, user.ID)
	if err != nil {
		t.Fatalf("assign device: %v", err)
	}
	if assigned.Status != StatusAssigned || assigned.AssignedUserID != user.ID || assigned.AssignedUserName != user.FullName {
		t.Fatalf("unexpected assigned device: %#v", assigned)
	}

	_, err = service.ChangeStatus(ctx, device.ID, StatusMaintenance)
	if !errors.Is(err, ErrDeviceAssigned) {
		t.Fatalf("expected ErrDeviceAssigned, got %v", err)
	}

	unassigned, err := service.Assign(ctx, device.ID, "")
	if err != nil {
		t.Fatalf("unassign device: %v", err)
	}
	if unassigned.Status != StatusAvailable || unassigned.AssignedUserID != "" {
		t.Fatalf("unexpected unassigned device: %#v", unassigned)
	}
}

func TestMaintenanceDeviceCannotBeAssigned(t *testing.T) {
	service, _, user := newDeviceTestService()
	ctx := context.Background()
	device, err := service.Create(ctx, validDeviceInput())
	if err != nil {
		t.Fatalf("create device: %v", err)
	}
	if _, err := service.ChangeStatus(ctx, device.ID, StatusMaintenance); err != nil {
		t.Fatalf("set maintenance status: %v", err)
	}

	_, err = service.Assign(ctx, device.ID, user.ID)
	if !errors.Is(err, ErrDeviceUnavailable) {
		t.Fatalf("expected ErrDeviceUnavailable, got %v", err)
	}
}

func TestLockedUserCannotReceiveDevice(t *testing.T) {
	lockedUser := auth.User{
		ID: "locked-user", Email: "locked@example.com", FullName: "Locked User",
		EmployeeCode: "LOCK-001", Role: auth.RoleEmployee, Status: auth.StatusLocked,
	}
	users := auth.NewMemoryRepository([]auth.User{lockedUser})
	service := NewService(NewMemoryRepository(), users)
	device, err := service.Create(context.Background(), validDeviceInput())
	if err != nil {
		t.Fatalf("create device: %v", err)
	}
	_, err = service.Assign(context.Background(), device.ID, lockedUser.ID)
	if !errors.Is(err, ErrUserUnavailable) {
		t.Fatalf("expected ErrUserUnavailable, got %v", err)
	}
}

func TestCreateRejectsInvalidWarrantyRange(t *testing.T) {
	service, _, _ := newDeviceTestService()
	input := validDeviceInput()
	input.PurchasedAt = "2026-08-17"
	input.WarrantyExpiresAt = "2026-08-16"

	_, err := service.Create(context.Background(), input)
	if !errors.Is(err, ErrInvalidDateRange) {
		t.Fatalf("expected ErrInvalidDateRange, got %v", err)
	}
}

func newDeviceTestService() (*Service, *MemoryRepository, auth.User) {
	user := auth.User{
		ID: "active-user", Email: "employee@example.com", FullName: "Active Employee",
		EmployeeCode: "EMP-001", Role: auth.RoleEmployee, Status: auth.StatusActive,
		CreatedAt: time.Now(),
	}
	users := auth.NewMemoryRepository([]auth.User{user})
	repository := NewMemoryRepository()
	return NewService(repository, users), repository, user
}

func validDeviceInput() Input {
	return Input{
		AssetCode:         " lap-001 ",
		SerialNumber:      "SN-ABC-001",
		Name:              "Developer Laptop",
		DeviceType:        " Laptop ",
		Manufacturer:      "Dell",
		Model:             "Latitude 7450",
		PurchasedAt:       "2026-01-10",
		WarrantyExpiresAt: "2029-01-09",
	}
}
