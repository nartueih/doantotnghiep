package assignments

import (
	"context"
	"errors"
	"time"

	"license-manager/backend/internal/modules/auth"
	"license-manager/backend/internal/modules/devices"
	"license-manager/backend/internal/modules/licenses"
)

const (
	StatusActive  = "active"
	StatusRevoked = "revoked"
)

var (
	ErrNotFound          = errors.New("license assignment not found or already revoked")
	ErrInvalidTarget     = errors.New("exactly one user_id or device_id is required")
	ErrLicenseNotFound   = errors.New("license not found")
	ErrLicenseInactive   = errors.New("license is not currently active")
	ErrAssignmentType    = errors.New("target does not match the license assignment type")
	ErrTargetUnavailable = errors.New("assignment target does not exist or is unavailable")
	ErrDuplicate         = errors.New("license is already assigned to this target")
	ErrNoAvailableSeats  = errors.New("license has no available seats")
)

type Assignment struct {
	ID             string     `json:"id"`
	LicenseID      string     `json:"license_id"`
	LicenseName    string     `json:"license_name"`
	UserID         string     `json:"user_id,omitempty"`
	DeviceID       string     `json:"device_id,omitempty"`
	TargetName     string     `json:"target_name"`
	AssignedBy     string     `json:"assigned_by"`
	AssignedByName string     `json:"assigned_by_name"`
	AssignedAt     time.Time  `json:"assigned_at"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
	RevokedBy      string     `json:"revoked_by,omitempty"`
	RevokedByName  string     `json:"revoked_by_name,omitempty"`
	Status         string     `json:"status"`
	Notes          string     `json:"notes,omitempty"`
}

type Repository interface {
	List(context.Context) ([]Assignment, error)
	Create(context.Context, Assignment) (Assignment, error)
	Revoke(context.Context, string, string, string, time.Time) (Assignment, error)
}

type LicenseFinder interface {
	FindByID(context.Context, string) (licenses.License, error)
}

type UserFinder interface {
	FindByID(context.Context, string) (auth.User, error)
}

type DeviceFinder interface {
	FindByID(context.Context, string) (devices.Device, error)
}
