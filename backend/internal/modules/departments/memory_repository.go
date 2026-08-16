package departments

import (
	"context"
	"sort"
	"strings"
	"sync"
)

type MemoryRepository struct {
	mu          sync.RWMutex
	departments map[string]Department
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{departments: make(map[string]Department)}
}

func (r *MemoryRepository) List(_ context.Context) ([]Department, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]Department, 0, len(r.departments))
	for _, item := range r.departments {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Code < items[j].Code })
	return items, nil
}

func (r *MemoryRepository) FindByID(_ context.Context, departmentID string) (Department, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, exists := r.departments[departmentID]
	if !exists {
		return Department{}, ErrNotFound
	}
	return item, nil
}

func (r *MemoryRepository) Create(_ context.Context, item Department) (Department, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.checkUnique(item, ""); err != nil {
		return Department{}, err
	}
	r.departments[item.ID] = item
	return item, nil
}

func (r *MemoryRepository) Update(_ context.Context, item Department) (Department, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, exists := r.departments[item.ID]
	if !exists {
		return Department{}, ErrNotFound
	}
	if err := r.checkUnique(item, item.ID); err != nil {
		return Department{}, err
	}
	item.CreatedAt = existing.CreatedAt
	r.departments[item.ID] = item
	return item, nil
}

func (r *MemoryRepository) checkUnique(candidate Department, ignoredID string) error {
	for id, item := range r.departments {
		if id == ignoredID {
			continue
		}
		if strings.EqualFold(item.Name, candidate.Name) {
			return ErrNameExists
		}
		if strings.EqualFold(item.Code, candidate.Code) {
			return ErrCodeExists
		}
	}
	return nil
}
