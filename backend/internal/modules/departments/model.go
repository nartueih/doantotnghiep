package departments

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound    = errors.New("department not found")
	ErrNameExists  = errors.New("department name already exists")
	ErrCodeExists  = errors.New("department code already exists")
	ErrInvalidData = errors.New("department name and code are required")
)

type Department struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Repository interface {
	List(context.Context) ([]Department, error)
	FindByID(context.Context, string) (Department, error)
	Create(context.Context, Department) (Department, error)
	Update(context.Context, Department) (Department, error)
}
