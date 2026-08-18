package developmentseed

import (
	"bytes"
	"context"
	"testing"
	"time"

	"license-manager/backend/internal/modules/assignments"
	"license-manager/backend/internal/modules/auth"
	"license-manager/backend/internal/modules/dashboard"
	"license-manager/backend/internal/modules/departments"
	"license-manager/backend/internal/modules/devices"
	"license-manager/backend/internal/modules/licenses"
	"license-manager/backend/internal/modules/software"
	"license-manager/backend/internal/modules/users"
	"license-manager/backend/internal/platform/securevalue"
)

func TestSeedCreatesMeaningfulEncryptedDashboardScenario(t *testing.T) {
	ctx := context.Background()
	adminID := "00000000-0000-0000-0000-000000000001"
	authRepository := auth.NewMemoryRepository([]auth.User{{
		ID: adminID, Email: "admin@local.test", FullName: "Development Admin",
		EmployeeCode: "DEV-ADMIN", Role: auth.RoleAdmin, Status: auth.StatusActive,
	}})
	departmentRepository := departments.NewMemoryRepository()
	softwareRepository := software.NewMemoryRepository()
	licenseRepository := licenses.NewMemoryRepository()
	deviceRepository := devices.NewMemoryRepository()
	assignmentRepository := assignments.NewMemoryRepository(licenseRepository)
	cipher, err := securevalue.NewCipher("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")
	if err != nil {
		t.Fatalf("create cipher: %v", err)
	}

	services := Services{
		Departments: departments.NewService(departmentRepository),
		Users:       users.NewService(authRepository, auth.NewPasswordHasher(4), departmentRepository),
		Software:    software.NewService(softwareRepository),
		Licenses:    licenses.NewService(licenseRepository, softwareRepository, cipher),
		Devices:     devices.NewService(deviceRepository, authRepository),
		Assignments: assignments.NewService(assignmentRepository, licenseRepository, authRepository, deviceRepository),
	}

	result, err := Seed(ctx, services, adminID, time.Now().UTC())
	if err != nil {
		t.Fatalf("seed demo data: %v", err)
	}
	if result != (Result{Departments: 3, Users: 5, Software: 6, Licenses: 6, Devices: 6, Assignments: 14}) {
		t.Fatalf("unexpected seed result: %+v", result)
	}

	summary, err := dashboard.NewService(licenseRepository, deviceRepository, softwareRepository).Summary(ctx)
	if err != nil {
		t.Fatalf("load dashboard summary: %v", err)
	}
	if summary.TotalSoftwareProducts != 6 || summary.TotalLicenses != 6 || summary.TotalDevices != 6 {
		t.Fatalf("unexpected totals: %+v", summary)
	}
	if summary.TotalSeats != 32 || summary.UsedSeats != 14 || summary.AvailableSeats != 18 {
		t.Fatalf("unexpected seat totals: %+v", summary)
	}
	if summary.ExpiredLicenses != 1 || summary.ExpiringIn30Days != 2 || summary.ExpiringIn60Days != 3 || summary.ExpiringIn90Days != 4 {
		t.Fatalf("unexpected expiry totals: %+v", summary)
	}
	if summary.ExhaustedLicenses != 1 || summary.HighUsageLicenses != 1 {
		t.Fatalf("unexpected usage alerts: %+v", summary)
	}
	if summary.DevicesByStatus[devices.StatusAvailable] != 2 ||
		summary.DevicesByStatus[devices.StatusAssigned] != 2 ||
		summary.DevicesByStatus[devices.StatusMaintenance] != 1 ||
		summary.DevicesByStatus[devices.StatusRetired] != 1 {
		t.Fatalf("unexpected device statuses: %+v", summary.DevicesByStatus)
	}

	alerts, err := dashboard.NewService(licenseRepository, deviceRepository, softwareRepository).LicenseAlerts(ctx, 90)
	if err != nil {
		t.Fatalf("load license alerts: %v", err)
	}
	if len(alerts) != 5 {
		t.Fatalf("expected 5 alerts, got %d", len(alerts))
	}

	rawLicenses, err := licenseRepository.List(ctx)
	if err != nil {
		t.Fatalf("list raw licenses: %v", err)
	}
	for _, item := range rawLicenses {
		if len(item.EncryptedKey) == 0 {
			t.Fatalf("expected encrypted key for %q", item.Name)
		}
		if bytes.Contains(item.EncryptedKey, []byte("DEMO-")) {
			t.Fatalf("license %q contains a plaintext demo key", item.Name)
		}
	}
}
