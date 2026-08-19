package licenses

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"license-manager/backend/internal/modules/software"
	"license-manager/backend/internal/platform/id"
	"license-manager/backend/internal/platform/securevalue"
)

type Input struct {
	SoftwareProductID string
	Name              string
	LicenseType       string
	AssignmentType    string
	SeatCount         int
	LicenseKey        string
	Vendor            string
	PurchasedAt       string
	StartsAt          string
	ExpiresAt         string
	Cost              float64
	Currency          string
	Notes             string
}

type Service struct {
	repository Repository
	software   SoftwareFinder
	cipher     *securevalue.Cipher
	now        func() time.Time
}

func NewService(repository Repository, softwareFinder SoftwareFinder, cipher *securevalue.Cipher) *Service {
	return &Service{repository: repository, software: softwareFinder, cipher: cipher, now: time.Now}
}

func (s *Service) List(ctx context.Context) ([]License, error) {
	items, err := s.repository.List(ctx)
	if err != nil {
		return nil, err
	}
	for index := range items {
		s.decorate(&items[index])
	}
	return items, nil
}

func (s *Service) Create(ctx context.Context, input Input) (License, error) {
	input = normalizeInput(input)
	if err := validateInput(input); err != nil {
		return License{}, err
	}
	if _, err := s.software.FindByID(ctx, input.SoftwareProductID); errors.Is(err, software.ErrNotFound) {
		return License{}, ErrSoftwareNotFound
	} else if err != nil {
		return License{}, err
	}

	licenseID, err := id.NewUUID()
	if err != nil {
		return License{}, fmt.Errorf("generate license id: %w", err)
	}
	now := s.now().UTC()
	item := License{
		ID:                licenseID,
		SoftwareProductID: input.SoftwareProductID,
		Name:              input.Name,
		LicenseType:       input.LicenseType,
		AssignmentType:    input.AssignmentType,
		SeatCount:         input.SeatCount,
		AvailableSeats:    input.SeatCount,
		Vendor:            input.Vendor,
		PurchasedAt:       input.PurchasedAt,
		StartsAt:          input.StartsAt,
		ExpiresAt:         input.ExpiresAt,
		Cost:              input.Cost,
		Currency:          input.Currency,
		Notes:             input.Notes,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := s.protectKey(&item, input.LicenseKey); err != nil {
		return License{}, err
	}
	created, err := s.repository.Create(ctx, item)
	if err != nil {
		return License{}, err
	}
	s.decorate(&created)
	return created, nil
}

func (s *Service) Update(ctx context.Context, licenseID string, input Input) (License, error) {
	input = normalizeInput(input)
	if licenseID == "" {
		return License{}, ErrInvalidData
	}
	if err := validateInput(input); err != nil {
		return License{}, err
	}
	existing, err := s.repository.FindByID(ctx, licenseID)
	if err != nil {
		return License{}, err
	}
	if existing.ArchivedAt != nil {
		return License{}, ErrArchived
	}
	if _, err := s.software.FindByID(ctx, input.SoftwareProductID); errors.Is(err, software.ErrNotFound) {
		return License{}, ErrSoftwareNotFound
	} else if err != nil {
		return License{}, err
	}

	item := License{
		ID:                licenseID,
		SoftwareProductID: input.SoftwareProductID,
		Name:              input.Name,
		LicenseType:       input.LicenseType,
		AssignmentType:    input.AssignmentType,
		SeatCount:         input.SeatCount,
		Vendor:            input.Vendor,
		PurchasedAt:       input.PurchasedAt,
		StartsAt:          input.StartsAt,
		ExpiresAt:         input.ExpiresAt,
		Cost:              input.Cost,
		Currency:          input.Currency,
		Notes:             input.Notes,
		UpdatedAt:         s.now().UTC(),
	}
	if err := s.protectKey(&item, input.LicenseKey); err != nil {
		return License{}, err
	}
	updated, err := s.repository.Update(ctx, item)
	if err != nil {
		return License{}, err
	}
	s.decorate(&updated)
	return updated, nil
}

func (s *Service) Archive(ctx context.Context, licenseID string) (License, error) {
	licenseID = strings.TrimSpace(licenseID)
	if licenseID == "" {
		return License{}, ErrInvalidData
	}
	item, err := s.repository.FindByID(ctx, licenseID)
	if err != nil {
		return License{}, err
	}
	if item.ArchivedAt != nil {
		return License{}, ErrAlreadyArchived
	}
	if item.UsedSeats > 0 {
		return License{}, ErrActiveAssignments
	}
	archived, err := s.repository.Archive(ctx, licenseID, s.now().UTC())
	if err != nil {
		return License{}, err
	}
	s.decorate(&archived)
	return archived, nil
}

func (s *Service) RevealKey(ctx context.Context, licenseID string) (string, error) {
	licenseID = strings.TrimSpace(licenseID)
	if licenseID == "" {
		return "", ErrInvalidData
	}
	item, err := s.repository.FindByID(ctx, licenseID)
	if err != nil {
		return "", err
	}
	if len(item.EncryptedKey) == 0 {
		return "", ErrKeyNotSet
	}
	plaintext, err := s.cipher.Decrypt(item.EncryptedKey)
	if err != nil {
		return "", fmt.Errorf("decrypt license key: %w", err)
	}
	return plaintext, nil
}

func (s *Service) protectKey(item *License, plaintext string) error {
	if plaintext == "" {
		return nil
	}
	encrypted, err := s.cipher.Encrypt(plaintext)
	if err != nil {
		return fmt.Errorf("encrypt license key: %w", err)
	}
	item.EncryptedKey = encrypted
	item.KeyHint = keyHint(plaintext)
	return nil
}

func (s *Service) decorate(item *License) {
	item.AvailableSeats = item.SeatCount - item.UsedSeats
	today := s.now().UTC().Format("2006-01-02")
	switch {
	case item.ArchivedAt != nil:
		item.LifecycleStatus = "archived"
	case item.StartsAt != "" && item.StartsAt > today:
		item.LifecycleStatus = "upcoming"
	case item.ExpiresAt != "" && item.ExpiresAt < today:
		item.LifecycleStatus = "expired"
	default:
		item.LifecycleStatus = "active"
	}
}

func normalizeInput(input Input) Input {
	input.SoftwareProductID = strings.TrimSpace(input.SoftwareProductID)
	input.Name = strings.TrimSpace(input.Name)
	input.LicenseType = strings.ToLower(strings.TrimSpace(input.LicenseType))
	input.AssignmentType = strings.ToLower(strings.TrimSpace(input.AssignmentType))
	input.LicenseKey = strings.TrimSpace(input.LicenseKey)
	input.Vendor = strings.TrimSpace(input.Vendor)
	input.PurchasedAt = strings.TrimSpace(input.PurchasedAt)
	input.StartsAt = strings.TrimSpace(input.StartsAt)
	input.ExpiresAt = strings.TrimSpace(input.ExpiresAt)
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	input.Notes = strings.TrimSpace(input.Notes)
	return input
}

func validateInput(input Input) error {
	if input.SoftwareProductID == "" || input.Name == "" {
		return ErrInvalidData
	}
	if input.LicenseType != TypeSubscription && input.LicenseType != TypePerpetual {
		return ErrInvalidType
	}
	if input.AssignmentType != AssignmentUser && input.AssignmentType != AssignmentDevice && input.AssignmentType != AssignmentMixed {
		return ErrInvalidAssignment
	}
	if input.SeatCount <= 0 {
		return ErrInvalidSeatCount
	}
	for _, value := range []string{input.PurchasedAt, input.StartsAt, input.ExpiresAt} {
		if value != "" {
			if _, err := time.Parse("2006-01-02", value); err != nil {
				return ErrInvalidDate
			}
		}
	}
	if input.LicenseType == TypeSubscription && input.ExpiresAt == "" {
		return ErrExpirationRequired
	}
	if input.StartsAt != "" && input.ExpiresAt != "" && input.ExpiresAt < input.StartsAt {
		return ErrInvalidDateRange
	}
	if input.Cost < 0 || (input.Cost > 0 && len(input.Currency) != 3) || (input.Currency != "" && len(input.Currency) != 3) {
		return ErrInvalidCost
	}
	return nil
}

func keyHint(plaintext string) string {
	runes := []rune(plaintext)
	if len(runes) > 4 {
		runes = runes[len(runes)-4:]
	}
	return "****" + string(runes)
}
