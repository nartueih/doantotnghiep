package software

import (
	"context"
	"errors"
	"testing"
)

func TestCreateNormalizesProduct(t *testing.T) {
	service := NewService(NewMemoryRepository())

	created, err := service.Create(context.Background(), Input{
		Name:        " Adobe Photoshop ",
		Publisher:   " Adobe ",
		Version:     " 2026 ",
		Description: " Image editing software ",
	})
	if err != nil {
		t.Fatalf("create software product: %v", err)
	}
	if created.Name != "Adobe Photoshop" || created.Publisher != "Adobe" || created.Version != "2026" {
		t.Fatalf("product was not normalized: %#v", created)
	}
	if created.ID == "" || created.CreatedAt.IsZero() || created.UpdatedAt.IsZero() {
		t.Fatal("expected generated identity and timestamps")
	}
}

func TestCreateRejectsCaseInsensitiveDuplicate(t *testing.T) {
	service := NewService(NewMemoryRepository())
	ctx := context.Background()

	_, err := service.Create(ctx, Input{Name: "Microsoft 365", Publisher: "Microsoft", Version: "Business"})
	if err != nil {
		t.Fatalf("create first product: %v", err)
	}
	_, err = service.Create(ctx, Input{Name: "microsoft 365", Publisher: "MICROSOFT", Version: "business"})
	if !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}
}

func TestUpdatePreservesCreatedAt(t *testing.T) {
	service := NewService(NewMemoryRepository())
	ctx := context.Background()
	created, err := service.Create(ctx, Input{Name: "Office", Publisher: "Microsoft", Version: "2024"})
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	updated, err := service.Update(ctx, created.ID, Input{
		Name:        "Office LTSC",
		Publisher:   "Microsoft",
		Version:     "2024",
		Description: "Updated description",
	})
	if err != nil {
		t.Fatalf("update product: %v", err)
	}
	if updated.CreatedAt != created.CreatedAt || updated.Name != "Office LTSC" {
		t.Fatalf("unexpected updated product: %#v", updated)
	}
}

func TestUpdateMissingProductReturnsNotFound(t *testing.T) {
	service := NewService(NewMemoryRepository())

	_, err := service.Update(context.Background(), "missing-id", Input{Name: "Office", Publisher: "Microsoft"})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
