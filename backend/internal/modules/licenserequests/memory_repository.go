package licenserequests

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
)

type MemoryRepository struct {
	mu       sync.RWMutex
	requests map[string]Request
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{requests: make(map[string]Request)}
}

func (r *MemoryRepository) List(_ context.Context, filter Filter) ([]Request, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	search := strings.ToLower(strings.TrimSpace(filter.Search))
	items := make([]Request, 0)
	for _, item := range r.requests {
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		if filter.Priority != "" && item.Priority != filter.Priority {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(item.RequesterName), search) &&
			!strings.Contains(strings.ToLower(item.SoftwareProductName), search) {
			continue
		}
		items = append(items, cloneRequest(item))
	}
	sortRequests(items)
	return items, nil
}

func (r *MemoryRepository) ListByRequester(_ context.Context, requesterID string) ([]Request, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]Request, 0)
	for _, item := range r.requests {
		if item.RequesterID == requesterID {
			items = append(items, cloneRequest(item))
		}
	}
	sortRequests(items)
	return items, nil
}

func (r *MemoryRepository) FindByID(_ context.Context, requestID string) (Request, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	item, exists := r.requests[requestID]
	if !exists {
		return Request{}, ErrNotFound
	}
	return cloneRequest(item), nil
}

func (r *MemoryRepository) FindForUpdate(ctx context.Context, requestID string) (Request, error) {
	return r.FindByID(ctx, requestID)
}

func (r *MemoryRepository) Create(_ context.Context, item Request) (Request, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, existing := range r.requests {
		if existing.Status == StatusPending && existing.RequesterID == item.RequesterID && existing.SoftwareProductID == item.SoftwareProductID {
			return Request{}, ErrPendingDuplicate
		}
	}
	r.requests[item.ID] = cloneRequest(item)
	return cloneRequest(item), nil
}

func (r *MemoryRepository) Cancel(_ context.Context, requestID, requesterID string, cancelledAt time.Time) (Request, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	item, exists := r.requests[requestID]
	if !exists || item.RequesterID != requesterID {
		return Request{}, ErrNotFound
	}
	if item.Status != StatusPending {
		return Request{}, ErrInvalidState
	}
	item.Status = StatusCancelled
	item.CancelledAt = &cancelledAt
	item.UpdatedAt = cancelledAt
	r.requests[item.ID] = cloneRequest(item)
	return cloneRequest(item), nil
}

func (r *MemoryRepository) Approve(_ context.Context, update ApprovalUpdate) (Request, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	item, exists := r.requests[update.RequestID]
	if !exists {
		return Request{}, ErrNotFound
	}
	if item.Status != StatusPending {
		return Request{}, ErrInvalidState
	}
	item.Status = StatusApproved
	item.SelectedLicenseID = update.LicenseID
	item.SelectedLicenseName = update.LicenseName
	item.AssignmentID = update.AssignmentID
	item.ReviewedBy = update.ReviewerID
	item.ReviewedByName = update.ReviewerName
	item.ResponseNote = update.ResponseNote
	item.ReviewedAt = &update.ReviewedAt
	item.UpdatedAt = update.ReviewedAt
	r.requests[item.ID] = cloneRequest(item)
	return cloneRequest(item), nil
}

func (r *MemoryRepository) Reject(_ context.Context, update RejectionUpdate) (Request, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	item, exists := r.requests[update.RequestID]
	if !exists {
		return Request{}, ErrNotFound
	}
	if item.Status != StatusPending {
		return Request{}, ErrInvalidState
	}
	item.Status = StatusRejected
	item.ReviewedBy = update.ReviewerID
	item.ReviewedByName = update.ReviewerName
	item.DecisionReason = update.DecisionReason
	item.ResponseNote = update.ResponseNote
	item.ReviewedAt = &update.ReviewedAt
	item.UpdatedAt = update.ReviewedAt
	r.requests[item.ID] = cloneRequest(item)
	return cloneRequest(item), nil
}

func sortRequests(items []Request) {
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
}

func cloneRequest(item Request) Request {
	if item.ReviewedAt != nil {
		reviewedAt := *item.ReviewedAt
		item.ReviewedAt = &reviewedAt
	}
	if item.CancelledAt != nil {
		cancelledAt := *item.CancelledAt
		item.CancelledAt = &cancelledAt
	}
	return item
}
