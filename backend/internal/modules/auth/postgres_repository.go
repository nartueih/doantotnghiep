package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

func (r *PostgresRepository) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, email, password_hash, full_name, employee_code, role, status, created_at
		FROM users
		ORDER BY created_at, id`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		var user User
		if err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.PasswordHash,
			&user.FullName,
			&user.EmployeeCode,
			&user.Role,
			&user.Status,
			&user.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}
	return users, nil
}

func (r *PostgresRepository) CreateUser(ctx context.Context, user User) (User, error) {
	var created User
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (id, email, password_hash, full_name, employee_code, role, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id::text, email, password_hash, full_name, employee_code, role, status, created_at`,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.FullName,
		user.EmployeeCode,
		user.Role,
		user.Status,
	).Scan(
		&created.ID,
		&created.Email,
		&created.PasswordHash,
		&created.FullName,
		&created.EmployeeCode,
		&created.Role,
		&created.Status,
		&created.CreatedAt,
	)
	if err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) && postgresError.Code == "23505" {
			switch postgresError.ConstraintName {
			case "users_email_key":
				return User{}, ErrEmailAlreadyExists
			case "users_employee_code_key":
				return User{}, ErrCodeAlreadyExists
			}
		}
		return User{}, fmt.Errorf("create user: %w", err)
	}
	return created, nil
}

func (r *PostgresRepository) UpdateUserStatus(ctx context.Context, userID, status string) (User, error) {
	var user User
	err := r.pool.QueryRow(ctx, `
		UPDATE users
		SET status = $2, updated_at = NOW()
		WHERE id = $1
		RETURNING id::text, email, password_hash, full_name, employee_code, role, status, created_at`,
		userID,
		status,
	).Scan(
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
		return User{}, fmt.Errorf("update user status: %w", err)
	}
	return user, nil
}
