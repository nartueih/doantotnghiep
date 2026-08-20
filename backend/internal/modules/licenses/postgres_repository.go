package licenses

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) List(ctx context.Context) ([]License, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT l.id::text, l.software_product_id::text, l.name, l.license_type,
		       l.assignment_type, l.seat_count, COUNT(a.id)::int,
		       l.encrypted_key, COALESCE(l.key_hint, ''), l.allow_employee_key_view,
		       COALESCE(l.vendor, ''),
		       COALESCE(l.purchased_at::text, ''), COALESCE(l.starts_at::text, ''),
		       COALESCE(l.expires_at::text, ''), COALESCE(l.cost, 0)::float8,
		       COALESCE(l.currency, ''), COALESCE(l.notes, ''), l.created_at, l.updated_at,
		       l.archived_at
		FROM licenses l
		LEFT JOIN license_assignments a ON a.license_id = l.id AND a.status = 'active'
		GROUP BY l.id
		ORDER BY l.name, l.id`)
	if err != nil {
		return nil, fmt.Errorf("list licenses: %w", err)
	}
	defer rows.Close()

	items := make([]License, 0)
	for rows.Next() {
		item, err := scanLicense(rows)
		if err != nil {
			return nil, fmt.Errorf("scan license: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate licenses: %w", err)
	}
	return items, nil
}

func (r *PostgresRepository) FindByID(ctx context.Context, licenseID string) (License, error) {
	item, err := scanLicense(r.pool.QueryRow(ctx, `
		SELECT l.id::text, l.software_product_id::text, l.name, l.license_type,
		       l.assignment_type, l.seat_count, COUNT(a.id)::int,
		       l.encrypted_key, COALESCE(l.key_hint, ''), l.allow_employee_key_view,
		       COALESCE(l.vendor, ''),
		       COALESCE(l.purchased_at::text, ''), COALESCE(l.starts_at::text, ''),
		       COALESCE(l.expires_at::text, ''), COALESCE(l.cost, 0)::float8,
		       COALESCE(l.currency, ''), COALESCE(l.notes, ''), l.created_at, l.updated_at,
		       l.archived_at
		FROM licenses l
		LEFT JOIN license_assignments a ON a.license_id = l.id AND a.status = 'active'
		WHERE l.id = $1
		GROUP BY l.id`, licenseID))
	if errors.Is(err, pgx.ErrNoRows) {
		return License{}, ErrNotFound
	}
	if err != nil {
		return License{}, fmt.Errorf("find license: %w", err)
	}
	return item, nil
}

func (r *PostgresRepository) Create(ctx context.Context, item License) (License, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO licenses (
			id, software_product_id, name, license_type, assignment_type, seat_count,
			encrypted_key, key_hint, allow_employee_key_view, vendor, purchased_at, starts_at, expires_at,
			cost, currency, notes
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, NULLIF($8, ''), $9, NULLIF($10, ''),
			NULLIF($11, '')::date, NULLIF($12, '')::date, NULLIF($13, '')::date,
			$14, NULLIF($15, ''), NULLIF($16, '')
		)
		RETURNING id::text, software_product_id::text, name, license_type,
		          assignment_type, seat_count, 0::int, encrypted_key,
		          COALESCE(key_hint, ''), allow_employee_key_view, COALESCE(vendor, ''),
		          COALESCE(purchased_at::text, ''), COALESCE(starts_at::text, ''),
		          COALESCE(expires_at::text, ''), COALESCE(cost, 0)::float8,
		          COALESCE(currency, ''), COALESCE(notes, ''), created_at, updated_at,
		          archived_at`,
		item.ID,
		item.SoftwareProductID,
		item.Name,
		item.LicenseType,
		item.AssignmentType,
		item.SeatCount,
		nullBytes(item.EncryptedKey),
		item.KeyHint,
		item.AllowEmployeeKeyView,
		item.Vendor,
		item.PurchasedAt,
		item.StartsAt,
		item.ExpiresAt,
		item.Cost,
		item.Currency,
		item.Notes,
	)
	created, err := scanLicense(row)
	if err != nil {
		return License{}, fmt.Errorf("create license: %w", err)
	}
	return created, nil
}

