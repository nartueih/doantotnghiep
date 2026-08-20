package licenses

import (
	"context"
	"errors"
	"time"

	"license-manager/backend/internal/modules/software"
)

const (
	TypeSubscription = "subscription"
	TypePerpetual    = "perpetual"

	AssignmentUser   = "user"
	AssignmentDevice = "device"
	AssignmentMixed  = "mixed"
)

var (
	ErrNotFound              = errors.New("license not found")
	ErrSoftwareNotFound      = errors.New("software product not found")
	ErrInvalidData           = errors.New("license data is invalid")
	ErrInvalidType           = errors.New("license type must be subscription or perpetual")
	ErrInvalidAssignment     = errors.New("assignment type must be user, device or mixed")
	ErrInvalidSeatCount      = errors.New("seat count must be greater than zero")
	ErrInvalidDate           = errors.New("date must use YYYY-MM-DD format")
	ErrExpirationRequired    = errors.New("subscription licenses require an expiration date")
	ErrInvalidDateRange      = errors.New("expiration date must not be before start date")
	ErrInvalidCost           = errors.New("cost cannot be negative and requires a three-letter currency")
	ErrSeatCountBelowUsage   = errors.New("seat count cannot be lower than the number of active assignments")
	ErrNoAvailableSeats      = errors.New("license has no available seats")
	ErrKeyNotSet             = errors.New("license key is not configured")
	ErrEmployeeKeyNotAllowed = errors.New("employee key access is not allowed")
	ErrKeyUnavailable        = errors.New("license key is not available for activation")
	ErrArchived              = errors.New("archived license cannot be modified")
	ErrAlreadyArchived       = errors.New("license is already archived")
	ErrActiveAssignments     = errors.New("license with active assignments cannot be archived")
)

type License struct {
	ID                   string     `json:"id"`
	SoftwareProductID    string     `json:"software_product_id"`
	Name                 string     `json:"name"`
	LicenseType          string     `json:"license_type"`
	AssignmentType       string     `json:"assignment_type"`
	SeatCount            int        `json:"seat_count"`
	UsedSeats            int        `json:"used_seats"`
	AvailableSeats       int        `json:"available_seats"`
	EncryptedKey         []byte     `json:"-"`
	KeyHint              string     `json:"key_hint,omitempty"`
	AllowEmployeeKeyView bool       `json:"allow_employee_key_view"`
	Vendor               string     `json:"vendor"`
	PurchasedAt          string     `json:"purchased_at,omitempty"`
	StartsAt             string     `json:"starts_at,omitempty"`
	ExpiresAt            string     `json:"expires_at,omitempty"`
	Cost                 float64    `json:"cost"`
	Currency             string     `json:"currency,omitempty"`
	Notes                string     `json:"notes,omitempty"`
	LifecycleStatus      string     `json:"lifecycle_status"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
	ArchivedAt           *time.Time `json:"archived_at,omitempty"`
}

type Repository interface {
	List(context.Context) ([]License, error)
	FindByID(context.Context, string) (License, error)
	Create(context.Context, License) (License, error)
	Update(context.Context, License) (License, error)
	Archive(context.Context, string, time.Time) (License, error)
}

type SoftwareFinder interface {
	FindByID(context.Context, string) (software.Product, error)
}
