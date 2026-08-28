package licenserequests

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
	rows, err := database.Querier(ctx, r.pool).Query(ctx, requestSelect+`
		WHERE ($1 = '' OR r.status = $1)
		  AND ($2 = '' OR r.priority = $2)
		  AND ($3 = '' OR r.requester_name ILIKE '%' || $3 || '%'
		               OR r.software_product_name ILIKE '%' || $3 || '%')
		ORDER BY r.created_at DESC, r.id DESC`, filter.Status, filter.Priority, filter.Search)
	if err != nil {
		return nil, fmt.Errorf("list license requests: %w", err)
	}
	defer rows.Close()
	return scanRequests(rows)
}

func (r *PostgresRepository) ListByRequester(ctx context.Context, requesterID string) ([]Request, error) {
	rows, err := database.Querier(ctx, r.pool).Query(ctx, requestSelect+`
		WHERE r.requester_id = $1
		ORDER BY r.created_at DESC, r.id DESC`, requesterID)
	if err != nil {
		return nil, fmt.Errorf("list requester license requests: %w", err)
	}
	defer rows.Close()
	return scanRequests(rows)
}

func (r *PostgresRepository) FindByID(ctx context.Context, requestID string) (Request, error) {
	return r.find(ctx, requestSelect+` WHERE r.id = $1`, requestID)
}

func (r *PostgresRepository) FindForUpdate(ctx context.Context, requestID string) (Request, error) {
	return r.find(ctx, requestSelect+` WHERE r.id = $1 FOR UPDATE`, requestID)
}

func (r *PostgresRepository) Create(ctx context.Context, item Request) (Request, error) {
	created, err := scanRequest(database.Querier(ctx, r.pool).QueryRow(ctx, `
		WITH inserted AS (
			INSERT INTO license_requests (
				id, requester_id, requester_name, software_product_id,
				software_product_name, priority, reason, status, created_at, updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			RETURNING *
		)
		SELECT `+requestColumns+` FROM inserted r`,
		item.ID,
		item.RequesterID,
		item.RequesterName,
		item.SoftwareProductID,
		item.SoftwareProductName,
		item.Priority,
		item.Reason,
		item.Status,
		item.CreatedAt,
		item.UpdatedAt,
	))
	if err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) && postgresError.Code == "23505" && postgresError.ConstraintName == "uq_pending_license_request" {
			return Request{}, ErrPendingDuplicate
		}
		return Request{}, fmt.Errorf("create license request: %w", err)
	}
	return created, nil
}

func (r *PostgresRepository) Cancel(
	ctx context.Context,
	requestID string,
	requesterID string,
	cancelledAt time.Time,
) (Request, error) {
	item, err := scanRequest(database.Querier(ctx, r.pool).QueryRow(ctx, `
		WITH updated AS (
			UPDATE license_requests
			SET status = 'cancelled', cancelled_at = $3, updated_at = $3
			WHERE id = $1 AND requester_id = $2 AND status = 'pending'
			RETURNING *
		)
		SELECT `+requestColumns+` FROM updated r`, requestID, requesterID, cancelledAt))
	if errors.Is(err, pgx.ErrNoRows) {
		return Request{}, r.cancelError(ctx, requestID, requesterID)
	}
	if err != nil {
		return Request{}, fmt.Errorf("cancel license request: %w", err)
	}
	return item, nil
}

