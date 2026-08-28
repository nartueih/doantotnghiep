package audit

import (
	"context"
	"time"
)

const (
	ActionCreate       = "create"
	ActionUpdate       = "update"
	ActionStatusChange = "status_change"
	ActionAssign       = "assign"
	ActionRevoke       = "revoke"
	ActionViewKey      = "view_key"
	ActionArchive      = "archive"
	ActionRequest      = "request"
	ActionCancel       = "cancel"
	ActionApprove      = "approve"
	ActionReject       = "reject"
	ActionAccept       = "accept"
	ActionComplete     = "complete"

	EntityUser               = "user"
	EntityDepartment         = "department"
	EntitySoftware           = "software_product"
	EntityLicense            = "license"
	EntityDevice             = "device"
	EntityAssignment         = "license_assignment"
	EntityLicenseRequest     = "license_request"
	EntityMaintenanceRequest = "maintenance_request"
)

type Log struct {
	ID         string         `json:"id"`
	ActorID    string         `json:"actor_id,omitempty"`
	ActorName  string         `json:"actor_name,omitempty"`
	ActorEmail string         `json:"actor_email,omitempty"`
	Action     string         `json:"action"`
	EntityType string         `json:"entity_type"`
	EntityID   string         `json:"entity_id,omitempty"`
	Metadata   map[string]any `json:"metadata"`
	IPAddress  string         `json:"ip_address,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

type Filter struct {
	Action     string
	EntityType string
	ActorID    string
	Limit      int
}

type RecordInput struct {
	ActorID    string
	Action     string
	EntityType string
	EntityID   string
	Metadata   map[string]any
	IPAddress  string
}

type Repository interface {
	List(context.Context, Filter) ([]Log, error)
	Create(context.Context, Log) (Log, error)
}

type Recorder interface {
	Record(context.Context, RecordInput) (Log, error)
}
