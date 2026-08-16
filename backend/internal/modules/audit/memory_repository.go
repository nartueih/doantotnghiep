package audit

import (
	"context"
	"sort"
	"sync"
)

type MemoryRepository struct {
	mu   sync.RWMutex
	logs []Log
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{logs: make([]Log, 0)}
}

func (r *MemoryRepository) List(_ context.Context, filter Filter) ([]Log, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]Log, 0)
	for _, item := range r.logs {
		if filter.Action != "" && item.Action != filter.Action {
			continue
		}
		if filter.EntityType != "" && item.EntityType != filter.EntityType {
			continue
		}
		if filter.ActorID != "" && item.ActorID != filter.ActorID {
			continue
		}
		items = append(items, cloneLog(item))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	if filter.Limit > 0 && len(items) > filter.Limit {
		items = items[:filter.Limit]
	}
	return items, nil
}

func (r *MemoryRepository) Create(_ context.Context, item Log) (Log, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item = cloneLog(item)
	r.logs = append(r.logs, item)
	return cloneLog(item), nil
}

func cloneLog(item Log) Log {
	item.Metadata = sanitizeMetadata(item.Metadata)
	return item
}
