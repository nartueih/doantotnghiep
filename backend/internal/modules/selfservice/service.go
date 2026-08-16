package selfservice

import (
	"context"
	"sort"
	"strings"
	"time"

	"license-manager/backend/internal/modules/assignments"
	"license-manager/backend/internal/modules/devices"
	"license-manager/backend/internal/modules/licenses"
)

type Service struct {
	devices     DeviceLister
	assignments AssignmentLister
	licenses    LicenseLister
	now         func() time.Time
}

func NewService(deviceLister DeviceLister, assignmentLister AssignmentLister, licenseLister LicenseLister) *Service {
	return &Service{devices: deviceLister, assignments: assignmentLister, licenses: licenseLister, now: time.Now}
}

func (s *Service) Devices(ctx context.Context, currentUserID string) ([]devices.Device, error) {
	currentUserID = strings.TrimSpace(currentUserID)
	if currentUserID == "" {
		return []devices.Device{}, nil
	}
	items, err := s.devices.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]devices.Device, 0)
	for _, item := range items {
		if item.AssignedUserID == currentUserID {
			result = append(result, item)
		}
	}
	return result, nil
}

func (s *Service) Licenses(ctx context.Context, currentUserID string) ([]AssignedLicense, error) {
	currentUserID = strings.TrimSpace(currentUserID)
	if currentUserID == "" {
		return []AssignedLicense{}, nil
	}
	deviceItems, err := s.Devices(ctx, currentUserID)
	if err != nil {
		return nil, err
	}
	assignmentItems, err := s.assignments.List(ctx)
	if err != nil {
		return nil, err
	}
	licenseItems, err := s.licenses.List(ctx)
	if err != nil {
		return nil, err
	}

	ownedDevices := make(map[string]devices.Device, len(deviceItems))
	for _, item := range deviceItems {
		ownedDevices[item.ID] = item
	}
	licensesByID := make(map[string]licenses.License, len(licenseItems))
	for _, item := range licenseItems {
		licensesByID[item.ID] = item
	}

	result := make([]AssignedLicense, 0)
	for _, item := range assignmentItems {
		if item.Status != assignments.StatusActive {
			continue
		}
		source := ""
		device := devices.Device{}
		switch {
		case item.UserID == currentUserID:
			source = SourceUser
		case item.DeviceID != "":
			var owned bool
			device, owned = ownedDevices[item.DeviceID]
			if owned {
				source = SourceDevice
			}
		}
		if source == "" {
			continue
		}

		license := licensesByID[item.LicenseID]
		lifecycleStatus := license.LifecycleStatus
		if lifecycleStatus == "" {
			lifecycleStatus = calculateLifecycleStatus(s.now(), license.StartsAt, license.ExpiresAt)
		}
		result = append(result, AssignedLicense{
			AssignmentID:      item.ID,
			LicenseID:         item.LicenseID,
			LicenseName:       item.LicenseName,
			SoftwareProductID: license.SoftwareProductID,
			LicenseType:       license.LicenseType,
			AssignmentSource:  source,
			DeviceID:          device.ID,
			DeviceAssetCode:   device.AssetCode,
			AssignedAt:        item.AssignedAt,
			ExpiresAt:         license.ExpiresAt,
			LifecycleStatus:   lifecycleStatus,
			Notes:             item.Notes,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if !result[i].AssignedAt.Equal(result[j].AssignedAt) {
			return result[i].AssignedAt.After(result[j].AssignedAt)
		}
		return result[i].AssignmentID < result[j].AssignmentID
	})
	return result, nil
}

func calculateLifecycleStatus(now time.Time, startsAt, expiresAt string) string {
	today := now.Format("2006-01-02")
	switch {
	case startsAt != "" && startsAt > today:
		return "upcoming"
	case expiresAt != "" && expiresAt < today:
		return "expired"
	default:
		return "active"
	}
}
