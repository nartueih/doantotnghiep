package maintenancerequests

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
		if filter.Category != "" && item.Category != filter.Category {
			continue
		}
		if search != "" && !matchesSearch(item, search) {
			continue
		}
		items = append(items, cloneRequest(item))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}

func matchesSearch(item Request, search string) bool {
	for _, value := range []string{item.RequesterName, item.DeviceAssetCode, item.DeviceSerialNumber, item.DeviceName, item.Title} {
		if strings.Contains(strings.ToLower(value), search) {
			return true
		}
	}
	return false
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
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}

func (r *MemoryRepository) FindForUpdate(_ context.Context, requestID string) (Request, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, exists := r.requests[requestID]
	if !exists {
		return Request{}, ErrNotFound
	}
	return cloneRequest(item), nil
}

func (r *MemoryRepository) Create(_ context.Context, item Request) (Request, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.requests {
		if existing.DeviceID == item.DeviceID && isOpenStatus(existing.Status) {
			return Request{}, ErrOpenDuplicate
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
	item.LastActorID = item.RequesterID
	item.LastActorName = item.RequesterName
	item.CancelledAt = &cancelledAt
	item.UpdatedAt = cancelledAt
	r.requests[item.ID] = cloneRequest(item)
	return cloneRequest(item), nil
}

func (r *MemoryRepository) Accept(_ context.Context, update AcceptUpdate) (Request, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, exists := r.requests[update.RequestID]
	if !exists {
		return Request{}, ErrNotFound
	}
	if item.Status != StatusPending {
		return Request{}, ErrInvalidState
	}
	item.Status = StatusInProgress
	item.AssignedTo = update.ActorID
	item.AssignedToName = update.ActorName
	item.LastActorID = update.ActorID
	item.LastActorName = update.ActorName
	item.AcceptedAt = &update.AcceptedAt
	item.UpdatedAt = update.AcceptedAt
	r.requests[item.ID] = cloneRequest(item)
	return cloneRequest(item), nil
}

func (r *MemoryRepository) Complete(_ context.Context, update CompleteUpdate) (Request, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, exists := r.requests[update.RequestID]
	if !exists {
		return Request{}, ErrNotFound
	}
	if item.Status != StatusInProgress {
		return Request{}, ErrInvalidState
	}
	item.Status = StatusCompleted
	item.LastActorID = update.ActorID
	item.LastActorName = update.ActorName
	item.ResponseNote = update.ResponseNote
	item.CompletedAt = &update.CompletedAt
	item.UpdatedAt = update.CompletedAt
	r.requests[item.ID] = cloneRequest(item)
	return cloneRequest(item), nil
}

func (r *MemoryRepository) Reject(_ context.Context, update RejectUpdate) (Request, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, exists := r.requests[update.RequestID]
	if !exists {
		return Request{}, ErrNotFound
	}
	if item.Status != StatusPending && item.Status != StatusInProgress {
		return Request{}, ErrInvalidState
	}
	item.Status = StatusRejected
	item.LastActorID = update.ActorID
	item.LastActorName = update.ActorName
	item.ResponseNote = update.ResponseNote
	item.RejectedAt = &update.RejectedAt
	item.UpdatedAt = update.RejectedAt
	r.requests[item.ID] = cloneRequest(item)
	return cloneRequest(item), nil
}

func isOpenStatus(status string) bool {
	return status == StatusPending || status == StatusInProgress
}

func cloneRequest(item Request) Request {
	item.AcceptedAt = cloneTime(item.AcceptedAt)
	item.CompletedAt = cloneTime(item.CompletedAt)
	item.RejectedAt = cloneTime(item.RejectedAt)
	item.CancelledAt = cloneTime(item.CancelledAt)
	return item
}

func cloneTime(source *time.Time) *time.Time {
	if source == nil {
		return nil
	}
	value := *source
	return &value
}
