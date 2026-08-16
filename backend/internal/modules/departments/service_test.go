package departments

import (
	"context"
	"errors"
	"testing"
)

func TestCreateNormalizesDepartment(t *testing.T) {
	service := NewService(NewMemoryRepository())

	created, err := service.Create(context.Background(), Input{
		Name: " Information Technology ",
		Code: " it ",
	})
	if err != nil {
		t.Fatalf("create department: %v", err)
	}
	if created.Name != "Information Technology" || created.Code != "IT" {
		t.Fatalf("department was not normalized: %#v", created)
	}
}

func TestCreateRejectsDuplicateNameAndCodeCaseInsensitively(t *testing.T) {
	service := NewService(NewMemoryRepository())
	if _, err := service.Create(context.Background(), Input{Name: "Finance", Code: "FIN"}); err != nil {
		t.Fatalf("create initial department: %v", err)
	}

	if _, err := service.Create(context.Background(), Input{Name: "finance", Code: "OTHER"}); !errors.Is(err, ErrNameExists) {
		t.Fatalf("expected ErrNameExists, got %v", err)
	}
	if _, err := service.Create(context.Background(), Input{Name: "Other", Code: "fin"}); !errors.Is(err, ErrCodeExists) {
		t.Fatalf("expected ErrCodeExists, got %v", err)
	}
}

func TestUpdateMissingDepartmentReturnsNotFound(t *testing.T) {
	service := NewService(NewMemoryRepository())

	_, err := service.Update(context.Background(), "missing", Input{Name: "Finance", Code: "FIN"})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
