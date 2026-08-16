package software

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound      = errors.New("software product not found")
	ErrAlreadyExists = errors.New("software product already exists")
	ErrInvalidData   = errors.New("software name and publisher are required")
)

type Product struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Publisher   string    `json:"publisher"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Repository interface {
	List(context.Context) ([]Product, error)
	Create(context.Context, Product) (Product, error)
	Update(context.Context, Product) (Product, error)
}
