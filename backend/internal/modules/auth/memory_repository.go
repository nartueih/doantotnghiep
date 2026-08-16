package auth

import (
	"context"
	"strings"
	"sync"
	"time"
)

type MemoryRepository struct {
	mu       sync.RWMutex
	users    map[string]User
	byEmail  map[string]string
	sessions map[string]RefreshSession
}

func NewMemoryRepository(users []User) *MemoryRepository {
	repository := &MemoryRepository{
		users:    make(map[string]User),
		byEmail:  make(map[string]string),
		sessions: make(map[string]RefreshSession),
	}
	for _, user := range users {
		repository.users[user.ID] = user
		repository.byEmail[strings.ToLower(user.Email)] = user.ID
	}
	return repository
}

func (r *MemoryRepository) FindByEmail(_ context.Context, email string) (User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	userID, exists := r.byEmail[strings.ToLower(email)]
	if !exists {
		return User{}, ErrUserNotFound
	}
	return r.users[userID], nil
}

func (r *MemoryRepository) FindByID(_ context.Context, userID string) (User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, exists := r.users[userID]
	if !exists {
		return User{}, ErrUserNotFound
	}
	return user, nil
}

func (r *MemoryRepository) SaveRefreshSession(_ context.Context, session RefreshSession) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[session.TokenHash] = session
	return nil
}

func (r *MemoryRepository) RotateRefreshSession(_ context.Context, oldTokenHash string, replacement RefreshSession) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	oldSession, exists := r.sessions[oldTokenHash]
	if !exists || !oldSession.ExpiresAt.After(time.Now()) {
		return ErrInvalidRefreshToken
	}
	delete(r.sessions, oldTokenHash)
	r.sessions[replacement.TokenHash] = replacement
	return nil
}

func (r *MemoryRepository) RevokeRefreshSession(_ context.Context, tokenHash string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.sessions[tokenHash]; !exists {
		return ErrInvalidRefreshToken
	}
	delete(r.sessions, tokenHash)
	return nil
}
