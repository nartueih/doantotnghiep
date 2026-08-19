package assignments

import (
	"context"
	"errors"
	"strings"
	"time"

	"license-manager/backend/internal/modules/auth"
	"license-manager/backend/internal/modules/devices"
	"license-manager/backend/internal/modules/licenses"
	"license-manager/backend/internal/platform/id"
)

type CreateInput struct {
	LicenseID string
	UserID    string
	DeviceID  string
	Notes     string
}

type Service struct {
	repository Repository
	licenses   LicenseFinder
	users      UserFinder
	devices    DeviceFinder
	now        func() time.Time
}

func NewService(repository Repository, licenseFinder LicenseFinder, userFinder UserFinder, deviceFinder DeviceFinder) *Service {
	return &Service{
		repository: repository,
		licenses:   licenseFinder,
		users:      userFinder,
		devices:    deviceFinder,
		now:        time.Now,
	}
}

func (s *Service) List(ctx context.Context) ([]Assignment, error) {
	return s.repository.List(ctx)
}

func (s *Service) Create(ctx context.Context, actorID string, input CreateInput) (Assignment, error) {
	input.LicenseID = strings.TrimSpace(input.LicenseID)
	input.UserID = strings.TrimSpace(input.UserID)
	input.DeviceID = strings.TrimSpace(input.DeviceID)
	input.Notes = strings.TrimSpace(input.Notes)
	if input.LicenseID == "" || (input.UserID == "") == (input.DeviceID == "") {
		return Assignment{}, ErrInvalidTarget
	}

	license, err := s.licenses.FindByID(ctx, input.LicenseID)
	if errors.Is(err, licenses.ErrNotFound) {
		return Assignment{}, ErrLicenseNotFound
	}
	if err != nil {
		return Assignment{}, err
	}
	today := s.now().UTC().Format("2006-01-02")
	if license.ArchivedAt != nil || (license.StartsAt != "" && license.StartsAt > today) || (license.ExpiresAt != "" && license.ExpiresAt < today) {
		return Assignment{}, ErrLicenseInactive
	}
	if license.UsedSeats >= license.SeatCount {
		return Assignment{}, ErrNoAvailableSeats
	}

	item := Assignment{
		LicenseID:   license.ID,
		LicenseName: license.Name,
		UserID:      input.UserID,
		DeviceID:    input.DeviceID,
		AssignedBy:  actorID,
		AssignedAt:  s.now().UTC(),
		Status:      StatusActive,
		Notes:       input.Notes,
	}
	if actor, actorErr := s.users.FindByID(ctx, actorID); actorErr == nil {
		item.AssignedByName = actor.FullName
	}

	if input.UserID != "" {
		if license.AssignmentType == licenses.AssignmentDevice {
			return Assignment{}, ErrAssignmentType
		}
		user, err := s.users.FindByID(ctx, input.UserID)
		if errors.Is(err, auth.ErrUserNotFound) || (err == nil && user.Status != auth.StatusActive) {
			return Assignment{}, ErrTargetUnavailable
		}
		if err != nil {
			return Assignment{}, err
		}
		item.TargetName = user.FullName
	} else {
		if license.AssignmentType == licenses.AssignmentUser {
			return Assignment{}, ErrAssignmentType
		}
		device, err := s.devices.FindByID(ctx, input.DeviceID)
		if errors.Is(err, devices.ErrNotFound) ||
			(err == nil && (device.Status == devices.StatusRetired || device.Status == devices.StatusLost)) {
			return Assignment{}, ErrTargetUnavailable
		}
		if err != nil {
			return Assignment{}, err
		}
		item.TargetName = device.AssetCode
	}

	assignmentID, err := id.NewUUID()
	if err != nil {
		return Assignment{}, err
	}
	item.ID = assignmentID
	return s.repository.Create(ctx, item)
}

func (s *Service) Revoke(ctx context.Context, actorID, assignmentID string) (Assignment, error) {
	actorName := ""
	if actor, err := s.users.FindByID(ctx, actorID); err == nil {
		actorName = actor.FullName
	}
	return s.repository.Revoke(ctx, assignmentID, actorID, actorName, s.now().UTC())
}
