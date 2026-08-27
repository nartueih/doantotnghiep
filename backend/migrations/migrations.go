package migrations

import (
	"context"
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Migration struct {
	Version int
	Name    string
	SQL     string
}

type MigrationStatus struct {
	Version   int
	Name      string
	AppliedAt *time.Time
}

const LatestVersion = 4

//go:embed *.sql
var files embed.FS

func All() []Migration {
	entries, err := files.ReadDir(".")
	if err != nil {
		panic(fmt.Sprintf("read embedded migrations: %v", err))
	}

	items := make([]Migration, 0, len(entries))
	seen := make(map[int]string, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		version, err := parseVersion(entry.Name())
		if err != nil {
			panic(err)
		}
		if existing, exists := seen[version]; exists {
			panic(fmt.Sprintf("duplicate migration version %03d in %s and %s", version, existing, entry.Name()))
		}
		contents, err := files.ReadFile(entry.Name())
		if err != nil {
			panic(fmt.Sprintf("read embedded migration %s: %v", entry.Name(), err))
		}
		seen[version] = entry.Name()
		items = append(items, Migration{Version: version, Name: entry.Name(), SQL: string(contents)})
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Version < items[j].Version })
	return items
}

func parseVersion(name string) (int, error) {
	separator := strings.IndexByte(name, '_')
	if separator != 3 || !strings.HasSuffix(name, ".sql") {
		return 0, fmt.Errorf("invalid migration filename %q; expected NNN_name.sql", name)
	}
	version, err := strconv.Atoi(name[:separator])
	if err != nil || version <= 0 {
		return 0, fmt.Errorf("invalid migration version in %q", name)
	}
	return version, nil
}

const migrationAdvisoryLock int64 = 720240824

func Up(ctx context.Context, pool *pgxpool.Pool) error {
	connection, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire migration connection: %w", err)
	}
	defer connection.Release()

	if _, err := connection.Exec(ctx, "SELECT pg_advisory_lock($1)", migrationAdvisoryLock); err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = connection.Exec(unlockCtx, "SELECT pg_advisory_unlock($1)", migrationAdvisoryLock)
	}()

	if err := ensureMigrationTable(ctx, connection); err != nil {
		return err
	}

	applied, err := appliedVersions(ctx, connection)
	if err != nil {
		return err
	}

	for _, migration := range All() {
		if applied[migration.Version] {
			continue
		}

		transaction, err := connection.Begin(ctx)
		if err != nil {
			return fmt.Errorf("begin migration %03d %s: %w", migration.Version, migration.Name, err)
		}
		if _, err := transaction.Exec(ctx, migration.SQL); err != nil {
			_ = transaction.Rollback(context.Background())
			return fmt.Errorf("apply migration %03d %s: %w", migration.Version, migration.Name, err)
		}
		if _, err := transaction.Exec(ctx,
			"INSERT INTO schema_migrations (version, name) VALUES ($1, $2)",
			migration.Version,
			migration.Name,
		); err != nil {
			_ = transaction.Rollback(context.Background())
			return fmt.Errorf("apply migration %03d %s: record version: %w", migration.Version, migration.Name, err)
		}
		if err := transaction.Commit(ctx); err != nil {
			_ = transaction.Rollback(context.Background())
			return fmt.Errorf("apply migration %03d %s: commit: %w", migration.Version, migration.Name, err)
		}
	}

	return nil
}

func Status(ctx context.Context, pool *pgxpool.Pool) ([]MigrationStatus, error) {
	if err := ensureMigrationTable(ctx, pool); err != nil {
		return nil, err
	}

	rows, err := pool.Query(ctx, "SELECT version, applied_at FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("list applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[int]time.Time)
	for rows.Next() {
		var version int
		var appliedAt time.Time
		if err := rows.Scan(&version, &appliedAt); err != nil {
			return nil, fmt.Errorf("scan applied migration: %w", err)
		}
		applied[version] = appliedAt
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list applied migrations: %w", err)
	}

	migrations := All()
	statuses := make([]MigrationStatus, 0, len(migrations))
	for _, migration := range migrations {
		status := MigrationStatus{Version: migration.Version, Name: migration.Name}
		if appliedAt, exists := applied[migration.Version]; exists {
			value := appliedAt
			status.AppliedAt = &value
		}
		statuses = append(statuses, status)
	}
	return statuses, nil
}

func RequireCurrent(ctx context.Context, pool *pgxpool.Pool) error {
	var tableName *string
	if err := pool.QueryRow(ctx, "SELECT to_regclass('public.schema_migrations')::text").Scan(&tableName); err != nil {
		return fmt.Errorf("inspect migration state: %w", err)
	}

	currentVersion := 0
	if tableName != nil {
		if err := pool.QueryRow(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_migrations").Scan(&currentVersion); err != nil {
			return fmt.Errorf("inspect migration state: %w", err)
		}
	}
	return requireVersion(currentVersion)
}

func requireVersion(currentVersion int) error {
	if currentVersion == LatestVersion {
		return nil
	}
	return fmt.Errorf(
		"database schema is at version %d; expected %d; run go run ./cmd/migrate up",
		currentVersion,
		LatestVersion,
	)
}

func ensureMigrationTable(ctx context.Context, executor interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}) error {
	if _, err := executor.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		ALTER TABLE schema_migrations
			ADD COLUMN IF NOT EXISTS name TEXT NOT NULL DEFAULT '',
			ADD COLUMN IF NOT EXISTS applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
	`); err != nil {
		return fmt.Errorf("ensure migration table: %w", err)
	}

	for _, migration := range All() {
		if _, err := executor.Exec(ctx, `
			UPDATE schema_migrations
			SET name = $2
			WHERE version = $1 AND BTRIM(name) = ''
		`, migration.Version, migration.Name); err != nil {
			return fmt.Errorf("backfill migration %03d name: %w", migration.Version, err)
		}
	}
	return nil
}

func appliedVersions(ctx context.Context, connection interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}) (map[int]bool, error) {
	rows, err := connection.Query(ctx, "SELECT version FROM schema_migrations")
	if err != nil {
		return nil, fmt.Errorf("list applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[int]bool)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("scan applied migration: %w", err)
		}
		applied[version] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list applied migrations: %w", err)
	}
	return applied, nil
}
