package licenses

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"license-manager/backend/internal/modules/software"
	"license-manager/backend/internal/platform/securevalue"
)

func TestCreateEncryptsLicenseKeyAndCalculatesSeats(t *testing.T) {
	service, repository, cipher, product := newLicenseTestService(t)

	created, err := service.Create(context.Background(), validInput(product.ID))
	if err != nil {
		t.Fatalf("create license: %v", err)
	}
	if created.KeyHint != "****7890" {
		t.Fatalf("unexpected key hint %q", created.KeyHint)
	}
	if created.UsedSeats != 0 || created.AvailableSeats != 25 {
		t.Fatalf("unexpected seat counts: used=%d available=%d", created.UsedSeats, created.AvailableSeats)
	}
	if bytes.Contains(created.EncryptedKey, []byte("SECRET-LICENSE-KEY-1234567890")) {
		t.Fatal("encrypted value contains the plaintext key")
	}
	plaintext, err := cipher.Decrypt(created.EncryptedKey)
	if err != nil || plaintext != "SECRET-LICENSE-KEY-1234567890" {
		t.Fatalf("encrypted key did not round-trip: plaintext=%q error=%v", plaintext, err)
	}

	stored, err := repository.List(context.Background())
	if err != nil || len(stored) != 1 {
		t.Fatalf("expected one stored license, got %d (error: %v)", len(stored), err)
	}
}

func TestSubscriptionRequiresExpirationDate(t *testing.T) {
	service, _, _, product := newLicenseTestService(t)
	input := validInput(product.ID)
	input.ExpiresAt = ""

	_, err := service.Create(context.Background(), input)
	if !errors.Is(err, ErrExpirationRequired) {
		t.Fatalf("expected ErrExpirationRequired, got %v", err)
	}
}

func TestCreateRequiresExistingSoftwareProduct(t *testing.T) {
	service, _, _, _ := newLicenseTestService(t)

	_, err := service.Create(context.Background(), validInput("missing-product"))
	if !errors.Is(err, ErrSoftwareNotFound) {
		t.Fatalf("expected ErrSoftwareNotFound, got %v", err)
	}
}

func TestUpdateWithoutNewKeyPreservesEncryptedKey(t *testing.T) {
	service, _, cipher, product := newLicenseTestService(t)
	ctx := context.Background()
	created, err := service.Create(ctx, validInput(product.ID))
	if err != nil {
		t.Fatalf("create license: %v", err)
	}
	input := validInput(product.ID)
	input.LicenseKey = ""
	input.SeatCount = 50

	updated, err := service.Update(ctx, created.ID, input)
	if err != nil {
		t.Fatalf("update license: %v", err)
	}
	plaintext, err := cipher.Decrypt(updated.EncryptedKey)
	if err != nil || plaintext != "SECRET-LICENSE-KEY-1234567890" {
		t.Fatalf("license key was not preserved: plaintext=%q error=%v", plaintext, err)
	}
	if updated.AvailableSeats != 50 {
		t.Fatalf("expected 50 available seats, got %d", updated.AvailableSeats)
	}
}

func TestListMarksExpiredLicense(t *testing.T) {
	service, _, _, product := newLicenseTestService(t)
	service.now = func() time.Time { return time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC) }
	input := validInput(product.ID)
	input.StartsAt = "2025-01-01"
	input.ExpiresAt = "2026-08-16"
	if _, err := service.Create(context.Background(), input); err != nil {
		t.Fatalf("create expired license: %v", err)
	}

	items, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("list licenses: %v", err)
	}
	if items[0].LifecycleStatus != "expired" {
		t.Fatalf("expected expired status, got %q", items[0].LifecycleStatus)
	}
}

func newLicenseTestService(t *testing.T) (*Service, *MemoryRepository, *securevalue.Cipher, software.Product) {
	t.Helper()
	softwareRepository := software.NewMemoryRepository()
	softwareService := software.NewService(softwareRepository)
	product, err := softwareService.Create(context.Background(), software.Input{
		Name: "Adobe Photoshop", Publisher: "Adobe", Version: "2026",
	})
	if err != nil {
		t.Fatalf("create software product: %v", err)
	}
	cipher, err := securevalue.NewCipher("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")
	if err != nil {
		t.Fatalf("create cipher: %v", err)
	}
	repository := NewMemoryRepository()
	return NewService(repository, softwareRepository, cipher), repository, cipher, product
}

func validInput(productID string) Input {
	return Input{
		SoftwareProductID: productID,
		Name:              "Photoshop Teams",
		LicenseType:       TypeSubscription,
		AssignmentType:    AssignmentUser,
		SeatCount:         25,
		LicenseKey:        "SECRET-LICENSE-KEY-1234567890",
		Vendor:            "Adobe",
		PurchasedAt:       "2026-01-10",
		StartsAt:          "2026-01-10",
		ExpiresAt:         "2099-01-09",
		Cost:              1200,
		Currency:          "USD",
	}
}
