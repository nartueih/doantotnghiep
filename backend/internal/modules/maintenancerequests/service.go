package maintenancerequests

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"license-manager/backend/internal/modules/auth"
	"license-manager/backend/internal/modules/devices"
	"license-manager/backend/internal/modules/notifications"
	"license-manager/backend/internal/platform/id"
)

type CreateInput struct {
	DeviceID    string
	Category    string
	Priority    string
	Title       string
	Description string
}

type CompleteInput struct {
	ResponseNote string
}

type RejectInput struct {
	ResponseNote string
}

type ListResult struct {
	Items     []Request `json:"items"`
	Total     int       `json:"total"`
	OpenCount int       `json:"open_count"`
}

type Service struct {
	repository    Repository
	devices       DeviceFinder
	users         UserFinder
	notifications NotificationCreator
	transactions  TransactionManager
	transitionMu  sync.Mutex
	now           func() time.Time
}

var uuidPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func NewService(repository Repository, deviceFinder DeviceFinder, userFinder UserFinder, notificationCreator NotificationCreator, transactions TransactionManager) *Service {
	return &Service{
		repository: repository, devices: deviceFinder, users: userFinder,
		notifications: notificationCreator, transactions: transactions, now: time.Now,
	}
}

func (s *Service) ListMine(ctx context.Context, requesterID string) (ListResult, error) {
	items, err := s.repository.ListByRequester(ctx, strings.TrimSpace(requesterID))
	if err != nil {
		return ListResult{}, err
	}
	openCount := 0
	for _, item := range items {
		if isOpenStatus(item.Status) {
			openCount++
		}
	}
	return ListResult{Items: items, Total: len(items), OpenCount: openCount}, nil
}

func (s *Service) ListAdmin(ctx context.Context, filter Filter) ([]Request, error) {
	filter.Status = strings.ToLower(strings.TrimSpace(filter.Status))
	filter.Priority = strings.ToLower(strings.TrimSpace(filter.Priority))
	filter.Category = strings.ToLower(strings.TrimSpace(filter.Category))
	filter.Search = strings.TrimSpace(filter.Search)
	if filter.Status != "" && !validStatus(filter.Status) {
		return nil, ErrInvalidData
	}
	if filter.Priority != "" && !validPriority(filter.Priority) {
		return nil, ErrInvalidPriority
	}
	if filter.Category != "" && !validCategory(filter.Category) {
		return nil, ErrInvalidCategory
	}
	return s.repository.List(ctx, filter)
}

func (s *Service) Create(ctx context.Context, requesterID string, input CreateInput) (Request, error) {
	s.transitionMu.Lock()
	defer s.transitionMu.Unlock()

	requesterID = strings.TrimSpace(requesterID)
	input.DeviceID = strings.TrimSpace(input.DeviceID)
	input.Category = strings.ToLower(strings.TrimSpace(input.Category))
	input.Priority = strings.ToLower(strings.TrimSpace(input.Priority))
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	if !validUUID(requesterID) || !validUUID(input.DeviceID) || input.Title == "" || input.Description == "" || utf8.RuneCountInString(input.Title) > 200 {
		return Request{}, ErrInvalidData
	}
	if !validCategory(input.Category) {
		return Request{}, ErrInvalidCategory
	}
	if !validPriority(input.Priority) {
		return Request{}, ErrInvalidPriority
	}
	requester, err := s.users.FindByID(ctx, requesterID)
	if errors.Is(err, auth.ErrUserNotFound) || (err == nil && (requester.Status != auth.StatusActive || requester.Role != auth.RoleEmployee)) {
		return Request{}, ErrRequesterUnavailable
	}
	if err != nil {
		return Request{}, err
	}
	device, err := s.devices.FindByID(ctx, input.DeviceID)
	if errors.Is(err, devices.ErrNotFound) || (err == nil && device.AssignedUserID != requester.ID) {
		return Request{}, ErrDeviceNotFound
	}
	if err != nil {
		return Request{}, err
	}
	requestID, err := id.NewUUID()
	if err != nil {
		return Request{}, err
	}
	now := s.now().UTC()
	return s.repository.Create(ctx, Request{
		ID: requestID, RequesterID: requester.ID, RequesterName: requester.FullName,
		DeviceID: device.ID, DeviceAssetCode: device.AssetCode, DeviceSerialNumber: device.SerialNumber,
		DeviceName: device.Name, DeviceType: device.DeviceType, DeviceManufacturer: device.Manufacturer,
		DeviceModel: device.Model, DevicePurchasedAt: device.PurchasedAt, DeviceWarrantyExpiresAt: device.WarrantyExpiresAt,
		Category: input.Category, Priority: input.Priority, Title: input.Title, Description: input.Description,
		Status: StatusPending, LastActorID: requester.ID, LastActorName: requester.FullName,
		CreatedAt: now, UpdatedAt: now,
	})
}

func (s *Service) Cancel(ctx context.Context, requesterID, requestID string) (Request, error) {
	s.transitionMu.Lock()
	defer s.transitionMu.Unlock()
	requesterID = strings.TrimSpace(requesterID)
	requestID = strings.TrimSpace(requestID)
	if !validUUID(requesterID) || !validUUID(requestID) {
		return Request{}, ErrInvalidData
	}
	return s.repository.Cancel(ctx, requestID, requesterID, s.now().UTC())
}

