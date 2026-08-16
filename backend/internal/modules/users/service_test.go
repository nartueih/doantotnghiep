package users

import (
	"context"
	"errors"
	"testing"
	"time"

	"license-manager/backend/internal/modules/auth"
	"license-manager/backend/internal/modules/departments"
)

func TestCreateUserNormalizesAndHashesData(t *testing.T) {
	service, repository, hasher := newUsersTestService(t)

	created, err := service.Create(context.Background(), CreateInput{
		Email:        " NEW.USER@EXAMPLE.COM ",
		Password:     "SecurePass123",
		FullName:     " New User ",
		EmployeeCode: " emp-002 ",
		Role:         auth.RoleEmployee,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if created.Email != "new.user@example.com" || created.EmployeeCode != "EMP-002" {
		t.Fatalf("user data was not normalized: %#v", created)
	}
	if created.PasswordHash == "SecurePass123" || hasher.Compare(created.PasswordHash, "SecurePass123") != nil {
		t.Fatal("password was not hashed correctly")
	}

	items, err := repository.ListUsers(context.Background())
	if err != nil || len(items) != 2 {
		t.Fatalf("expected two users, got %d (error: %v)", len(items), err)
	}
}

func TestCreateUserRejectsWeakPassword(t *testing.T) {
	service, _, _ := newUsersTestService(t)

	_, err := service.Create(context.Background(), CreateInput{
		Email:        "new.user@example.com",
		Password:     "weak",
		FullName:     "New User",
		EmployeeCode: "EMP-002",
		Role:         auth.RoleEmployee,
	})
	if !errors.Is(err, ErrWeakPassword) {
		t.Fatalf("expected ErrWeakPassword, got %v", err)
	}
}

func TestCreateUserRejectsDuplicateEmail(t *testing.T) {
	service, _, _ := newUsersTestService(t)

	_, err := service.Create(context.Background(), CreateInput{
		Email:        "admin@example.com",
		Password:     "SecurePass123",
		FullName:     "Other Admin",
		EmployeeCode: "EMP-002",
		Role:         auth.RoleAdmin,
	})
	if !errors.Is(err, auth.ErrEmailAlreadyExists) {
		t.Fatalf("expected ErrEmailAlreadyExists, got %v", err)
	}
}

func TestAdministratorCannotLockOwnAccount(t *testing.T) {
	service, _, _ := newUsersTestService(t)

	_, err := service.UpdateStatus(
		context.Background(),
		"00000000-0000-0000-0000-000000000001",
		"00000000-0000-0000-0000-000000000001",
		auth.StatusLocked,
	)
	if !errors.Is(err, ErrCannotLockSelf) {
		t.Fatalf("expected ErrCannotLockSelf, got %v", err)
	}
}

func TestCreateUserWithDepartment(t *testing.T) {
	_, repository, hasher := newUsersTestService(t)
	departmentRepository := departments.NewMemoryRepository()
	department := departments.Department{
		ID:   "00000000-0000-0000-0000-000000000010",
		Name: "Information Technology",
		Code: "IT",
	}
	if _, err := departmentRepository.Create(context.Background(), department); err != nil {
		t.Fatalf("create department: %v", err)
	}
	service := NewService(repository, hasher, departmentRepository)

	created, err := service.Create(context.Background(), CreateInput{
		Email:        "employee@example.com",
		Password:     "SecurePass123",
		FullName:     "Employee User",
		EmployeeCode: "EMP-002",
		DepartmentID: department.ID,
		Role:         auth.RoleEmployee,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if created.DepartmentID != department.ID || created.DepartmentName != department.Name {
		t.Fatalf("unexpected department data: %#v", created)
	}
}

func TestCreateUserRejectsMissingDepartment(t *testing.T) {
	_, repository, hasher := newUsersTestService(t)
	service := NewService(repository, hasher, departments.NewMemoryRepository())

	_, err := service.Create(context.Background(), CreateInput{
		Email:        "employee@example.com",
		Password:     "SecurePass123",
		FullName:     "Employee User",
		EmployeeCode: "EMP-002",
		DepartmentID: "missing",
		Role:         auth.RoleEmployee,
	})
	if !errors.Is(err, ErrDepartmentNotFound) {
		t.Fatalf("expected ErrDepartmentNotFound, got %v", err)
	}
}

func newUsersTestService(t *testing.T) (*Service, *auth.MemoryRepository, auth.PasswordHasher) {
	t.Helper()
	hasher := auth.NewPasswordHasher(4)
	passwordHash, err := hasher.Hash("AdminPassword123")
	if err != nil {
		t.Fatalf("hash admin password: %v", err)
	}
	repository := auth.NewMemoryRepository([]auth.User{{
		ID:           "00000000-0000-0000-0000-000000000001",
		Email:        "admin@example.com",
		PasswordHash: passwordHash,
		FullName:     "Admin User",
		EmployeeCode: "ADMIN-001",
		Role:         auth.RoleAdmin,
		Status:       auth.StatusActive,
		CreatedAt:    time.Now(),
	}})
	return NewService(repository, hasher), repository, hasher
}
