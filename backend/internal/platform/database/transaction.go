package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DBTX interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Transactor interface {
	WithinTransaction(context.Context, func(context.Context) error) error
}

type DirectTransactor struct{}

func (DirectTransactor) WithinTransaction(ctx context.Context, callback func(context.Context) error) error {
	return callback(ctx)
}

type PostgresTransactor struct {
	pool *pgxpool.Pool
}

func NewPostgresTransactor(pool *pgxpool.Pool) *PostgresTransactor {
	return &PostgresTransactor{pool: pool}
}

type transactionContextKey struct{}

func (t *PostgresTransactor) WithinTransaction(
	ctx context.Context,
	callback func(context.Context) error,
) error {
	if _, exists := ctx.Value(transactionContextKey{}).(pgx.Tx); exists {
		return callback(ctx)
	}

	transaction, err := t.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	transactionContext := context.WithValue(ctx, transactionContextKey{}, transaction)

	if err := callback(transactionContext); err != nil {
		_ = transaction.Rollback(context.Background())
		return err
	}
	if err := transaction.Commit(ctx); err != nil {
		_ = transaction.Rollback(context.Background())
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func Querier(ctx context.Context, pool *pgxpool.Pool) DBTX {
	if transaction, exists := ctx.Value(transactionContextKey{}).(pgx.Tx); exists {
		return transaction
	}
	return pool
}
