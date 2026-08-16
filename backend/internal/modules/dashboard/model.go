package dashboard

import (
	"context"
	"errors"
	"time"

	"license-manager/backend/internal/modules/devices"
	"license-manager/backend/internal/modules/licenses"
	"license-manager/backend/internal/modules/software"
)

const HighUsageThreshold = 80.0

var ErrInvalidExpiryWindow = errors.New("expiry window must be 30, 60 or 90 days")

type CostByCurrency struct {
	Currency string  `json:"currency"`
	Amount   float64 `json:"amount"`
}

type Summary struct {
	TotalDevices          int              `json:"total_devices"`
	DevicesByStatus       map[string]int   `json:"devices_by_status"`
	TotalSoftwareProducts int              `json:"total_software_products"`
	TotalLicenses         int              `json:"total_licenses"`
	TotalSeats            int              `json:"total_seats"`
	UsedSeats             int              `json:"used_seats"`
	AvailableSeats        int              `json:"available_seats"`
	ExpiredLicenses       int              `json:"expired_licenses"`
	ExpiringIn30Days      int              `json:"expiring_in_30_days"`
	ExpiringIn60Days      int              `json:"expiring_in_60_days"`
	ExpiringIn90Days      int              `json:"expiring_in_90_days"`
	ExhaustedLicenses     int              `json:"exhausted_licenses"`
	HighUsageLicenses     int              `json:"high_usage_licenses"`
	CostsByCurrency       []CostByCurrency `json:"costs_by_currency"`
	GeneratedAt           time.Time        `json:"generated_at"`
}

type LicenseAlert struct {
	LicenseID          string   `json:"license_id"`
	LicenseName        string   `json:"license_name"`
	LicenseType        string   `json:"license_type"`
	ExpiresAt          string   `json:"expires_at,omitempty"`
	DaysUntilExpiry    *int     `json:"days_until_expiry,omitempty"`
	SeatCount          int      `json:"seat_count"`
	UsedSeats          int      `json:"used_seats"`
	AvailableSeats     int      `json:"available_seats"`
	UtilizationPercent float64  `json:"utilization_percent"`
	Severity           string   `json:"severity"`
	AlertTypes         []string `json:"alert_types"`
}

type LicenseLister interface {
	List(context.Context) ([]licenses.License, error)
}

type DeviceLister interface {
	List(context.Context) ([]devices.Device, error)
}

type SoftwareLister interface {
	List(context.Context) ([]software.Product, error)
}