func (r *PostgresRepository) Update(ctx context.Context, item License) (License, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return License{}, fmt.Errorf("begin license update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var usedSeats int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM license_assignments
		WHERE license_id = $1 AND status = 'active'`, item.ID).Scan(&usedSeats); err != nil {
		return License{}, fmt.Errorf("count active assignments: %w", err)
	}
	if item.SeatCount < usedSeats {
		return License{}, ErrSeatCountBelowUsage
	}

	row := tx.QueryRow(ctx, `
		UPDATE licenses
		SET software_product_id = $2, name = $3, license_type = $4,
		    assignment_type = $5, seat_count = $6,
		    encrypted_key = COALESCE($7, encrypted_key),
		    key_hint = CASE WHEN $7 IS NULL THEN key_hint ELSE NULLIF($8, '') END,
		    allow_employee_key_view = $9,
		    vendor = NULLIF($10, ''), purchased_at = NULLIF($11, '')::date,
		    starts_at = NULLIF($12, '')::date, expires_at = NULLIF($13, '')::date,
		    cost = $14, currency = NULLIF($15, ''), notes = NULLIF($16, ''),
		    updated_at = NOW()
		WHERE id = $1 AND archived_at IS NULL
		RETURNING id::text, software_product_id::text, name, license_type,
		          assignment_type, seat_count, $17::int, encrypted_key,
		          COALESCE(key_hint, ''), allow_employee_key_view, COALESCE(vendor, ''),
		          COALESCE(purchased_at::text, ''), COALESCE(starts_at::text, ''),
		          COALESCE(expires_at::text, ''), COALESCE(cost, 0)::float8,
		          COALESCE(currency, ''), COALESCE(notes, ''), created_at, updated_at,
		          archived_at`,
		item.ID,
		item.SoftwareProductID,
		item.Name,
		item.LicenseType,
		item.AssignmentType,
		item.SeatCount,
		nullBytes(item.EncryptedKey),
		item.KeyHint,
		item.AllowEmployeeKeyView,
		item.Vendor,
		item.PurchasedAt,
		item.StartsAt,
		item.ExpiresAt,
		item.Cost,
		item.Currency,
		item.Notes,
		usedSeats,
	)
	updated, err := scanLicense(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return License{}, ErrArchived
	}
	if err != nil {
		return License{}, fmt.Errorf("update license: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return License{}, fmt.Errorf("commit license update: %w", err)
	}
	return updated, nil
}

func (r *PostgresRepository) Archive(ctx context.Context, licenseID string, archivedAt time.Time) (License, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return License{}, fmt.Errorf("begin license archive: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var currentArchivedAt *time.Time
	if err := tx.QueryRow(ctx, `
		SELECT archived_at
		FROM licenses
		WHERE id = $1
		FOR UPDATE`, licenseID).Scan(&currentArchivedAt); errors.Is(err, pgx.ErrNoRows) {
		return License{}, ErrNotFound
	} else if err != nil {
		return License{}, fmt.Errorf("lock license for archive: %w", err)
	}
	if currentArchivedAt != nil {
		return License{}, ErrAlreadyArchived
	}

	var usedSeats int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)::int
		FROM license_assignments
		WHERE license_id = $1 AND status = 'active'`, licenseID).Scan(&usedSeats); err != nil {
		return License{}, fmt.Errorf("count active assignments before archive: %w", err)
	}
	if usedSeats > 0 {
		return License{}, ErrActiveAssignments
	}

	if _, err := tx.Exec(ctx, `
		UPDATE licenses
		SET archived_at = $2, updated_at = $2
		WHERE id = $1`, licenseID, archivedAt.UTC()); err != nil {
		return License{}, fmt.Errorf("archive license: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return License{}, fmt.Errorf("commit license archive: %w", err)
	}
	return r.FindByID(ctx, licenseID)
}

type scanner interface {
	Scan(...any) error
}

func scanLicense(row scanner) (License, error) {
	var item License
	err := row.Scan(
		&item.ID,
		&item.SoftwareProductID,
		&item.Name,
		&item.LicenseType,
		&item.AssignmentType,
		&item.SeatCount,
		&item.UsedSeats,
		&item.EncryptedKey,
		&item.KeyHint,
		&item.AllowEmployeeKeyView,
		&item.Vendor,
		&item.PurchasedAt,
		&item.StartsAt,
		&item.ExpiresAt,
		&item.Cost,
		&item.Currency,
		&item.Notes,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.ArchivedAt,
	)
	return item, err
}

func nullBytes(value []byte) any {
	if len(value) == 0 {
		return nil
	}
	return value
}
