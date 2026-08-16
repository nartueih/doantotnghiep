package auth

import (
	"context"
	"errors"
	"strings"
)

type Service struct {
	repository Repository
	hasher     PasswordHasher
	tokens     *TokenManager
}

func NewService(repository Repository, hasher PasswordHasher, tokens *TokenManager) *Service {
	return &Service{repository: repository, hasher: hasher, tokens: tokens}
}

func (s *Service) Login(ctx context.Context, email, password string) (AuthResult, error) {
	user, err := s.repository.FindByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if errors.Is(err, ErrUserNotFound) {
		return AuthResult{}, ErrInvalidCredentials
	}
	if err != nil {
		return AuthResult{}, err
	}
	if user.Status == StatusLocked {
		return AuthResult{}, ErrAccountLocked
	}
	if err := s.hasher.Compare(user.PasswordHash, password); err != nil {
		return AuthResult{}, ErrInvalidCredentials
	}

	pair, err := s.tokens.IssuePair(user)
	if err != nil {
		return AuthResult{}, err
	}
	if err := s.repository.SaveRefreshSession(ctx, RefreshSession{
		UserID:    user.ID,
		TokenHash: HashToken(pair.RefreshToken),
		ExpiresAt: pair.refreshExpiresAt,
	}); err != nil {
		return AuthResult{}, err
	}

	return AuthResult{Tokens: pair, User: user}, nil
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (AuthResult, error) {
	claims, err := s.tokens.ParseRefresh(refreshToken)
	if err != nil {
		return AuthResult{}, ErrInvalidRefreshToken
	}

	user, err := s.repository.FindByID(ctx, claims.Subject)
	if errors.Is(err, ErrUserNotFound) {
		return AuthResult{}, ErrInvalidRefreshToken
	}
	if err != nil {
		return AuthResult{}, err
	}
	if user.Status == StatusLocked {
		return AuthResult{}, ErrAccountLocked
	}

	pair, err := s.tokens.IssuePair(user)
	if err != nil {
		return AuthResult{}, err
	}
	err = s.repository.RotateRefreshSession(ctx, HashToken(refreshToken), RefreshSession{
		UserID:    user.ID,
		TokenHash: HashToken(pair.RefreshToken),
		ExpiresAt: pair.refreshExpiresAt,
	})
	if errors.Is(err, ErrInvalidRefreshToken) {
		return AuthResult{}, ErrInvalidRefreshToken
	}
	if err != nil {
		return AuthResult{}, err
	}

	return AuthResult{Tokens: pair, User: user}, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if _, err := s.tokens.ParseRefresh(refreshToken); err != nil {
		return ErrInvalidRefreshToken
	}
	return s.repository.RevokeRefreshSession(ctx, HashToken(refreshToken))
}

func (s *Service) Me(ctx context.Context, userID string) (User, error) {
	return s.repository.FindByID(ctx, userID)
}
