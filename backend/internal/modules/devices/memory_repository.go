package devices

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"
)

type MemoryRepository struct {
	mu      sync.RWMutex
	devices map[string]Device
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{devices: make(map[string]Device)}
}

func (r *MemoryRepository) List(_ context.Context) ([]Device, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]Device, 0, len(r.devices))
	for _, item := range r.devices {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].AssetCode < items[j].AssetCode })
	return items, nil
}

func (r *MemoryRepository) FindByID(_ context.Context, deviceID string) (Device, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, exists := r.devices[deviceID]
	if !exists {
		return Device{}, ErrNotFound
	}
	return item, nil
}

func (r *MemoryRepository) Create(_ context.Context, item Device) (Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.checkUnique(item, ""); err != nil {
		return Device{}, err
	}
	r.devices[item.ID] = item
	return item, nil
}

func (r *MemoryRepository) Update(_ context.Context, item Device) (Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	existing, exists := r.devices[item.ID]
	if !exists {
		return Device{}, ErrNotFound
	}
	if err := r.checkUnique(item, item.ID); err != nil {
		return Device{}, err
	}
	item.AssignedUserID = existing.AssignedUserID
	item.AssignedUserName = existing.AssignedUserName
	item.Status = existing.Status
	item.CreatedAt = existing.CreatedAt
	r.devices[item.ID] = item
	return item, nil
}

func (r *MemoryRepository) UpdateStatus(_ context.Context, deviceID, status string) (Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, exists := r.devices[deviceID]
	if !exists {
		return Device{}, ErrNotFound
	}
	item.Status = status
	item.UpdatedAt = time.Now().UTC()
	r.devices[deviceID] = item
	return item, nil
}

func (r *MemoryRepository) Assign(_ context.Context, deviceID, userID, userName string) (Device, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, exists := r.devices[deviceID]
	if !exists {
		return Device{}, ErrNotFound
	}
	if userID != "" && (item.Status != StatusAvailable || item.AssignedUserID != "") {
		return Device{}, ErrDeviceUnavailable
	}
	item.AssignedUserID = userID
	item.AssignedUserName = userName
	if userID == "" {
		item.AssignedUserName = ""
		item.Status = StatusAvailable
	} else {
		item.Status = StatusAssigned
	}
	item.UpdatedAt = time.Now().UTC()
	r.devices[deviceID] = item
	return item, nil
}

func (r *MemoryRepository) checkUnique(candidate Device, ignoredID string) error {
	for id, item := range r.devices {
		if id == ignoredID {
			continue
		}
		if strings.EqualFold(item.AssetCode, candidate.AssetCode) {
			return ErrAssetCodeExists
		}
		if candidate.SerialNumber != "" && strings.EqualFold(item.SerialNumber, candidate.SerialNumber) {
			return ErrSerialExists
		}
	}
	return nil
}
