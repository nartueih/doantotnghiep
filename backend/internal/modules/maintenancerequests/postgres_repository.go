package maintenancerequests

import (
	"context"
	"errors"
	"fmt"
	"time"

	"license-manager/backend/internal/platform/database"

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

func (r *PostgresRepository) List(ctx context.Context, filter Filter) ([]Request, error) {
	rows, err := database.Querier(ctx, r.pool).Query(ctx, maintenanceRequestSelect+`
		WHERE ($1 = '' OR r.status = $1)
		  AND ($2 = '' OR r.priority = $2)
		  AND ($3 = '' OR r.category = $3)
		  AND ($4 = '' OR r.requester_name ILIKE '%' || $4 || '%'
		               OR r.device_asset_code ILIKE '%' || $4 || '%'
		               OR r.device_serial_number ILIKE '%' || $4 || '%'
		               OR r.device_name ILIKE '%' || $4 || '%'
		               OR r.title ILIKE '%' || $4 || '%')
		ORDER BY r.created_at DESC, r.id DESC`, filter.Status, filter.Priority, filter.Category, filter.Search)
	if err != nil {
		return nil, fmt.Errorf("list maintenance requests: %w", err)
	}
	defer rows.Close()
	return scanMaintenanceRequests(rows)
}

func (r *PostgresRepository) ListByRequester(ctx context.Context, requesterID string) ([]Request, error) {
	rows, err := database.Querier(ctx, r.pool).Query(ctx, maintenanceRequestSelect+`
		WHERE r.requester_id = $1
		ORDER BY r.created_at DESC, r.id DESC`, requesterID)
	if err != nil {
		return nil, fmt.Errorf("list requester maintenance requests: %w", err)
	}
	defer rows.Close()
	return scanMaintenanceRequests(rows)
}

func (r *PostgresRepository) FindForUpdate(ctx context.Context, requestID string) (Request, error) {
	return r.find(ctx, maintenanceRequestSelect+` WHERE r.id = $1 FOR UPDATE`, requestID)
}

func (r *PostgresRepository) Create(ctx context.Context, item Request) (Request, error) {
	created, err := scanMaintenanceRequest(database.Querier(ctx, r.pool).QueryRow(ctx, `
		WITH inserted AS (
			INSERT INTO maintenance_requests (
				id, requester_id, requester_name, device_id, device_asset_code,
				device_serial_number, device_name, device_type, device_manufacturer,
				device_model, device_purchased_at, device_warranty_expires_at,
				category, priority, title, description, status,
				last_actor_id, last_actor_name, created_at, updated_at
			)
			SELECT
				$1, $2, $3, d.id, d.asset_code, d.serial_number, d.name, d.device_type,
				d.manufacturer, d.model, d.purchased_at, d.warranty_expires_at,
				$5, $6, $7, $8, $9, $10, $11, $12, $13
			FROM devices d
			WHERE d.id = $4 AND d.assigned_user_id = $2
			FOR UPDATE
			RETURNING *
		)
		SELECT `+maintenanceRequestColumns+` FROM inserted r`,
		item.ID, item.RequesterID, item.RequesterName, item.DeviceID,
		item.Category, item.Priority, item.Title, item.Description, item.Status,
		item.LastActorID, item.LastActorName, item.CreatedAt, item.UpdatedAt,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Request{}, ErrDeviceNotFound
		}
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) && postgresError.Code == "23505" && postgresError.ConstraintName == "uq_open_maintenance_request" {
			return Request{}, ErrOpenDuplicate
		}
		return Request{}, fmt.Errorf("create maintenance request: %w", err)
	}
	return created, nil
}

func (r *PostgresRepository) Cancel(ctx context.Context, requestID, requesterID string, cancelledAt time.Time) (Request, error) {
	item, err := scanMaintenanceRequest(database.Querier(ctx, r.pool).QueryRow(ctx, `
		WITH updated AS (
			UPDATE maintenance_requests
			SET status = 'cancelled', last_actor_id = requester_id,
			    last_actor_name = requester_name, cancelled_at = $3, updated_at = $3
			WHERE id = $1 AND requester_id = $2 AND status = 'pending'
			RETURNING *
		)
		SELECT `+maintenanceRequestColumns+` FROM updated r`, requestID, requesterID, cancelledAt))
	if errors.Is(err, pgx.ErrNoRows) {
		return Request{}, r.cancelError(ctx, requestID, requesterID)
	}
	if err != nil {
		return Request{}, fmt.Errorf("cancel maintenance request: %w", err)
	}
	return item, nil
}

func (r *PostgresRepository) Accept(ctx context.Context, update AcceptUpdate) (Request, error) {
	item, err := scanMaintenanceRequest(database.Querier(ctx, r.pool).QueryRow(ctx, `
		WITH updated AS (
			UPDATE maintenance_requests
			SET status = 'in_progress', assigned_to = $2, assigned_to_name = $3,
			    last_actor_id = $2, last_actor_name = $3,
			    accepted_at = $4, updated_at = $4
			WHERE id = $1 AND status = 'pending'
			RETURNING *
		)
		SELECT `+maintenanceRequestColumns+` FROM updated r`, update.RequestID, update.ActorID, update.ActorName, update.AcceptedAt))
	if errors.Is(err, pgx.ErrNoRows) {
		return Request{}, r.stateChangeError(ctx, update.RequestID)
	}
	if err != nil {
		return Request{}, fmt.Errorf("accept maintenance request: %w", err)
	}
	return item, nil
}

