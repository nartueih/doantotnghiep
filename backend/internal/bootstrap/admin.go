package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"license-manager/backend/internal/modules/auth"
)

const AdminID = "00000000-0000-0000-0000-000000000001"

var (
	ErrInvalidAdminInput = errors.New("Admin email, password, full name and employee code are required")
	ErrAdminConflict     = errors.New("configured Admin email belongs to an incompatible account")
)

type AdminStore interface {
	FindByEmail(context.Context, string) (auth.User, error)
	CreateUser(context.Context, auth.User) (auth.User, error)
}

type AdminInput struct {
	Email        string
	Password     string
	FullName     string
	EmployeeCode string
}

type Result struct {
	User    auth.User
	Created bool
}

func EnsureAdmin(
	ctx context.Context,
	store AdminStore,
	hasher auth.PasswordHasher,
	input AdminInput,
) (Result, error) {
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.Password = strings.TrimSpace(input.Password)
	input.FullName = strings.TrimSpace(input.FullName)
	input.EmployeeCode = strings.TrimSpace(input.EmployeeCode)
	if input.Email == "" || input.Password == "" || input.FullName == "" || input.EmployeeCode == "" {
		return Result{}, ErrInvalidAdminInput
	}

	existing, err := store.FindByEmail(ctx, input.Email)
	if err == nil {
		if existing.Role != auth.RoleAdmin || existing.Status != auth.StatusActive || existing.EmployeeCode != input.EmployeeCode {
			return Result{}, ErrAdminConflict
		}
		return Result{User: existing, Created: false}, nil
	}
	if !errors.Is(err, auth.ErrUserNotFound) {
		return Result{}, fmt.Errorf("find Admin account: %w", err)
	}

	passwordHash, err := hasher.Hash(input.Password)
	if err != nil {
		return Result{}, fmt.Errorf("hash Admin password: %w", err)
	}
	created, err := store.CreateUser(ctx, auth.User{
		ID:           AdminID,
		Email:        input.Email,
		PasswordHash: passwordHash,
		FullName:     input.FullName,
		EmployeeCode: input.EmployeeCode,
		Role:         auth.RoleAdmin,
		Status:       auth.StatusActive,
		CreatedAt:    time.Now().UTC(),
	})
	if err != nil {
		return Result{}, fmt.Errorf("create Admin account: %w", err)
	}
	return Result{User: created, Created: true}, nil
}
