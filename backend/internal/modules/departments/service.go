package departments

import (
	"context"
	"fmt"
	"strings"
	"time"

	"license-manager/backend/internal/platform/id"
)

type Input struct {
	Name string
	Code string
}

type Service struct {
	repository Repository
	now        func() time.Time
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository, now: time.Now}
}

func (s *Service) List(ctx context.Context) ([]Department, error) {
	return s.repository.List(ctx)
}

func (s *Service) Create(ctx context.Context, input Input) (Department, error) {
	input = normalize(input)
	if input.Name == "" || input.Code == "" {
		return Department{}, ErrInvalidData
	}
	departmentID, err := id.NewUUID()
	if err != nil {
		return Department{}, fmt.Errorf("generate department id: %w", err)
	}
	now := s.now().UTC()
	return s.repository.Create(ctx, Department{
		ID: departmentID, Name: input.Name, Code: input.Code, CreatedAt: now, UpdatedAt: now,
	})
}

func (s *Service) Update(ctx context.Context, departmentID string, input Input) (Department, error) {
	input = normalize(input)
	if departmentID == "" || input.Name == "" || input.Code == "" {
		return Department{}, ErrInvalidData
	}
	return s.repository.Update(ctx, Department{
		ID: departmentID, Name: input.Name, Code: input.Code, UpdatedAt: s.now().UTC(),
	})
}

func normalize(input Input) Input {
	input.Name = strings.TrimSpace(input.Name)
	input.Code = strings.ToUpper(strings.TrimSpace(input.Code))
	return input
}