func (r *PostgresRepository) Complete(ctx context.Context, update CompleteUpdate) (Request, error) {
	item, err := scanMaintenanceRequest(database.Querier(ctx, r.pool).QueryRow(ctx, `
		WITH updated AS (
			UPDATE maintenance_requests
			SET status = 'completed', last_actor_id = $2, last_actor_name = $3,
			    response_note = $4, completed_at = $5, updated_at = $5
			WHERE id = $1 AND status = 'in_progress'
			RETURNING *
		)
		SELECT `+maintenanceRequestColumns+` FROM updated r`,
		update.RequestID, update.ActorID, update.ActorName, update.ResponseNote, update.CompletedAt))
	if errors.Is(err, pgx.ErrNoRows) {
		return Request{}, r.stateChangeError(ctx, update.RequestID)
	}
	if err != nil {
		return Request{}, fmt.Errorf("complete maintenance request: %w", err)
	}
	return item, nil
}

func (r *PostgresRepository) Reject(ctx context.Context, update RejectUpdate) (Request, error) {
	item, err := scanMaintenanceRequest(database.Querier(ctx, r.pool).QueryRow(ctx, `
		WITH updated AS (
			UPDATE maintenance_requests
			SET status = 'rejected', last_actor_id = $2, last_actor_name = $3,
			    response_note = $4, rejected_at = $5, updated_at = $5
			WHERE id = $1 AND status IN ('pending', 'in_progress')
			RETURNING *
		)
		SELECT `+maintenanceRequestColumns+` FROM updated r`,
		update.RequestID, update.ActorID, update.ActorName, update.ResponseNote, update.RejectedAt))
	if errors.Is(err, pgx.ErrNoRows) {
		return Request{}, r.stateChangeError(ctx, update.RequestID)
	}
	if err != nil {
		return Request{}, fmt.Errorf("reject maintenance request: %w", err)
	}
	return item, nil
}

func (r *PostgresRepository) find(ctx context.Context, query, requestID string) (Request, error) {
	item, err := scanMaintenanceRequest(database.Querier(ctx, r.pool).QueryRow(ctx, query, requestID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Request{}, ErrNotFound
	}
	if err != nil {
		return Request{}, fmt.Errorf("find maintenance request: %w", err)
	}
	return item, nil
}

func (r *PostgresRepository) cancelError(ctx context.Context, requestID, requesterID string) error {
	var storedRequesterID, status string
	err := database.Querier(ctx, r.pool).QueryRow(ctx, `
		SELECT requester_id::text, status FROM maintenance_requests WHERE id = $1`, requestID).Scan(&storedRequesterID, &status)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && storedRequesterID != requesterID) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("inspect maintenance request cancellation: %w", err)
	}
	return ErrInvalidState
}

func (r *PostgresRepository) stateChangeError(ctx context.Context, requestID string) error {
	var exists bool
	if err := database.Querier(ctx, r.pool).QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM maintenance_requests WHERE id = $1)`, requestID).Scan(&exists); err != nil {
		return fmt.Errorf("inspect maintenance request state: %w", err)
	}
	if !exists {
		return ErrNotFound
	}
	return ErrInvalidState
}

const maintenanceRequestColumns = `
	r.id::text,
	r.requester_id::text,
	r.requester_name,
	r.device_id::text,
	r.device_asset_code,
	COALESCE(r.device_serial_number, ''),
	r.device_name,
	r.device_type,
	COALESCE(r.device_manufacturer, ''),
	COALESCE(r.device_model, ''),
	COALESCE(r.device_purchased_at::text, ''),
	COALESCE(r.device_warranty_expires_at::text, ''),
	r.category,
	r.priority,
	r.title,
	r.description,
	r.status,
	COALESCE(r.assigned_to::text, ''),
	COALESCE(r.assigned_to_name, ''),
	r.last_actor_id::text,
	r.last_actor_name,
	COALESCE(r.response_note, ''),
	r.created_at,
	r.updated_at,
	r.accepted_at,
	r.completed_at,
	r.rejected_at,
	r.cancelled_at`

const maintenanceRequestSelect = `SELECT ` + maintenanceRequestColumns + ` FROM maintenance_requests r`

type maintenanceRequestScanner interface {
	Scan(...any) error
}

func scanMaintenanceRequest(row maintenanceRequestScanner) (Request, error) {
	var item Request
	err := row.Scan(
		&item.ID, &item.RequesterID, &item.RequesterName,
		&item.DeviceID, &item.DeviceAssetCode, &item.DeviceSerialNumber,
		&item.DeviceName, &item.DeviceType, &item.DeviceManufacturer, &item.DeviceModel,
		&item.DevicePurchasedAt, &item.DeviceWarrantyExpiresAt,
		&item.Category, &item.Priority, &item.Title, &item.Description, &item.Status,
		&item.AssignedTo, &item.AssignedToName, &item.LastActorID, &item.LastActorName,
		&item.ResponseNote, &item.CreatedAt, &item.UpdatedAt,
		&item.AcceptedAt, &item.CompletedAt, &item.RejectedAt, &item.CancelledAt,
	)
	return item, err
}

func scanMaintenanceRequests(rows pgx.Rows) ([]Request, error) {
	items := make([]Request, 0)
	for rows.Next() {
		item, err := scanMaintenanceRequest(rows)
		if err != nil {
			return nil, fmt.Errorf("scan maintenance request: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate maintenance requests: %w", err)
	}
	return items, nil
}
