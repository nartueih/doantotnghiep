package notifications

import (
	"context"
	"errors"
	"time"
)

const (
	TypeLicenseRequestApproved = "license_request_approved"
	TypeLicenseRequestRejected = "license_request_rejected"
	TypeMaintenanceAccepted    = "maintenance_accepted"
	TypeMaintenanceCompleted   = "maintenance_completed"
	TypeMaintenanceRejected    = "maintenance_rejected"
	EntityLicenseRequest       = "license_request"
	EntityMaintenanceRequest   = "maintenance_request"
)

var (
	ErrNotFound    = errors.New("notification not found")
	ErrInvalidData = errors.New("notification data is required")
)

type Notification struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	Type       string     `json:"type"`
	Title      string     `json:"title"`
	Message    string     `json:"message"`
	EntityType string     `json:"entity_type"`
	EntityID   string     `json:"entity_id"`
	CreatedAt  time.Time  `json:"created_at"`
	ReadAt     *time.Time `json:"read_at,omitempty"`
}

type Repository interface {
	ListByUser(context.Context, string) ([]Notification, error)
	Create(context.Context, Notification) (Notification, error)
	MarkRead(context.Context, string, string, time.Time) (Notification, error)
	MarkAllRead(context.Context, string, time.Time) (int, error)
}
