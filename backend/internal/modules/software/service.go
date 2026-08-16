package software

import (
	"context"
	"fmt"
	"strings"
	"time"

	"license-manager/backend/internal/platform/id"
)

type Input struct {
	Name        string
	Publisher   string
	Version     string
	Description string
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) List(ctx context.Context) ([]Product, error) {
	return s.repository.List(ctx)
}

func (s *Service) Create(ctx context.Context, input Input) (Product, error) {
	input = normalize(input)
	if input.Name == "" || input.Publisher == "" {
		return Product{}, ErrInvalidData
	}

	productID, err := id.NewUUID()
	if err != nil {
		return Product{}, fmt.Errorf("generate software product id: %w", err)
	}
	now := time.Now().UTC()
	return s.repository.Create(ctx, Product{
		ID:          productID,
		Name:        input.Name,
		Publisher:   input.Publisher,
		Version:     input.Version,
		Description: input.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
}

func (s *Service) Update(ctx context.Context, productID string, input Input) (Product, error) {
	input = normalize(input)
	if productID == "" || input.Name == "" || input.Publisher == "" {
		return Product{}, ErrInvalidData
	}

	return s.repository.Update(ctx, Product{
		ID:          productID,
		Name:        input.Name,
		Publisher:   input.Publisher,
		Version:     input.Version,
		Description: input.Description,
		UpdatedAt:   time.Now().UTC(),
	})
}

func normalize(input Input) Input {
	input.Name = strings.TrimSpace(input.Name)
	input.Publisher = strings.TrimSpace(input.Publisher)
	input.Version = strings.TrimSpace(input.Version)
	input.Description = strings.TrimSpace(input.Description)
	return input
}
