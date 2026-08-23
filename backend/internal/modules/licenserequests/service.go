package licenserequests

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"license-manager/backend/internal/modules/assignments"
	"license-manager/backend/internal/modules/auth"
	"license-manager/backend/internal/modules/licenses"
	"license-manager/backend/internal/modules/notifications"
	"license-manager/backend/internal/modules/software"
	"license-manager/backend/internal/platform/id"
)

type CreateInput struct {
	SoftwareProductID string
	Priority          string
	Reason            string
}

type ApproveInput struct {
	LicenseID    string
	ResponseNote string
}

type RejectInput struct {
	DecisionReason string
	ResponseNote   string
}

type Service struct {
	repository    Repository
	software      SoftwareCatalog
	licenses      LicenseFinder
	users         UserFinder
	assignments   AssignmentCreator
	notifications NotificationCreator
	transitionMu  sync.Mutex
	now           func() time.Time
}

func NewService(repository Repository, softwareCatalog SoftwareCatalog, licenseFinder LicenseFinder, userFinder UserFinder, assignmentCreator AssignmentCreator, notificationCreator NotificationCreator) *Service {
	return &Service{
		repository: repository, software: softwareCatalog, licenses: licenseFinder,
		users: userFinder, assignments: assignmentCreator, notifications: notificationCreator, now: time.Now,
	}
}

func (s *Service) RequestableSoftware(ctx context.Context) ([]software.Product, error) {
	return s.software.List(ctx)
}

func (s *Service) ListMine(ctx context.Context, requesterID string) ([]Request, error) {
	return s.repository.ListByRequester(ctx, strings.TrimSpace(requesterID))
}

func (s *Service) ListAdmin(ctx context.Context, filter Filter) ([]Request, error) {
	filter.Status = strings.TrimSpace(filter.Status)
	filter.Priority = strings.TrimSpace(filter.Priority)
	filter.Search = strings.TrimSpace(filter.Search)
	if filter.Status != "" && !validStatus(filter.Status) {
		return nil, ErrInvalidData
	}
	if filter.Priority != "" && !validPriority(filter.Priority) {
		return nil, ErrInvalidPriority
	}
	return s.repository.List(ctx, filter)
}

func (s *Service) Create(ctx context.Context, requesterID string, input CreateInput) (Request, error) {
	s.transitionMu.Lock()
	defer s.transitionMu.Unlock()

	requesterID = strings.TrimSpace(requesterID)
	input.SoftwareProductID = strings.TrimSpace(input.SoftwareProductID)
	input.Priority = strings.TrimSpace(input.Priority)
	input.Reason = strings.TrimSpace(input.Reason)
	if requesterID == "" || input.SoftwareProductID == "" || input.Reason == "" {
		return Request{}, ErrInvalidData
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
	product, err := s.software.FindByID(ctx, input.SoftwareProductID)
	if errors.Is(err, software.ErrNotFound) {
		return Request{}, ErrSoftwareNotFound
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
		SoftwareProductID: product.ID, SoftwareProductName: product.Name,
		Priority: input.Priority, Reason: input.Reason, Status: StatusPending,
		CreatedAt: now, UpdatedAt: now,
	})
}

func (s *Service) Cancel(ctx context.Context, requesterID, requestID string) (Request, error) {
	s.transitionMu.Lock()
	defer s.transitionMu.Unlock()
	return s.repository.Cancel(ctx, strings.TrimSpace(requestID), strings.TrimSpace(requesterID), s.now().UTC())
}

