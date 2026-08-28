package assignments

import (
	"context"
	"errors"
	"fmt"
	"license-manager/backend/internal/platform/database"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool         *pgxpool.Pool
	transactions *database.PostgresTransactor
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{
		pool:         pool,
		transactions: database.NewPostgresTransactor(pool),
	}
}

func (r *PostgresRepository) List(ctx context.Context) ([]Assignment, error) {
	rows, err := r.pool.Query(ctx, assignmentSelect+` ORDER BY a.assigned_at, a.id`)
	if err != nil {
		return nil, fmt.Errorf("list license assignments: %w", err)
	}
	defer rows.Close()

	items := make([]Assignment, 0)
	for rows.Next() {
		item, err := scanAssignment(rows)
		if err != nil {
			return nil, fmt.Errorf("scan license assignment: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate license assignments: %w", err)
	}
	return items, nil
}

func (r *PostgresRepository) Create(ctx context.Context, item Assignment) (Assignment, error) {
	var created Assignment
	err := r.transactions.WithinTransaction(ctx, func(transactionContext context.Context) error {
		query := database.Querier(transactionContext, r.pool)

		var seatCount int
		var archivedAt *time.Time
		if err := query.QueryRow(transactionContext, `SELECT seat_count, archived_at FROM licenses WHERE id = $1 FOR UPDATE`, item.LicenseID).Scan(&seatCount, &archivedAt); errors.Is(err, pgx.ErrNoRows) {
			return ErrLicenseNotFound
		} else if err != nil {
			return fmt.Errorf("lock license: %w", err)
		}
		if archivedAt != nil {
			return ErrLicenseInactive
		}

		var usedSeats int
		if err := query.QueryRow(transactionContext, `
			SELECT COUNT(*)::int FROM license_assignments
			WHERE license_id = $1 AND status = 'active'`, item.LicenseID).Scan(&usedSeats); err != nil {
			return fmt.Errorf("count license assignments: %w", err)
		}
		if usedSeats >= seatCount {
			return ErrNoAvailableSeats
		}

		_, err := query.Exec(transactionContext, `
			INSERT INTO license_assignments (
				id, license_id, user_id, device_id, assigned_by, assigned_at, status, notes
			)
			VALUES ($1, $2, $3, $4, $5, $6, 'active', NULLIF($7, ''))`,
			item.ID,
			item.LicenseID,
			nullString(item.UserID),
			nullString(item.DeviceID),
			item.AssignedBy,
			item.AssignedAt,
			item.Notes,
		)
		if err != nil {
			var postgresError *pgconn.PgError
			if errors.As(err, &postgresError) && postgresError.Code == "23505" {
				return ErrDuplicate
			}
			return fmt.Errorf("create license assignment: %w", err)
		}

		created, err = r.findByID(transactionContext, item.ID)
		return err
	})
	if err != nil {
		return Assignment{}, err
	}
	return created, nil
}

func (r *PostgresRepository) Revoke(ctx context.Context, assignmentID, actorID, _ string, revokedAt time.Time) (Assignment, error) {
	result, err := r.pool.Exec(ctx, `
		UPDATE license_assignments
		SET status = 'revoked', revoked_at = $2, revoked_by = $3
		WHERE id = $1 AND status = 'active'`, assignmentID, revokedAt, actorID)
	if err != nil {
		return Assignment{}, fmt.Errorf("revoke license assignment: %w", err)
	}
	if result.RowsAffected() != 1 {
		return Assignment{}, ErrNotFound
	}
	return r.findByID(ctx, assignmentID)
}

func (r *PostgresRepository) findByID(ctx context.Context, assignmentID string) (Assignment, error) {
	item, err := scanAssignment(database.Querier(ctx, r.pool).QueryRow(ctx, assignmentSelect+` WHERE a.id = $1`, assignmentID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Assignment{}, ErrNotFound
	}
	if err != nil {
		return Assignment{}, fmt.Errorf("find license assignment: %w", err)
	}
	return item, nil
}

const assignmentSelect = `
	SELECT a.id::text, a.license_id::text, l.name,
	       COALESCE(a.user_id::text, ''), COALESCE(a.device_id::text, ''),
	       COALESCE(target_user.full_name, target_device.asset_code, ''),
	       a.assigned_by::text, assigned_by.full_name, a.assigned_at,
	       a.revoked_at, COALESCE(a.revoked_by::text, ''),
	       COALESCE(revoked_by.full_name, ''), a.status, COALESCE(a.notes, '')
	FROM license_assignments a
	JOIN licenses l ON l.id = a.license_id
	LEFT JOIN users target_user ON target_user.id = a.user_id
	LEFT JOIN devices target_device ON target_device.id = a.device_id
	JOIN users assigned_by ON assigned_by.id = a.assigned_by
	LEFT JOIN users revoked_by ON revoked_by.id = a.revoked_by`

type scanner interface {
	Scan(...any) error
}

func scanAssignment(row scanner) (Assignment, error) {
	var item Assignment
	err := row.Scan(
		&item.ID,
		&item.LicenseID,
		&item.LicenseName,
		&item.UserID,
		&item.DeviceID,
		&item.TargetName,
		&item.AssignedBy,
		&item.AssignedByName,
		&item.AssignedAt,
		&item.RevokedAt,
		&item.RevokedBy,
		&item.RevokedByName,
		&item.Status,
		&item.Notes,
	)
	return item, err
}

func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
