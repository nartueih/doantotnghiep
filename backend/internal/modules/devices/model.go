package devices

import (
	"context"
	"errors"
	"time"

	"license-manager/backend/internal/modules/auth"
)

const (
	StatusAvailable   = "available"
	StatusAssigned    = "assigned"
	StatusMaintenance = "maintenance"
	StatusRetired     = "retired"
	StatusLost        = "lost"
)

var (
	ErrNotFound          = errors.New("device not found")
	ErrAssetCodeExists   = errors.New("asset code already exists")
	ErrSerialExists      = errors.New("serial number already exists")
	ErrInvalidData       = errors.New("asset code, name and device type are required")
	ErrInvalidDate       = errors.New("date must use YYYY-MM-DD format")
	ErrInvalidDateRange  = errors.New("warranty expiration cannot be before purchase date")
	ErrInvalidStatus     = errors.New("device status is invalid")
	ErrDeviceAssigned    = errors.New("assigned device must be unassigned before changing status")
	ErrDeviceUnavailable = errors.New("device is not available for assignment")
	ErrUserUnavailable   = errors.New("user does not exist or is not active")
)

type Device struct {
	ID                string    `json:"id"`
	AssignedUserID    string    `json:"assigned_user_id,omitempty"`
	AssignedUserName  string    `json:"assigned_user_name,omitempty"`
	AssetCode         string    `json:"asset_code"`
	SerialNumber      string    `json:"serial_number,omitempty"`
	Name              string    `json:"name"`
	DeviceType        string    `json:"device_type"`
	Manufacturer      string    `json:"manufacturer,omitempty"`
	Model             string    `json:"model,omitempty"`
	Status            string    `json:"status"`
	PurchasedAt       string    `json:"purchased_at,omitempty"`
	WarrantyExpiresAt string    `json:"warranty_expires_at,omitempty"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type Repository interface {
	List(context.Context) ([]Device, error)
	FindByID(context.Context, string) (Device, error)
	Create(context.Context, Device) (Device, error)
	Update(context.Context, Device) (Device, error)
	UpdateStatus(context.Context, string, string) (Device, error)
	Assign(context.Context, string, string, string) (Device, error)
}

type UserFinder interface {
	FindByID(context.Context, string) (auth.User, error)
}
