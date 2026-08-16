package selfservice

import (
	"context"
	"time"

	"license-manager/backend/internal/modules/assignments"
	"license-manager/backend/internal/modules/devices"
	"license-manager/backend/internal/modules/licenses"
)

const (
	SourceUser   = "user"
	SourceDevice = "device"
)

type AssignedLicense struct {
	AssignmentID      string    `json:"assignment_id"`
	LicenseID         string    `json:"license_id"`
	LicenseName       string    `json:"license_name"`
	SoftwareProductID string    `json:"software_product_id"`
	LicenseType       string    `json:"license_type"`
	AssignmentSource  string    `json:"assignment_source"`
	DeviceID          string    `json:"device_id,omitempty"`
	DeviceAssetCode   string    `json:"device_asset_code,omitempty"`
	AssignedAt        time.Time `json:"assigned_at"`
	ExpiresAt         string    `json:"expires_at,omitempty"`
	LifecycleStatus   string    `json:"lifecycle_status"`
	Notes             string    `json:"notes,omitempty"`
}

type DeviceLister interface {
	List(context.Context) ([]devices.Device, error)
}

type AssignmentLister interface {
	List(context.Context) ([]assignments.Assignment, error)
}

type LicenseLister interface {
	List(context.Context) ([]licenses.License, error)
}