func (r *PostgresRepository) Approve(ctx context.Context, update ApprovalUpdate) (Request, error) {
	item, err := scanRequest(database.Querier(ctx, r.pool).QueryRow(ctx, `
		WITH updated AS (
			UPDATE license_requests
			SET status = 'approved',
			    selected_license_id = $2,
			    selected_license_name = $3,
			    assignment_id = $4,
			    reviewed_by = $5,
			    reviewed_by_name = $6,
			    response_note = NULLIF($7, ''),
			    reviewed_at = $8,
			    updated_at = $8
			WHERE id = $1 AND status = 'pending'
			RETURNING *
		)
		SELECT `+requestColumns+` FROM updated r`,
		update.RequestID,
		update.LicenseID,
		update.LicenseName,
		update.AssignmentID,
		update.ReviewerID,
		update.ReviewerName,
		update.ResponseNote,
		update.ReviewedAt,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return Request{}, r.stateChangeError(ctx, update.RequestID)
	}
	if err != nil {
		return Request{}, fmt.Errorf("approve license request: %w", err)
	}
	return item, nil
}

func (r *PostgresRepository) Reject(ctx context.Context, update RejectionUpdate) (Request, error) {
	item, err := scanRequest(database.Querier(ctx, r.pool).QueryRow(ctx, `
		WITH updated AS (
			UPDATE license_requests
			SET status = 'rejected',
			    reviewed_by = $2,
			    reviewed_by_name = $3,
			    decision_reason = $4,
			    response_note = $5,
			    reviewed_at = $6,
			    updated_at = $6
			WHERE id = $1 AND status = 'pending'
			RETURNING *
		)
		SELECT `+requestColumns+` FROM updated r`,
		update.RequestID,
		update.ReviewerID,
		update.ReviewerName,
		update.DecisionReason,
		update.ResponseNote,
		update.ReviewedAt,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return Request{}, r.stateChangeError(ctx, update.RequestID)
	}
	if err != nil {
		return Request{}, fmt.Errorf("reject license request: %w", err)
	}
	return item, nil
}

func (r *PostgresRepository) find(ctx context.Context, query string, requestID string) (Request, error) {
	item, err := scanRequest(database.Querier(ctx, r.pool).QueryRow(ctx, query, requestID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Request{}, ErrNotFound
	}
	if err != nil {
		return Request{}, fmt.Errorf("find license request: %w", err)
	}
	return item, nil
}

func (r *PostgresRepository) cancelError(ctx context.Context, requestID, requesterID string) error {
	var storedRequesterID string
	var status string
	err := database.Querier(ctx, r.pool).QueryRow(ctx, `
		SELECT requester_id::text, status
		FROM license_requests
		WHERE id = $1`, requestID).Scan(&storedRequesterID, &status)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && storedRequesterID != requesterID) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("inspect license request cancellation: %w", err)
	}
	return ErrInvalidState
}

func (r *PostgresRepository) stateChangeError(ctx context.Context, requestID string) error {
	var exists bool
	if err := database.Querier(ctx, r.pool).QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM license_requests WHERE id = $1)`, requestID).Scan(&exists); err != nil {
		return fmt.Errorf("inspect license request state: %w", err)
	}
	if !exists {
		return ErrNotFound
	}
	return ErrInvalidState
}

const requestColumns = `
	r.id::text,
	r.requester_id::text,
	r.requester_name,
	r.software_product_id::text,
	r.software_product_name,
	r.priority,
	r.reason,
	r.status,
	COALESCE(r.selected_license_id::text, ''),
	COALESCE(r.selected_license_name, ''),
	COALESCE(r.assignment_id::text, ''),
	COALESCE(r.reviewed_by::text, ''),
	COALESCE(r.reviewed_by_name, ''),
	COALESCE(r.decision_reason, ''),
	COALESCE(r.response_note, ''),
	r.created_at,
	r.updated_at,
	r.reviewed_at,
	r.cancelled_at`

const requestSelect = `SELECT ` + requestColumns + ` FROM license_requests r`

type requestScanner interface {
	Scan(...any) error
}

func scanRequest(row requestScanner) (Request, error) {
	var item Request
	err := row.Scan(
		&item.ID,
		&item.RequesterID,
		&item.RequesterName,
		&item.SoftwareProductID,
		&item.SoftwareProductName,
		&item.Priority,
		&item.Reason,
		&item.Status,
		&item.SelectedLicenseID,
		&item.SelectedLicenseName,
		&item.AssignmentID,
		&item.ReviewedBy,
		&item.ReviewedByName,
		&item.DecisionReason,
		&item.ResponseNote,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.ReviewedAt,
		&item.CancelledAt,
	)
	return item, err
}

func scanRequests(rows pgx.Rows) ([]Request, error) {
	items := make([]Request, 0)
	for rows.Next() {
		item, err := scanRequest(rows)
		if err != nil {
			return nil, fmt.Errorf("scan license request: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate license requests: %w", err)
	}
	return items, nil
}
