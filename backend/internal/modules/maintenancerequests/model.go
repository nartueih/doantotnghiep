package maintenancerequests

import (
	"context"
	"errors"
	"time"

	"license-manager/backend/internal/modules/auth"
	"license-manager/backend/internal/modules/devices"
	"license-manager/backend/internal/modules/notifications"
)

const (
	CategoryHardware  = "hardware"
	CategorySoftware  = "software"
	CategoryNetwork   = "network"
	CategoryAccessory = "accessory"
	CategoryOther     = "other"

	PriorityNormal = "normal"
	PriorityHigh   = "high"
	PriorityUrgent = "urgent"

	StatusPending    = "pending"
	StatusInProgress = "in_progress"
	StatusCompleted  = "completed"
	StatusRejected   = "rejected"
	StatusCancelled  = "cancelled"
)

var (
	ErrNotFound             = errors.New("maintenance request not found")
	ErrInvalidData          = errors.New("device, category, priority, title and description are required")
	ErrInvalidCategory      = errors.New("category must be hardware, software, network, accessory or other")
	ErrInvalidPriority      = errors.New("priority must be normal, high or urgent")
	ErrOpenDuplicate        = errors.New("an open maintenance request already exists for this device")
	ErrInvalidState         = errors.New("maintenance request state does not allow this action")
	ErrRequesterUnavailable = errors.New("requester does not exist or is unavailable")
	ErrReviewerUnavailable  = errors.New("reviewer does not exist or is unavailable")
	ErrDeviceNotFound       = errors.New("assigned device not found")
)

type Request struct {
	ID                      string     `json:"id"`
	RequesterID             string     `json:"requester_id"`
	RequesterName           string     `json:"requester_name"`
	DeviceID                string     `json:"device_id"`
	DeviceAssetCode         string     `json:"device_asset_code"`
	DeviceSerialNumber      string     `json:"device_serial_number,omitempty"`
	DeviceName              string     `json:"device_name"`
	DeviceType              string     `json:"device_type"`
	DeviceManufacturer      string     `json:"device_manufacturer,omitempty"`
	DeviceModel             string     `json:"device_model,omitempty"`
	DevicePurchasedAt       string     `json:"device_purchased_at,omitempty"`
	DeviceWarrantyExpiresAt string     `json:"device_warranty_expires_at,omitempty"`
	Category                string     `json:"category"`
	Priority                string     `json:"priority"`
	Title                   string     `json:"title"`
	Description             string     `json:"description"`
	Status                  string     `json:"status"`
	AssignedTo              string     `json:"assigned_to,omitempty"`
	AssignedToName          string     `json:"assigned_to_name,omitempty"`
	LastActorID             string     `json:"last_actor_id,omitempty"`
	LastActorName           string     `json:"last_actor_name,omitempty"`
	ResponseNote            string     `json:"response_note,omitempty"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
	AcceptedAt              *time.Time `json:"accepted_at,omitempty"`
	CompletedAt             *time.Time `json:"completed_at,omitempty"`
	RejectedAt              *time.Time `json:"rejected_at,omitempty"`
	CancelledAt             *time.Time `json:"cancelled_at,omitempty"`
}

type Filter struct {
	Status   string
	Priority string
	Category string
	Search   string
}

type AcceptUpdate struct {
	RequestID  string
	ActorID    string
	ActorName  string
	AcceptedAt time.Time
}

type CompleteUpdate struct {
	RequestID    string
	ActorID      string
	ActorName    string
	ResponseNote string
	CompletedAt  time.Time
}

type RejectUpdate struct {
	RequestID    string
	ActorID      string
	ActorName    string
	ResponseNote string
	RejectedAt   time.Time
}

type Repository interface {
	List(context.Context, Filter) ([]Request, error)
	ListByRequester(context.Context, string) ([]Request, error)
	FindForUpdate(context.Context, string) (Request, error)
	Create(context.Context, Request) (Request, error)
	Cancel(context.Context, string, string, time.Time) (Request, error)
	Accept(context.Context, AcceptUpdate) (Request, error)
	Complete(context.Context, CompleteUpdate) (Request, error)
	Reject(context.Context, RejectUpdate) (Request, error)
}

type DeviceFinder interface {
	FindByID(context.Context, string) (devices.Device, error)
}

type UserFinder interface {
	FindByID(context.Context, string) (auth.User, error)
}

type NotificationCreator interface {
	Create(context.Context, notifications.CreateInput) (notifications.Notification, error)
}

type TransactionManager interface {
	WithinTransaction(context.Context, func(context.Context) error) error
}
