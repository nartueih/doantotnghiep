package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"license-manager/backend/internal/platform/database"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) List(ctx context.Context, filter Filter) ([]Log, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT a.id::text, COALESCE(a.actor_id::text, ''), COALESCE(u.full_name, ''),
		       COALESCE(u.email, ''), a.action, a.entity_type,
		       COALESCE(a.entity_id::text, ''), a.metadata,
		       COALESCE(a.ip_address::text, ''), a.created_at
		FROM audit_logs a
		LEFT JOIN users u ON u.id = a.actor_id
		WHERE ($1 = '' OR a.action = $1)
		  AND ($2 = '' OR a.entity_type = $2)
		  AND ($3 = '' OR a.actor_id::text = $3)
		ORDER BY a.created_at DESC, a.id DESC
		LIMIT $4`, filter.Action, filter.EntityType, filter.ActorID, filter.Limit)
	if err != nil {
		return nil, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()

	items := make([]Log, 0)
	for rows.Next() {
		item, err := scanLog(rows)
		if err != nil {
			return nil, fmt.Errorf("scan audit log: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit logs: %w", err)
	}
	return items, nil
}

func (r *PostgresRepository) Create(ctx context.Context, item Log) (Log, error) {
	metadata, err := json.Marshal(item.Metadata)
	if err != nil {
		return Log{}, fmt.Errorf("encode audit metadata: %w", err)
	}
	_, err = database.Querier(ctx, r.pool).Exec(ctx, `
		INSERT INTO audit_logs (id, actor_id, action, entity_type, entity_id, metadata, ip_address, created_at)
		VALUES ($1, NULLIF($2, '')::uuid, $3, $4, NULLIF($5, '')::uuid, $6, NULLIF($7, '')::inet, $8)`,
		item.ID, item.ActorID, item.Action, item.EntityType, item.EntityID, metadata, item.IPAddress, item.CreatedAt)
	if err != nil {
		return Log{}, fmt.Errorf("create audit log: %w", err)
	}
	return item, nil
}

type scanner interface {
	Scan(...any) error
}

func scanLog(row scanner) (Log, error) {
	var item Log
	var metadata []byte
	err := row.Scan(
		&item.ID,
		&item.ActorID,
		&item.ActorName,
		&item.ActorEmail,
		&item.Action,
		&item.EntityType,
		&item.EntityID,
		&metadata,
		&item.IPAddress,
		&item.CreatedAt,
	)
	if err != nil {
		return Log{}, err
	}
	if err := json.Unmarshal(metadata, &item.Metadata); err != nil {
		return Log{}, err
	}
	return item, nil
}