func (s *Service) Accept(ctx context.Context, reviewerID, requestID string) (Request, error) {
	s.transitionMu.Lock()
	defer s.transitionMu.Unlock()
	reviewerID = strings.TrimSpace(reviewerID)
	requestID = strings.TrimSpace(requestID)
	if !validUUID(reviewerID) || !validUUID(requestID) {
		return Request{}, ErrInvalidData
	}
	var accepted Request
	err := s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		item, err := s.requestForUpdate(txCtx, requestID, StatusPending)
		if err != nil {
			return err
		}
		reviewer, err := s.reviewer(txCtx, reviewerID)
		if err != nil {
			return err
		}
		accepted, err = s.repository.Accept(txCtx, AcceptUpdate{RequestID: item.ID, ActorID: reviewer.ID, ActorName: reviewer.FullName, AcceptedAt: s.now().UTC()})
		if err != nil {
			return err
		}
		_, err = s.notifications.Create(txCtx, notifications.CreateInput{
			UserID: item.RequesterID, Type: notifications.TypeMaintenanceAccepted,
			Title:      "Yêu cầu bảo trì đã được tiếp nhận",
			Message:    "Yêu cầu bảo trì " + item.DeviceAssetCode + " đã được " + reviewer.FullName + " tiếp nhận.",
			EntityType: notifications.EntityMaintenanceRequest, EntityID: item.ID,
		})
		return err
	})
	return accepted, err
}

func (s *Service) Complete(ctx context.Context, reviewerID, requestID string, input CompleteInput) (Request, error) {
	s.transitionMu.Lock()
	defer s.transitionMu.Unlock()
	reviewerID = strings.TrimSpace(reviewerID)
	requestID = strings.TrimSpace(requestID)
	input.ResponseNote = strings.TrimSpace(input.ResponseNote)
	if !validUUID(reviewerID) || !validUUID(requestID) || input.ResponseNote == "" {
		return Request{}, ErrInvalidData
	}
	var completed Request
	err := s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		item, err := s.requestForUpdate(txCtx, requestID, StatusInProgress)
		if err != nil {
			return err
		}
		reviewer, err := s.reviewer(txCtx, reviewerID)
		if err != nil {
			return err
		}
		completed, err = s.repository.Complete(txCtx, CompleteUpdate{RequestID: item.ID, ActorID: reviewer.ID, ActorName: reviewer.FullName, ResponseNote: input.ResponseNote, CompletedAt: s.now().UTC()})
		if err != nil {
			return err
		}
		_, err = s.notifications.Create(txCtx, notifications.CreateInput{
			UserID: item.RequesterID, Type: notifications.TypeMaintenanceCompleted,
			Title: "Yêu cầu bảo trì đã hoàn thành", Message: input.ResponseNote,
			EntityType: notifications.EntityMaintenanceRequest, EntityID: item.ID,
		})
		return err
	})
	return completed, err
}

func (s *Service) Reject(ctx context.Context, reviewerID, requestID string, input RejectInput) (Request, error) {
	s.transitionMu.Lock()
	defer s.transitionMu.Unlock()
	reviewerID = strings.TrimSpace(reviewerID)
	requestID = strings.TrimSpace(requestID)
	input.ResponseNote = strings.TrimSpace(input.ResponseNote)
	if !validUUID(reviewerID) || !validUUID(requestID) || input.ResponseNote == "" {
		return Request{}, ErrInvalidData
	}
	var rejected Request
	err := s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
		item, err := s.repository.FindForUpdate(txCtx, requestID)
		if err != nil {
			return err
		}
		if item.Status != StatusPending && item.Status != StatusInProgress {
			return ErrInvalidState
		}
		reviewer, err := s.reviewer(txCtx, reviewerID)
		if err != nil {
			return err
		}
		rejected, err = s.repository.Reject(txCtx, RejectUpdate{RequestID: item.ID, ActorID: reviewer.ID, ActorName: reviewer.FullName, ResponseNote: input.ResponseNote, RejectedAt: s.now().UTC()})
		if err != nil {
			return err
		}
		_, err = s.notifications.Create(txCtx, notifications.CreateInput{
			UserID: item.RequesterID, Type: notifications.TypeMaintenanceRejected,
			Title: "Yêu cầu bảo trì đã được phản hồi", Message: input.ResponseNote,
			EntityType: notifications.EntityMaintenanceRequest, EntityID: item.ID,
		})
		return err
	})
	return rejected, err
}

func (s *Service) requestForUpdate(ctx context.Context, requestID, expectedStatus string) (Request, error) {
	item, err := s.repository.FindForUpdate(ctx, requestID)
	if err != nil {
		return Request{}, err
	}
	if item.Status != expectedStatus {
		return Request{}, ErrInvalidState
	}
	return item, nil
}

func (s *Service) reviewer(ctx context.Context, reviewerID string) (auth.User, error) {
	reviewer, err := s.users.FindByID(ctx, reviewerID)
	if errors.Is(err, auth.ErrUserNotFound) || (err == nil && (reviewer.Status != auth.StatusActive || (reviewer.Role != auth.RoleAdmin && reviewer.Role != auth.RoleITManager))) {
		return auth.User{}, ErrReviewerUnavailable
	}
	if err != nil {
		return auth.User{}, err
	}
	return reviewer, nil
}

func validCategory(value string) bool {
	return value == CategoryHardware || value == CategorySoftware || value == CategoryNetwork || value == CategoryAccessory || value == CategoryOther
}

func validPriority(value string) bool {
	return value == PriorityNormal || value == PriorityHigh || value == PriorityUrgent
}

func validStatus(value string) bool {
	return value == StatusPending || value == StatusInProgress || value == StatusCompleted || value == StatusRejected || value == StatusCancelled
}

func validUUID(value string) bool {
	return uuidPattern.MatchString(value)
}
