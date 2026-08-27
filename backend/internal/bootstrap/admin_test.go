package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"

	"license-manager/backend/internal/modules/auth"
)

func TestEnsureAdminCreatesOnceAndIsIdempotent(t *testing.T) {
	store := newFakeAdminStore()
	input := AdminInput{
		Email: " Admin@Local.Test ", Password: "ChangeMe123!",
		FullName: " Development Admin ", EmployeeCode: " DEV-ADMIN ",
	}

	first, err := EnsureAdmin(t.Context(), store, auth.NewPasswordHasher(4), input)
	if err != nil || !first.Created {
		t.Fatalf("first=%#v err=%v", first, err)
	}
	if first.User.Email != "admin@local.test" || first.User.Role != auth.RoleAdmin || first.User.Status != auth.StatusActive {
		t.Fatalf("unexpected Admin: %#v", first.User)
	}
	if err := (auth.NewPasswordHasher(4)).Compare(first.User.PasswordHash, input.Password); err != nil {
		t.Fatalf("stored password hash does not match: %v", err)
	}
	originalHash := first.User.PasswordHash

	second, err := EnsureAdmin(t.Context(), store, auth.NewPasswordHasher(4), input)
	if err != nil || second.Created || len(store.usersByEmail) != 1 {
		t.Fatalf("second=%#v users=%d err=%v", second, len(store.usersByEmail), err)
	}
	if second.User.PasswordHash != originalHash {
		t.Fatal("idempotent seed changed the existing Admin password")
	}
}

func TestEnsureAdminRejectsInvalidConfiguration(t *testing.T) {
	tests := []AdminInput{
		{Email: "", Password: "ChangeMe123!", FullName: "Admin", EmployeeCode: "DEV-ADMIN"},
		{Email: "admin@local.test", Password: "", FullName: "Admin", EmployeeCode: "DEV-ADMIN"},
		{Email: "admin@local.test", Password: "ChangeMe123!", FullName: "", EmployeeCode: "DEV-ADMIN"},
		{Email: "admin@local.test", Password: "ChangeMe123!", FullName: "Admin", EmployeeCode: ""},
	}
	for _, input := range tests {
		if _, err := EnsureAdmin(t.Context(), newFakeAdminStore(), auth.NewPasswordHasher(4), input); !errors.Is(err, ErrInvalidAdminInput) {
			t.Fatalf("input=%#v err=%v", input, err)
		}
	}
}

func TestEnsureAdminRejectsExistingNonAdminAccount(t *testing.T) {
	store := newFakeAdminStore()
	store.usersByEmail["admin@local.test"] = auth.User{
		ID: "existing", Email: "admin@local.test", EmployeeCode: "EMP-1",
		Role: auth.RoleEmployee, Status: auth.StatusActive,
	}
	_, err := EnsureAdmin(t.Context(), store, auth.NewPasswordHasher(4), AdminInput{
		Email: "admin@local.test", Password: "ChangeMe123!",
		FullName: "Development Admin", EmployeeCode: "DEV-ADMIN",
	})
	if !errors.Is(err, ErrAdminConflict) {
		t.Fatalf("expected Admin conflict, got %v", err)
	}
}

func TestEnsureAdminPreservesDuplicateEmployeeCodeError(t *testing.T) {
	store := newFakeAdminStore()
	store.usersByEmail["other@local.test"] = auth.User{
		ID: "existing", Email: "other@local.test", EmployeeCode: "DEV-ADMIN",
		Role: auth.RoleEmployee, Status: auth.StatusActive,
	}
	_, err := EnsureAdmin(t.Context(), store, auth.NewPasswordHasher(4), AdminInput{
		Email: "admin@local.test", Password: "ChangeMe123!",
		FullName: "Development Admin", EmployeeCode: "DEV-ADMIN",
	})
	if !errors.Is(err, auth.ErrCodeAlreadyExists) {
		t.Fatalf("expected duplicate employee code, got %v", err)
	}
}

type fakeAdminStore struct {
	usersByEmail map[string]auth.User
}

func newFakeAdminStore() *fakeAdminStore {
	return &fakeAdminStore{usersByEmail: make(map[string]auth.User)}
}

func (s *fakeAdminStore) FindByEmail(_ context.Context, email string) (auth.User, error) {
	user, exists := s.usersByEmail[strings.ToLower(email)]
	if !exists {
		return auth.User{}, auth.ErrUserNotFound
	}
	return user, nil
}

func (s *fakeAdminStore) CreateUser(_ context.Context, user auth.User) (auth.User, error) {
	for _, existing := range s.usersByEmail {
		if existing.EmployeeCode == user.EmployeeCode {
			return auth.User{}, auth.ErrCodeAlreadyExists
		}
	}
	s.usersByEmail[strings.ToLower(user.Email)] = user
	return user, nil
}
