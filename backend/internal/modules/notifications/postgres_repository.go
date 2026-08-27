package notifications

import (
	"context"
	"errors"
	"fmt"
	"time"

	"license-manager/backend/internal/platform/database"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) ListByUser(ctx context.Context, userID string) ([]Notification, error) {
	rows, err := database.Querier(ctx, r.pool).Query(ctx, notificationSelect+`
		WHERE user_id = $1
		ORDER BY created_at DESC, id DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	items := make([]Notification, 0)
	for rows.Next() {
		item, err := scanNotification(rows)
		if err != nil {
			return nil, fmt.Errorf("scan notification: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	return items, nil
}

func (r *PostgresRepository) Create(ctx context.Context, item Notification) (Notification, error) {
	created, err := scanNotification(database.Querier(ctx, r.pool).QueryRow(ctx, `
		INSERT INTO notifications (
			id, user_id, type, title, message, entity_type, entity_id, created_at, read_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id::text, user_id::text, type, title, message, entity_type,
		          entity_id::text, created_at, read_at`,
		item.ID,
		item.UserID,
		item.Type,
		item.Title,
		item.Message,
		item.EntityType,
		item.EntityID,
		item.CreatedAt,
		item.ReadAt,
	))
	if err != nil {
		return Notification{}, fmt.Errorf("create notification: %w", err)
	}
	return created, nil
}

func (r *PostgresRepository) MarkRead(
	ctx context.Context,
	userID string,
	notificationID string,
	readAt time.Time,
) (Notification, error) {
	item, err := scanNotification(database.Querier(ctx, r.pool).QueryRow(ctx, `
		UPDATE notifications
		SET read_at = COALESCE(read_at, $3)
		WHERE id = $1 AND user_id = $2
		RETURNING id::text, user_id::text, type, title, message, entity_type,
		          entity_id::text, created_at, read_at`, notificationID, userID, readAt))
	if errors.Is(err, pgx.ErrNoRows) {
		return Notification{}, ErrNotFound
	}
	if err != nil {
		return Notification{}, fmt.Errorf("mark notification read: %w", err)
	}
	return item, nil
}

func (r *PostgresRepository) MarkAllRead(ctx context.Context, userID string, readAt time.Time) (int, error) {
	result, err := database.Querier(ctx, r.pool).Exec(ctx, `
		UPDATE notifications
		SET read_at = $2
		WHERE user_id = $1 AND read_at IS NULL`, userID, readAt)
	if err != nil {
		return 0, fmt.Errorf("mark all notifications read: %w", err)
	}
	return int(result.RowsAffected()), nil
}

const notificationSelect = `
	SELECT id::text, user_id::text, type, title, message, entity_type,
	       entity_id::text, created_at, read_at
	FROM notifications`

type notificationScanner interface {
	Scan(...any) error
}

func scanNotification(row notificationScanner) (Notification, error) {
	var item Notification
	err := row.Scan(
		&item.ID,
		&item.UserID,
		&item.Type,
		&item.Title,
		&item.Message,
		&item.EntityType,
		&item.EntityID,
		&item.CreatedAt,
		&item.ReadAt,
	)
	return item, err
}
