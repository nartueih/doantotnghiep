package users

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"license-manager/backend/internal/modules/auth"
)

var (
	ErrInvalidRole        = errors.New("invalid user role")
	ErrInvalidStatus      = errors.New("invalid user status")
	ErrWeakPassword       = errors.New("password does not meet complexity requirements")
	ErrIncompleteUserData = errors.New("user data is incomplete")
	ErrCannotLockSelf     = errors.New("an administrator cannot lock their own account")
)

type Repository interface {
	ListUsers(context.Context) ([]auth.User, error)
	CreateUser(context.Context, auth.User) (auth.User, error)
	UpdateUserStatus(context.Context, string, string) (auth.User, error)
}

type CreateInput struct {
	Email        string
	Password     string
	FullName     string
	EmployeeCode string
	Role         string
}

type Service struct {
	repository Repository
	hasher     auth.PasswordHasher
}

func NewService(repository Repository, hasher auth.PasswordHasher) *Service {
	return &Service{repository: repository, hasher: hasher}
}

func (s *Service) List(ctx context.Context) ([]auth.User, error) {
	return s.repository.ListUsers(ctx)
}

func (s *Service) Create(ctx context.Context, input CreateInput) (auth.User, error) {
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.FullName = strings.TrimSpace(input.FullName)
	input.EmployeeCode = strings.ToUpper(strings.TrimSpace(input.EmployeeCode))
	input.Role = strings.TrimSpace(input.Role)

	if input.Email == "" || input.FullName == "" || input.EmployeeCode == "" {
		return auth.User{}, ErrIncompleteUserData
	}
	if !validRole(input.Role) {
		return auth.User{}, ErrInvalidRole
	}
	if !strongPassword(input.Password) {
		return auth.User{}, ErrWeakPassword
	}

	passwordHash, err := s.hasher.Hash(input.Password)
	if err != nil {
		return auth.User{}, fmt.Errorf("hash password: %w", err)
	}
	userID, err := randomUUID()
	if err != nil {
		return auth.User{}, fmt.Errorf("generate user id: %w", err)
	}

	return s.repository.CreateUser(ctx, auth.User{
		ID:           userID,
		Email:        input.Email,
		PasswordHash: passwordHash,
		FullName:     input.FullName,
		EmployeeCode: input.EmployeeCode,
		Role:         input.Role,
		Status:       auth.StatusActive,
		CreatedAt:    time.Now().UTC(),
	})
}

func (s *Service) UpdateStatus(ctx context.Context, actorID, userID, status string) (auth.User, error) {
	if status != auth.StatusActive && status != auth.StatusLocked {
		return auth.User{}, ErrInvalidStatus
	}
	if actorID == userID && status == auth.StatusLocked {
		return auth.User{}, ErrCannotLockSelf
	}
	return s.repository.UpdateUserStatus(ctx, userID, status)
}

func validRole(role string) bool {
	return role == auth.RoleAdmin || role == auth.RoleITManager || role == auth.RoleEmployee
}

func strongPassword(password string) bool {
	if len([]rune(password)) < 10 {
		return false
	}
	var upper, lower, digit bool
	for _, character := range password {
		upper = upper || unicode.IsUpper(character)
		lower = lower || unicode.IsLower(character)
		digit = digit || unicode.IsDigit(character)
	}
	return upper && lower && digit
}

func randomUUID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		value[0:4], value[4:6], value[6:8], value[8:10], value[10:16]), nil
}
