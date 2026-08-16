package auth

import (
	"context"
	"errors"
)

var (
	ErrUserNotFound        = errors.New("user not found")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrAccountLocked       = errors.New("account is locked")
	ErrInvalidToken        = errors.New("invalid token")
	ErrInvalidRefreshToken = errors.New("invalid or already used refresh token")
	ErrEmailAlreadyExists  = errors.New("email already exists")
	ErrCodeAlreadyExists   = errors.New("employee code already exists")
)

type Repository interface {
	FindByEmail(context.Context, string) (User, error)
	FindByID(context.Context, string) (User, error)
	SaveRefreshSession(context.Context, RefreshSession) error
	RotateRefreshSession(context.Context, string, RefreshSession) error
	RevokeRefreshSession(context.Context, string) error
}
