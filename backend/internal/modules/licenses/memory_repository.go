package licenses

import (
	"context"
	"sort"
	"sync"
)

type MemoryRepository struct {
	mu       sync.RWMutex
	licenses map[string]License
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{licenses: make(map[string]License)}
}

func (r *MemoryRepository) List(_ context.Context) ([]License, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]License, 0, len(r.licenses))
	for _, item := range r.licenses {
		items = append(items, clone(item))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}

func (r *MemoryRepository) Create(_ context.Context, item License) (License, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.licenses[item.ID] = clone(item)
	return clone(item), nil
}

func (r *MemoryRepository) Update(_ context.Context, item License) (License, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, exists := r.licenses[item.ID]
	if !exists {
		return License{}, ErrNotFound
	}
	if item.SeatCount < existing.UsedSeats {
		return License{}, ErrSeatCountBelowUsage
	}
	if len(item.EncryptedKey) == 0 {
		item.EncryptedKey = append([]byte(nil), existing.EncryptedKey...)
		item.KeyHint = existing.KeyHint
	}
	item.CreatedAt = existing.CreatedAt
	item.UsedSeats = existing.UsedSeats
	item.AvailableSeats = item.SeatCount - item.UsedSeats
	r.licenses[item.ID] = clone(item)
	return clone(item), nil
}

func clone(item License) License {
	item.EncryptedKey = append([]byte(nil), item.EncryptedKey...)
	return item
}
