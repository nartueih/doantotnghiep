package devices

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"license-manager/backend/internal/modules/auth"
	"license-manager/backend/internal/platform/id"
)

type Input struct {
	AssetCode         string
	SerialNumber      string
	Name              string
	DeviceType        string
	Manufacturer      string
	Model             string
	PurchasedAt       string
	WarrantyExpiresAt string
}

type Service struct {
	repository Repository
	users      UserFinder
	now        func() time.Time
}

func NewService(repository Repository, userFinder UserFinder) *Service {
	return &Service{repository: repository, users: userFinder, now: time.Now}
}

func (s *Service) List(ctx context.Context) ([]Device, error) {
	return s.repository.List(ctx)
}

func (s *Service) Create(ctx context.Context, input Input) (Device, error) {
	input = normalize(input)
	if err := validate(input); err != nil {
		return Device{}, err
	}
	deviceID, err := id.NewUUID()
	if err != nil {
		return Device{}, fmt.Errorf("generate device id: %w", err)
	}
	now := s.now().UTC()
	return s.repository.Create(ctx, Device{
		ID:                deviceID,
		AssetCode:         input.AssetCode,
		SerialNumber:      input.SerialNumber,
		Name:              input.Name,
		DeviceType:        input.DeviceType,
		Manufacturer:      input.Manufacturer,
		Model:             input.Model,
		Status:            StatusAvailable,
		PurchasedAt:       input.PurchasedAt,
		WarrantyExpiresAt: input.WarrantyExpiresAt,
		CreatedAt:         now,
		UpdatedAt:         now,
	})
}

func (s *Service) Update(ctx context.Context, deviceID string, input Input) (Device, error) {
	input = normalize(input)
	if deviceID == "" {
		return Device{}, ErrInvalidData
	}
	if err := validate(input); err != nil {
		return Device{}, err
	}
	return s.repository.Update(ctx, Device{
		ID:                deviceID,
		AssetCode:         input.AssetCode,
		SerialNumber:      input.SerialNumber,
		Name:              input.Name,
		DeviceType:        input.DeviceType,
		Manufacturer:      input.Manufacturer,
		Model:             input.Model,
		PurchasedAt:       input.PurchasedAt,
		WarrantyExpiresAt: input.WarrantyExpiresAt,
		UpdatedAt:         s.now().UTC(),
	})
}

func (s *Service) ChangeStatus(ctx context.Context, deviceID, status string) (Device, error) {
	status = strings.ToLower(strings.TrimSpace(status))
	if status != StatusAvailable && status != StatusMaintenance && status != StatusRetired && status != StatusLost {
		return Device{}, ErrInvalidStatus
	}
	device, err := s.repository.FindByID(ctx, deviceID)
	if err != nil {
		return Device{}, err
	}
	if device.AssignedUserID != "" {
		return Device{}, ErrDeviceAssigned
	}
	return s.repository.UpdateStatus(ctx, deviceID, status)
}

func (s *Service) Assign(ctx context.Context, deviceID, userID string) (Device, error) {
	userID = strings.TrimSpace(userID)
	device, err := s.repository.FindByID(ctx, deviceID)
	if err != nil {
		return Device{}, err
	}

	if userID == "" {
		if device.AssignedUserID == "" {
			return Device{}, ErrDeviceUnavailable
		}
		return s.repository.Assign(ctx, deviceID, "", "")
	}
	if device.Status != StatusAvailable || device.AssignedUserID != "" {
		return Device{}, ErrDeviceUnavailable
	}
	user, err := s.users.FindByID(ctx, userID)
	if errors.Is(err, auth.ErrUserNotFound) || (err == nil && user.Status != auth.StatusActive) {
		return Device{}, ErrUserUnavailable
	}
	if err != nil {
		return Device{}, err
	}
	return s.repository.Assign(ctx, deviceID, userID, user.FullName)
}

func normalize(input Input) Input {
	input.AssetCode = strings.ToUpper(strings.TrimSpace(input.AssetCode))
	input.SerialNumber = strings.TrimSpace(input.SerialNumber)
	input.Name = strings.TrimSpace(input.Name)
	input.DeviceType = strings.ToLower(strings.TrimSpace(input.DeviceType))
	input.Manufacturer = strings.TrimSpace(input.Manufacturer)
	input.Model = strings.TrimSpace(input.Model)
	input.PurchasedAt = strings.TrimSpace(input.PurchasedAt)
	input.WarrantyExpiresAt = strings.TrimSpace(input.WarrantyExpiresAt)
	return input
}

func validate(input Input) error {
	if input.AssetCode == "" || input.Name == "" || input.DeviceType == "" {
		return ErrInvalidData
	}
	for _, value := range []string{input.PurchasedAt, input.WarrantyExpiresAt} {
		if value != "" {
			if _, err := time.Parse("2006-01-02", value); err != nil {
				return ErrInvalidDate
			}
		}
	}
	if input.PurchasedAt != "" && input.WarrantyExpiresAt != "" && input.WarrantyExpiresAt < input.PurchasedAt {
		return ErrInvalidDateRange
	}
	return nil
}
