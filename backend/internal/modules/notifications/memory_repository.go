package notifications

import (
	"context"
	"sort"
	"sync"
	"time"
)

type MemoryRepository struct {
	mu            sync.RWMutex
	notifications map[string]Notification
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{notifications: make(map[string]Notification)}
}

func (r *MemoryRepository) ListByUser(_ context.Context, userID string) ([]Notification, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]Notification, 0)
	for _, item := range r.notifications {
		if item.UserID == userID {
			items = append(items, cloneNotification(item))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	return items, nil
}

func (r *MemoryRepository) Create(_ context.Context, item Notification) (Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.notifications[item.ID] = cloneNotification(item)
	return cloneNotification(item), nil
}

func (r *MemoryRepository) MarkRead(_ context.Context, userID, notificationID string, readAt time.Time) (Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	item, exists := r.notifications[notificationID]
	if !exists || item.UserID != userID {
		return Notification{}, ErrNotFound
	}
	if item.ReadAt == nil {
		item.ReadAt = &readAt
		r.notifications[item.ID] = cloneNotification(item)
	}
	return cloneNotification(item), nil
}

func (r *MemoryRepository) MarkAllRead(_ context.Context, userID string, readAt time.Time) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	updated := 0
	for notificationID, item := range r.notifications {
		if item.UserID != userID || item.ReadAt != nil {
			continue
		}
		item.ReadAt = &readAt
		r.notifications[notificationID] = cloneNotification(item)
		updated++
	}
	return updated, nil
}

func cloneNotification(item Notification) Notification {
	if item.ReadAt != nil {
		readAt := *item.ReadAt
		item.ReadAt = &readAt
	}
	return item
}
