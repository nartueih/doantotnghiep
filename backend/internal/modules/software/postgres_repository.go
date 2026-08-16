package software

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

func (r *PostgresRepository) List(ctx context.Context) ([]Product, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, name, publisher, version, description, created_at, updated_at
		FROM software_products
		ORDER BY name, publisher, version`)
	if err != nil {
		return nil, fmt.Errorf("list software products: %w", err)
	}
	defer rows.Close()

	products := make([]Product, 0)
	for rows.Next() {
		var product Product
		if err := rows.Scan(
			&product.ID,
			&product.Name,
			&product.Publisher,
			&product.Version,
			&product.Description,
			&product.CreatedAt,
			&product.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan software product: %w", err)
		}
		products = append(products, product)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate software products: %w", err)
	}
	return products, nil
}

func (r *PostgresRepository) FindByID(ctx context.Context, productID string) (Product, error) {
	var product Product
	err := r.pool.QueryRow(ctx, `
		SELECT id::text, name, publisher, version, description, created_at, updated_at
		FROM software_products
		WHERE id = $1`, productID).Scan(
		&product.ID,
		&product.Name,
		&product.Publisher,
		&product.Version,
		&product.Description,
		&product.CreatedAt,
		&product.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Product{}, ErrNotFound
	}
	if err != nil {
		return Product{}, fmt.Errorf("find software product: %w", err)
	}
	return product, nil
}

func (r *PostgresRepository) Create(ctx context.Context, product Product) (Product, error) {
	return r.save(ctx, `
		INSERT INTO software_products (id, name, publisher, version, description)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text, name, publisher, version, description, created_at, updated_at`, product)
}

func (r *PostgresRepository) Update(ctx context.Context, product Product) (Product, error) {
	updated, err := r.save(ctx, `
		UPDATE software_products
		SET name = $2, publisher = $3, version = $4, description = $5, updated_at = NOW()
		WHERE id = $1
		RETURNING id::text, name, publisher, version, description, created_at, updated_at`, product)
	if errors.Is(err, pgx.ErrNoRows) {
		return Product{}, ErrNotFound
	}
	return updated, err
}

func (r *PostgresRepository) save(ctx context.Context, query string, product Product) (Product, error) {
	var saved Product
	err := r.pool.QueryRow(
		ctx,
		query,
		product.ID,
		product.Name,
		product.Publisher,
		product.Version,
		product.Description,
	).Scan(
		&saved.ID,
		&saved.Name,
		&saved.Publisher,
		&saved.Version,
		&saved.Description,
		&saved.CreatedAt,
		&saved.UpdatedAt,
	)
	if err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) && postgresError.Code == "23505" {
			return Product{}, ErrAlreadyExists
		}
		return Product{}, err
	}
	return saved, nil
}
