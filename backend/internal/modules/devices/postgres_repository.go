package devices

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

func (r *PostgresRepository) List(ctx context.Context) ([]Device, error) {
	rows, err := r.pool.Query(ctx, deviceSelect+` ORDER BY d.asset_code, d.id`)
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	defer rows.Close()

	items := make([]Device, 0)
	for rows.Next() {
		item, err := scanDevice(rows)
		if err != nil {
			return nil, fmt.Errorf("scan device: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate devices: %w", err)
	}
	return items, nil
}

func (r *PostgresRepository) FindByID(ctx context.Context, deviceID string) (Device, error) {
	item, err := scanDevice(r.pool.QueryRow(ctx, deviceSelect+` WHERE d.id = $1`, deviceID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Device{}, ErrNotFound
	}
	if err != nil {
		return Device{}, fmt.Errorf("find device: %w", err)
	}
	return item, nil
}

func (r *PostgresRepository) Create(ctx context.Context, item Device) (Device, error) {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO devices (
			id, asset_code, serial_number, name, device_type, manufacturer,
			model, status, purchased_at, warranty_expires_at
		)
		VALUES (
			$1, $2, NULLIF($3, ''), $4, $5, NULLIF($6, ''), NULLIF($7, ''),
			$8, NULLIF($9, '')::date, NULLIF($10, '')::date
		)`,
		item.ID,
		item.AssetCode,
		item.SerialNumber,
		item.Name,
		item.DeviceType,
		item.Manufacturer,
		item.Model,
		item.Status,
		item.PurchasedAt,
		item.WarrantyExpiresAt,
	)
	if err != nil {
		return Device{}, mapWriteError("create device", err)
	}
	return r.FindByID(ctx, item.ID)
}

func (r *PostgresRepository) Update(ctx context.Context, item Device) (Device, error) {
	result, err := r.pool.Exec(ctx, `
		UPDATE devices
		SET asset_code = $2, serial_number = NULLIF($3, ''), name = $4,
		    device_type = $5, manufacturer = NULLIF($6, ''), model = NULLIF($7, ''),
		    purchased_at = NULLIF($8, '')::date,
		    warranty_expires_at = NULLIF($9, '')::date, updated_at = NOW()
		WHERE id = $1`,
		item.ID,
		item.AssetCode,
		item.SerialNumber,
		item.Name,
		item.DeviceType,
		item.Manufacturer,
		item.Model,
		item.PurchasedAt,
		item.WarrantyExpiresAt,
	)
	if err != nil {
		return Device{}, mapWriteError("update device", err)
	}
	if result.RowsAffected() != 1 {
		return Device{}, ErrNotFound
	}
	return r.FindByID(ctx, item.ID)
}

func (r *PostgresRepository) UpdateStatus(ctx context.Context, deviceID, status string) (Device, error) {
	result, err := r.pool.Exec(ctx, `
		UPDATE devices SET status = $2, updated_at = NOW() WHERE id = $1`, deviceID, status)
	if err != nil {
		return Device{}, fmt.Errorf("update device status: %w", err)
	}
	if result.RowsAffected() != 1 {
		return Device{}, ErrNotFound
	}
	return r.FindByID(ctx, deviceID)
}

func (r *PostgresRepository) Assign(ctx context.Context, deviceID, userID, _ string) (Device, error) {
	var result pgconn.CommandTag
	var err error
	if userID == "" {
		result, err = r.pool.Exec(ctx, `
			UPDATE devices
			SET assigned_user_id = NULL, status = 'available', updated_at = NOW()
			WHERE id = $1`, deviceID)
	} else {
		result, err = r.pool.Exec(ctx, `
			UPDATE devices
			SET assigned_user_id = $2, status = 'assigned', updated_at = NOW()
			WHERE id = $1 AND status = 'available' AND assigned_user_id IS NULL`, deviceID, userID)
	}
	if err != nil {
		return Device{}, fmt.Errorf("assign device: %w", err)
	}
	if result.RowsAffected() != 1 {
		return Device{}, ErrDeviceUnavailable
	}
	return r.FindByID(ctx, deviceID)
}

const deviceSelect = `
	SELECT d.id::text, COALESCE(d.assigned_user_id::text, ''), COALESCE(u.full_name, ''),
	       d.asset_code, COALESCE(d.serial_number, ''), d.name, d.device_type,
	       COALESCE(d.manufacturer, ''), COALESCE(d.model, ''), d.status,
	       COALESCE(d.purchased_at::text, ''), COALESCE(d.warranty_expires_at::text, ''),
	       d.created_at, d.updated_at
	FROM devices d
	LEFT JOIN users u ON u.id = d.assigned_user_id`

type scanner interface {
	Scan(...any) error
}

func scanDevice(row scanner) (Device, error) {
	var item Device
	err := row.Scan(
		&item.ID,
		&item.AssignedUserID,
		&item.AssignedUserName,
		&item.AssetCode,
		&item.SerialNumber,
		&item.Name,
		&item.DeviceType,
		&item.Manufacturer,
		&item.Model,
		&item.Status,
		&item.PurchasedAt,
		&item.WarrantyExpiresAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	return item, err
}

func mapWriteError(operation string, err error) error {
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) && postgresError.Code == "23505" {
		switch postgresError.ConstraintName {
		case "uq_devices_asset_code":
			return ErrAssetCodeExists
		case "uq_devices_serial_number":
			return ErrSerialExists
		}
	}
	return fmt.Errorf("%s: %w", operation, err)
}