func (s *Service) Approve(ctx context.Context, reviewerID, requestID string, input ApproveInput) (Request, error) {
	s.transitionMu.Lock()
	defer s.transitionMu.Unlock()

	reviewerID = strings.TrimSpace(reviewerID)
	requestID = strings.TrimSpace(requestID)
	input.LicenseID = strings.TrimSpace(input.LicenseID)
	input.ResponseNote = strings.TrimSpace(input.ResponseNote)
	if reviewerID == "" || requestID == "" || input.LicenseID == "" {
		return Request{}, ErrInvalidData
	}
	item, err := s.pendingRequest(ctx, requestID)
	if err != nil {
		return Request{}, err
	}
	reviewer, err := s.reviewer(ctx, reviewerID)
	if err != nil {
		return Request{}, err
	}
	license, err := s.licenses.FindByID(ctx, input.LicenseID)
	if errors.Is(err, licenses.ErrNotFound) {
		return Request{}, ErrLicenseNotFound
	}
	if err != nil {
		return Request{}, err
	}
	if license.SoftwareProductID != item.SoftwareProductID {
		return Request{}, ErrLicenseProductMismatch
	}
	assignment, err := s.assignments.Create(ctx, reviewer.ID, assignments.CreateInput{
		LicenseID: license.ID,
		UserID:    item.RequesterID,
		Notes:     fmt.Sprintf("Cấp từ yêu cầu license %s", item.ID),
	})
	if err != nil {
		return Request{}, err
	}
	now := s.now().UTC()
	approved, err := s.repository.Approve(ctx, ApprovalUpdate{
		RequestID: item.ID, LicenseID: license.ID, LicenseName: license.Name,
		AssignmentID: assignment.ID, ReviewerID: reviewer.ID, ReviewerName: reviewer.FullName,
		ResponseNote: input.ResponseNote, ReviewedAt: now,
	})
	if err != nil {
		return Request{}, err
	}
	message := input.ResponseNote
	if message == "" {
		message = fmt.Sprintf("Yêu cầu %s đã được duyệt và license đã được cấp cho bạn.", item.SoftwareProductName)
	}
	if _, err := s.notifications.Create(ctx, notifications.CreateInput{
		UserID: item.RequesterID, Type: notifications.TypeLicenseRequestApproved,
		Title: "Yêu cầu license đã được duyệt", Message: message,
		EntityType: notifications.EntityLicenseRequest, EntityID: item.ID,
	}); err != nil {
		return Request{}, err
	}
	return approved, nil
}

func (s *Service) Reject(ctx context.Context, reviewerID, requestID string, input RejectInput) (Request, error) {
	s.transitionMu.Lock()
	defer s.transitionMu.Unlock()

	reviewerID = strings.TrimSpace(reviewerID)
	requestID = strings.TrimSpace(requestID)
	input.DecisionReason = strings.TrimSpace(input.DecisionReason)
	input.ResponseNote = strings.TrimSpace(input.ResponseNote)
	if reviewerID == "" || requestID == "" || input.ResponseNote == "" {
		return Request{}, ErrInvalidData
	}
	if !validDecision(input.DecisionReason) {
		return Request{}, ErrInvalidDecision
	}
	item, err := s.pendingRequest(ctx, requestID)
	if err != nil {
		return Request{}, err
	}
	reviewer, err := s.reviewer(ctx, reviewerID)
	if err != nil {
		return Request{}, err
	}
	rejected, err := s.repository.Reject(ctx, RejectionUpdate{
		RequestID: item.ID, ReviewerID: reviewer.ID, ReviewerName: reviewer.FullName,
		DecisionReason: input.DecisionReason, ResponseNote: input.ResponseNote, ReviewedAt: s.now().UTC(),
	})
	if err != nil {
		return Request{}, err
	}
	if _, err := s.notifications.Create(ctx, notifications.CreateInput{
		UserID: item.RequesterID, Type: notifications.TypeLicenseRequestRejected,
		Title: "Yêu cầu license đã được phản hồi", Message: input.ResponseNote,
		EntityType: notifications.EntityLicenseRequest, EntityID: item.ID,
	}); err != nil {
		return Request{}, err
	}
	return rejected, nil
}

func (s *Service) pendingRequest(ctx context.Context, requestID string) (Request, error) {
	item, err := s.repository.FindByID(ctx, requestID)
	if err != nil {
		return Request{}, err
	}
	if item.Status != StatusPending {
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

func validPriority(value string) bool {
	return value == PriorityNormal || value == PriorityHigh || value == PriorityUrgent
}

func validDecision(value string) bool {
	return value == DecisionOutOfStock || value == DecisionNotApproved || value == DecisionOther
}

func validStatus(value string) bool {
	return value == StatusPending || value == StatusApproved || value == StatusRejected || value == StatusCancelled
}
