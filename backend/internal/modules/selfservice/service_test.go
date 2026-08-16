package selfservice

import (
	"context"
	"testing"
	"time"

	"license-manager/backend/internal/modules/assignments"
	"license-manager/backend/internal/modules/devices"
	"license-manager/backend/internal/modules/licenses"
)

func TestDevicesReturnsOnlyCurrentUsersDevices(t *testing.T) {
	service := newSelfServiceTestService(t)

	items, err := service.Devices(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("list devices: %v", err)
	}
	if len(items) != 1 || items[0].ID != "device-1" {
		t.Fatalf("unexpected devices: %#v", items)
	}
}

func TestLicensesIncludesDirectAndOwnedDeviceAssignmentsOnly(t *testing.T) {
	service := newSelfServiceTestService(t)

	items, err := service.Licenses(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("list licenses: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected two assigned licenses, got %d: %#v", len(items), items)
	}
	sources := map[string]bool{}
	for _, item := range items {
		sources[item.AssignmentSource] = true
		if item.LicenseID == "license-other" || item.LicenseID == "license-revoked" {
			t.Fatalf("response exposed another or revoked assignment: %#v", item)
		}
	}
	if !sources[SourceUser] || !sources[SourceDevice] {
		t.Fatalf("expected both assignment sources, got %#v", sources)
	}
}

func TestEmptyUserIDReturnsNoUnassignedDevices(t *testing.T) {
	service := newSelfServiceTestService(t)

	items, err := service.Devices(context.Background(), "")
	if err != nil || len(items) != 0 {
		t.Fatalf("expected no devices, got %#v (error: %v)", items, err)
	}
}

func newSelfServiceTestService(t *testing.T) *Service {
	t.Helper()
	ctx := context.Background()
	licenseRepository := licenses.NewMemoryRepository()
	deviceRepository := devices.NewMemoryRepository()
	assignmentRepository := assignments.NewMemoryRepository(licenseRepository)

	for _, item := range []licenses.License{
		{ID: "license-user", Name: "Direct License", LicenseType: licenses.TypeSubscription, SeatCount: 3, ExpiresAt: "2099-01-01"},
		{ID: "license-device", Name: "Device License", LicenseType: licenses.TypePerpetual, SeatCount: 3},
		{ID: "license-other", Name: "Other User License", LicenseType: licenses.TypePerpetual, SeatCount: 3},
		{ID: "license-revoked", Name: "Revoked License", LicenseType: licenses.TypePerpetual, SeatCount: 3},
	} {
		if _, err := licenseRepository.Create(ctx, item); err != nil {
			t.Fatalf("create license: %v", err)
		}
	}
	for _, item := range []devices.Device{
		{ID: "device-1", AssetCode: "USER-1-ASSET", AssignedUserID: "user-1", Status: devices.StatusAssigned},
		{ID: "device-2", AssetCode: "OTHER-ASSET", AssignedUserID: "user-2", Status: devices.StatusAssigned},
		{ID: "device-unassigned", AssetCode: "FREE-ASSET", Status: devices.StatusAvailable},
	} {
		if _, err := deviceRepository.Create(ctx, item); err != nil {
			t.Fatalf("create device: %v", err)
		}
	}

	assignedAt := time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC)
	assignmentItems := []assignments.Assignment{
		{ID: "assignment-user", LicenseID: "license-user", LicenseName: "Direct License", UserID: "user-1", Status: assignments.StatusActive, AssignedAt: assignedAt},
		{ID: "assignment-device", LicenseID: "license-device", LicenseName: "Device License", DeviceID: "device-1", Status: assignments.StatusActive, AssignedAt: assignedAt.Add(time.Minute)},
		{ID: "assignment-other", LicenseID: "license-other", LicenseName: "Other User License", UserID: "user-2", Status: assignments.StatusActive, AssignedAt: assignedAt},
		{ID: "assignment-revoked", LicenseID: "license-revoked", LicenseName: "Revoked License", UserID: "user-1", Status: assignments.StatusActive, AssignedAt: assignedAt},
	}
	for _, item := range assignmentItems {
		if _, err := assignmentRepository.Create(ctx, item); err != nil {
			t.Fatalf("create assignment: %v", err)
		}
	}
	if _, err := assignmentRepository.Revoke(ctx, "assignment-revoked", "admin", "Admin", assignedAt.Add(time.Hour)); err != nil {
		t.Fatalf("revoke assignment: %v", err)
	}

	service := NewService(deviceRepository, assignmentRepository, licenseRepository)
	service.now = func() time.Time { return assignedAt }
	return service
}
