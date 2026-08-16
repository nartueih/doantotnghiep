package users

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"license-manager/backend/internal/modules/auth"
	"license-manager/backend/internal/modules/departments"
	"license-manager/backend/internal/platform/id"
)

var (
	ErrInvalidRole        = errors.New("invalid user role")
	ErrInvalidStatus      = errors.New("invalid user status")
	ErrWeakPassword       = errors.New("password does not meet complexity requirements")
	ErrIncompleteUserData = errors.New("user data is incomplete")
	ErrCannotLockSelf     = errors.New("an administrator cannot lock their own account")
	ErrDepartmentNotFound = errors.New("department not found")
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
	DepartmentID string
	Role         string
}

type DepartmentFinder interface {
	FindByID(context.Context, string) (departments.Department, error)
}

type Service struct {
	repository       Repository
	hasher           auth.PasswordHasher
	departmentFinder DepartmentFinder
}

func NewService(repository Repository, hasher auth.PasswordHasher, departmentFinder ...DepartmentFinder) *Service {
	service := &Service{repository: repository, hasher: hasher}
	if len(departmentFinder) > 0 {
		service.departmentFinder = departmentFinder[0]
	}
	return service
}

func (s *Service) List(ctx context.Context) ([]auth.User, error) {
	return s.repository.ListUsers(ctx)
}

func (s *Service) Create(ctx context.Context, input CreateInput) (auth.User, error) {
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.FullName = strings.TrimSpace(input.FullName)
	input.EmployeeCode = strings.ToUpper(strings.TrimSpace(input.EmployeeCode))
	input.DepartmentID = strings.TrimSpace(input.DepartmentID)
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

	departmentName := ""
	if input.DepartmentID != "" {
		if s.departmentFinder == nil {
			return auth.User{}, ErrDepartmentNotFound
		}
		department, err := s.departmentFinder.FindByID(ctx, input.DepartmentID)
		if errors.Is(err, departments.ErrNotFound) {
			return auth.User{}, ErrDepartmentNotFound
		}
		if err != nil {
			return auth.User{}, fmt.Errorf("find department: %w", err)
		}
		departmentName = department.Name
	}

	passwordHash, err := s.hasher.Hash(input.Password)
	if err != nil {
		return auth.User{}, fmt.Errorf("hash password: %w", err)
	}
	userID, err := id.NewUUID()
	if err != nil {
		return auth.User{}, fmt.Errorf("generate user id: %w", err)
	}

	return s.repository.CreateUser(ctx, auth.User{
		ID:             userID,
		Email:          input.Email,
		PasswordHash:   passwordHash,
		FullName:       input.FullName,
		EmployeeCode:   input.EmployeeCode,
		DepartmentID:   input.DepartmentID,
		DepartmentName: departmentName,
		Role:           input.Role,
		Status:         auth.StatusActive,
		CreatedAt:      time.Now().UTC(),
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
