package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) FindByEmail(ctx context.Context, email string) (User, error) {
	return r.findUser(ctx, `
		SELECT id::text, email, password_hash, full_name, employee_code, role, status, created_at
		FROM users
		WHERE LOWER(email) = LOWER($1)`, email)
}

func (r *PostgresRepository) FindByID(ctx context.Context, userID string) (User, error) {
	return r.findUser(ctx, `
		SELECT id::text, email, password_hash, full_name, employee_code, role, status, created_at
		FROM users
		WHERE id = $1`, userID)
}

func (r *PostgresRepository) findUser(ctx context.Context, query string, argument any) (User, error) {
	var user User
	err := r.pool.QueryRow(ctx, query, argument).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FullName,
		&user.EmployeeCode,
		&user.Role,
		&user.Status,
		&user.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrUserNotFound
	}
	if err != nil {
		return User{}, fmt.Errorf("find user: %w", err)
	}
	return user, nil
}

func (r *PostgresRepository) SaveRefreshSession(ctx context.Context, session RefreshSession) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`, session.UserID, session.TokenHash, session.ExpiresAt)
	if err != nil {
		return fmt.Errorf("save refresh session: %w", err)
	}
	return nil
}

func (r *PostgresRepository) RotateRefreshSession(ctx context.Context, oldTokenHash string, replacement RefreshSession) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin refresh rotation: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	result, err := tx.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = NOW(), replaced_by_hash = $2
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()`,
		oldTokenHash, replacement.TokenHash)
	if err != nil {
		return fmt.Errorf("consume refresh session: %w", err)
	}
	if result.RowsAffected() != 1 {
		return ErrInvalidRefreshToken
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`, replacement.UserID, replacement.TokenHash, replacement.ExpiresAt)
	if err != nil {
		return fmt.Errorf("save replacement refresh session: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit refresh rotation: %w", err)
	}
	return nil
}

func (r *PostgresRepository) RevokeRefreshSession(ctx context.Context, tokenHash string) error {
	result, err := r.pool.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE token_hash = $1 AND revoked_at IS NULL AND expires_at > NOW()`, tokenHash)
	if err != nil {
		return fmt.Errorf("revoke refresh session: %w", err)
	}
	if result.RowsAffected() != 1 {
		return ErrInvalidRefreshToken
	}
	return nil
}
