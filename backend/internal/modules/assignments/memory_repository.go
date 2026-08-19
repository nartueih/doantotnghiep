package assignments

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"license-manager/backend/internal/modules/licenses"
)

type MemoryRepository struct {
	mu          sync.RWMutex
	assignments map[string]Assignment
	licenses    *licenses.MemoryRepository
}

func NewMemoryRepository(licenseRepository *licenses.MemoryRepository) *MemoryRepository {
	return &MemoryRepository{
		assignments: make(map[string]Assignment),
		licenses:    licenseRepository,
	}
}

func (r *MemoryRepository) List(_ context.Context) ([]Assignment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]Assignment, 0, len(r.assignments))
	for _, item := range r.assignments {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].AssignedAt.Before(items[j].AssignedAt) })
	return items, nil
}

func (r *MemoryRepository) Create(_ context.Context, item Assignment) (Assignment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.assignments {
		if existing.Status != StatusActive || existing.LicenseID != item.LicenseID {
			continue
		}
		if (item.UserID != "" && existing.UserID == item.UserID) ||
			(item.DeviceID != "" && existing.DeviceID == item.DeviceID) {
			return Assignment{}, ErrDuplicate
		}
	}
	if err := r.licenses.ReserveSeat(item.LicenseID); errors.Is(err, licenses.ErrNoAvailableSeats) {
		return Assignment{}, ErrNoAvailableSeats
	} else if errors.Is(err, licenses.ErrArchived) {
		return Assignment{}, ErrLicenseInactive
	} else if err != nil {
		return Assignment{}, err
	}
	r.assignments[item.ID] = item
	return item, nil
}

func (r *MemoryRepository) Revoke(_ context.Context, assignmentID, actorID, actorName string, revokedAt time.Time) (Assignment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, exists := r.assignments[assignmentID]
	if !exists || item.Status != StatusActive {
		return Assignment{}, ErrNotFound
	}
	item.Status = StatusRevoked
	item.RevokedAt = &revokedAt
	item.RevokedBy = actorID
	item.RevokedByName = actorName
	r.assignments[assignmentID] = item
	r.licenses.ReleaseSeat(item.LicenseID)
	return item, nil
}
