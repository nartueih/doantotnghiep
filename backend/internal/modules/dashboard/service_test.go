package dashboard

import (
	"context"
	"errors"
	"testing"
	"time"

	"license-manager/backend/internal/modules/devices"
	"license-manager/backend/internal/modules/licenses"
	"license-manager/backend/internal/modules/software"
)

func TestSummaryAggregatesInventorySeatsCostsAndExpiryWindows(t *testing.T) {
	service := newDashboardTestService(t)

	result, err := service.Summary(context.Background())
	if err != nil {
		t.Fatalf("get summary: %v", err)
	}
	if result.TotalDevices != 2 || result.TotalSoftwareProducts != 2 || result.TotalLicenses != 5 {
		t.Fatalf("unexpected inventory totals: %#v", result)
	}
	if result.TotalSeats != 30 || result.UsedSeats != 18 || result.AvailableSeats != 12 {
		t.Fatalf("unexpected seat totals: %#v", result)
	}
	if result.ExpiredLicenses != 1 || result.ExpiringIn30Days != 1 || result.ExpiringIn60Days != 2 || result.ExpiringIn90Days != 3 {
		t.Fatalf("unexpected expiry totals: %#v", result)
	}
	if result.ExhaustedLicenses != 1 || result.HighUsageLicenses != 1 {
		t.Fatalf("unexpected utilization totals: %#v", result)
	}
	if len(result.CostsByCurrency) != 2 || result.CostsByCurrency[0].Currency != "EUR" || result.CostsByCurrency[0].Amount != 200 || result.CostsByCurrency[1].Amount != 150 {
		t.Fatalf("unexpected currency totals: %#v", result.CostsByCurrency)
	}
	if result.DevicesByStatus[devices.StatusAvailable] != 1 || result.DevicesByStatus[devices.StatusAssigned] != 1 {
		t.Fatalf("unexpected device status totals: %#v", result.DevicesByStatus)
	}
}

func TestLicenseAlertsRespectWindowAndCombineReasons(t *testing.T) {
	service := newDashboardTestService(t)

	alerts30, err := service.LicenseAlerts(context.Background(), 30)
	if err != nil {
		t.Fatalf("get 30-day alerts: %v", err)
	}
	if len(alerts30) != 2 {
		t.Fatalf("expected two alerts, got %d: %#v", len(alerts30), alerts30)
	}
	if alerts30[0].Severity != "critical" || len(alerts30[0].AlertTypes) != 2 {
		t.Fatalf("expected combined critical alert, got %#v", alerts30[0])
	}
	if alerts30[1].Severity != "warning" || alerts30[1].UtilizationPercent != 80 {
		t.Fatalf("expected high-usage warning, got %#v", alerts30[1])
	}

	alerts60, err := service.LicenseAlerts(context.Background(), 60)
	if err != nil || len(alerts60) != 3 {
		t.Fatalf("expected three 60-day alerts, got %d (error: %v)", len(alerts60), err)
	}
}

func TestLicenseAlertsRejectUnsupportedWindow(t *testing.T) {
	service := newDashboardTestService(t)

	_, err := service.LicenseAlerts(context.Background(), 45)
	if !errors.Is(err, ErrInvalidExpiryWindow) {
		t.Fatalf("expected ErrInvalidExpiryWindow, got %v", err)
	}
}

func newDashboardTestService(t *testing.T) *Service {
	t.Helper()
	licenseRepository := licenses.NewMemoryRepository()
	deviceRepository := devices.NewMemoryRepository()
	softwareRepository := software.NewMemoryRepository()
	ctx := context.Background()

	for _, product := range []software.Product{
		{ID: "software-1", Name: "Product One"},
		{ID: "software-2", Name: "Product Two"},
	} {
		if _, err := softwareRepository.Create(ctx, product); err != nil {
			t.Fatalf("create software: %v", err)
		}
	}
	for _, device := range []devices.Device{
		{ID: "device-1", AssetCode: "DEV-1", Status: devices.StatusAvailable},
		{ID: "device-2", AssetCode: "DEV-2", Status: devices.StatusAssigned},
	} {
		if _, err := deviceRepository.Create(ctx, device); err != nil {
			t.Fatalf("create device: %v", err)
		}
	}
	for _, item := range []licenses.License{
		{ID: "expired", Name: "Expired and exhausted", LicenseType: licenses.TypeSubscription, SeatCount: 10, UsedSeats: 10, ExpiresAt: "2026-08-16", Cost: 100, Currency: "USD"},
		{ID: "soon", Name: "Soon and high usage", LicenseType: licenses.TypeSubscription, SeatCount: 10, UsedSeats: 8, ExpiresAt: "2026-09-01", Cost: 50, Currency: "USD"},
		{ID: "sixty", Name: "Within sixty days", LicenseType: licenses.TypeSubscription, SeatCount: 5, ExpiresAt: "2026-10-01", Cost: 200, Currency: "EUR"},
		{ID: "ninety", Name: "Within ninety days", LicenseType: licenses.TypeSubscription, SeatCount: 2, ExpiresAt: "2026-11-01"},
		{ID: "perpetual", Name: "Perpetual", LicenseType: licenses.TypePerpetual, SeatCount: 3},
	} {
		if _, err := licenseRepository.Create(ctx, item); err != nil {
			t.Fatalf("create license: %v", err)
		}
	}

	service := NewService(licenseRepository, deviceRepository, softwareRepository)
	serverLocation := time.FixedZone("UTC+7", 7*60*60)
	service.now = func() time.Time { return time.Date(2026, 8, 17, 2, 0, 0, 0, serverLocation) }
	return service
}
