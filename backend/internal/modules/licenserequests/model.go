package licenserequests

import (
	"context"
	"errors"
	"time"

	"license-manager/backend/internal/modules/assignments"
	"license-manager/backend/internal/modules/auth"
	"license-manager/backend/internal/modules/licenses"
	"license-manager/backend/internal/modules/notifications"
	"license-manager/backend/internal/modules/software"
)

const (
	PriorityNormal = "normal"
	PriorityHigh   = "high"
	PriorityUrgent = "urgent"

	StatusPending   = "pending"
	StatusApproved  = "approved"
	StatusRejected  = "rejected"
	StatusCancelled = "cancelled"

	DecisionOutOfStock  = "out_of_stock"
	DecisionNotApproved = "not_approved"
	DecisionOther       = "other"
)

var (
	ErrNotFound               = errors.New("license request not found")
	ErrInvalidData            = errors.New("software, priority and reason are required")
	ErrInvalidPriority        = errors.New("priority must be normal, high or urgent")
	ErrInvalidDecision        = errors.New("decision reason must be out_of_stock, not_approved or other")
	ErrPendingDuplicate       = errors.New("a pending request already exists for this software")
	ErrInvalidState           = errors.New("license request is no longer pending")
	ErrRequesterUnavailable   = errors.New("requester does not exist or is unavailable")
	ErrReviewerUnavailable    = errors.New("reviewer does not exist or is unavailable")
	ErrSoftwareNotFound       = errors.New("software product not found")
	ErrLicenseNotFound        = errors.New("license not found")
	ErrLicenseProductMismatch = errors.New("license does not belong to the requested software")
)

type Request struct {
	ID                  string     `json:"id"`
	RequesterID         string     `json:"requester_id"`
	RequesterName       string     `json:"requester_name"`
	SoftwareProductID   string     `json:"software_product_id"`
	SoftwareProductName string     `json:"software_product_name"`
	Priority            string     `json:"priority"`
	Reason              string     `json:"reason"`
	Status              string     `json:"status"`
	SelectedLicenseID   string     `json:"selected_license_id,omitempty"`
	SelectedLicenseName string     `json:"selected_license_name,omitempty"`
	AssignmentID        string     `json:"assignment_id,omitempty"`
	ReviewedBy          string     `json:"reviewed_by,omitempty"`
	ReviewedByName      string     `json:"reviewed_by_name,omitempty"`
	DecisionReason      string     `json:"decision_reason,omitempty"`
	ResponseNote        string     `json:"response_note,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	ReviewedAt          *time.Time `json:"reviewed_at,omitempty"`
	CancelledAt         *time.Time `json:"cancelled_at,omitempty"`
}

type Filter struct {
	Status   string
	Priority string
	Search   string
}

type ApprovalUpdate struct {
	RequestID    string
	LicenseID    string
	LicenseName  string
	AssignmentID string
	ReviewerID   string
	ReviewerName string
	ResponseNote string
	ReviewedAt   time.Time
}

type RejectionUpdate struct {
	RequestID      string
	ReviewerID     string
	ReviewerName   string
	DecisionReason string
	ResponseNote   string
	ReviewedAt     time.Time
}

type Repository interface {
	List(context.Context, Filter) ([]Request, error)
	ListByRequester(context.Context, string) ([]Request, error)
	FindByID(context.Context, string) (Request, error)
	FindForUpdate(context.Context, string) (Request, error)
	Create(context.Context, Request) (Request, error)
	Cancel(context.Context, string, string, time.Time) (Request, error)
	Approve(context.Context, ApprovalUpdate) (Request, error)
	Reject(context.Context, RejectionUpdate) (Request, error)
}

type SoftwareCatalog interface {
	List(context.Context) ([]software.Product, error)
	FindByID(context.Context, string) (software.Product, error)
}

type LicenseFinder interface {
	FindByID(context.Context, string) (licenses.License, error)
}

type UserFinder interface {
	FindByID(context.Context, string) (auth.User, error)
}

type AssignmentCreator interface {
	Create(context.Context, string, assignments.CreateInput) (assignments.Assignment, error)
}

type NotificationCreator interface {
	Create(context.Context, notifications.CreateInput) (notifications.Notification, error)
}

type TransactionManager interface {
	WithinTransaction(context.Context, func(context.Context) error) error
}
