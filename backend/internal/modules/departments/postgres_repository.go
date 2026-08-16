package departments

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

func (r *PostgresRepository) List(ctx context.Context) ([]Department, error) {
	rows, err := r.pool.Query(ctx, departmentSelect+` ORDER BY code, id`)
	if err != nil {
		return nil, fmt.Errorf("list departments: %w", err)
	}
	defer rows.Close()
	items := make([]Department, 0)
	for rows.Next() {
		item, err := scanDepartment(rows)
		if err != nil {
			return nil, fmt.Errorf("scan department: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate departments: %w", err)
	}
	return items, nil
}

func (r *PostgresRepository) FindByID(ctx context.Context, departmentID string) (Department, error) {
	item, err := scanDepartment(r.pool.QueryRow(ctx, departmentSelect+` WHERE id = $1`, departmentID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Department{}, ErrNotFound
	}
	if err != nil {
		return Department{}, fmt.Errorf("find department: %w", err)
	}
	return item, nil
}

func (r *PostgresRepository) Create(ctx context.Context, item Department) (Department, error) {
	_, err := r.pool.Exec(ctx, `INSERT INTO departments (id, name, code) VALUES ($1, $2, $3)`, item.ID, item.Name, item.Code)
	if err != nil {
		return Department{}, mapWriteError("create department", err)
	}
	return r.FindByID(ctx, item.ID)
}

func (r *PostgresRepository) Update(ctx context.Context, item Department) (Department, error) {
	result, err := r.pool.Exec(ctx, `
		UPDATE departments SET name = $2, code = $3, updated_at = NOW() WHERE id = $1`, item.ID, item.Name, item.Code)
	if err != nil {
		return Department{}, mapWriteError("update department", err)
	}
	if result.RowsAffected() != 1 {
		return Department{}, ErrNotFound
	}
	return r.FindByID(ctx, item.ID)
}

const departmentSelect = `SELECT id::text, name, code, created_at, updated_at FROM departments`

type scanner interface {
	Scan(...any) error
}

func scanDepartment(row scanner) (Department, error) {
	var item Department
	err := row.Scan(&item.ID, &item.Name, &item.Code, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func mapWriteError(operation string, err error) error {
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && postgresError.Code == "23505" {
		switch postgresError.ConstraintName {
		case "uq_departments_name":
			return ErrNameExists
		case "uq_departments_code":
			return ErrCodeExists
		}
	}
	return fmt.Errorf("%s: %w", operation, err)
}
