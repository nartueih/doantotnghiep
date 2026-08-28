# PostgreSQL Persistence Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Persist every existing License Manager workflow in PostgreSQL, including atomic license-request decisions, repeatable migrations, Admin bootstrap, and restart-safe verification.

**Architecture:** Keep the current repository interfaces and `pgx/v5` data-access style. Add an embedded, versioned migration runner; PostgreSQL repositories for license requests and notifications; and a context-scoped transaction manager that lets request, assignment, and notification repositories participate in one transaction while memory tests keep using a direct manager.

**Tech Stack:** Go 1.25, PostgreSQL 18, `pgx/v5`, PowerShell, Go `embed`, existing Gin HTTP API and Node/Vite web verification.

**Spec:** `docs/superpowers/specs/2026-08-24-postgresql-persistence-design.md`

## Global Constraints

- PostgreSQL is accessed only by the Go backend; web and Android clients use REST API only.
- Keep memory repositories for fast unit tests and development fallback.
- Do not introduce GORM or another ORM.
- Use `STORAGE_DRIVER=postgres` and `DATABASE_URL` for runtime; use a separate `TEST_DATABASE_URL` for integration tests.
- Use `sslmode=disable` only for local development.
- Do not commit database passwords, JWT secrets, or license encryption keys.
- Keep activation keys encrypted and never copy them into notifications, logs, tests, or audit metadata.
- Every state-changing SQL statement uses parameters and validates affected-row counts.
- Every implementation task follows red-green-refactor and ends with a focused commit.

---

### Task 1: Versioned migration runner and PostgreSQL schema

**Files:**
- Create: `backend/migrations/migrations.go`
- Create: `backend/migrations/migrations_test.go`
- Create: `backend/migrations/004_license_requests_and_notifications.sql`
- Create: `backend/cmd/migrate/main.go`
- Test: `backend/migrations/migrations_test.go`

**Interfaces:**
- Consumes: `database.Open(ctx, databaseURL) (*pgxpool.Pool, error)` from `backend/internal/platform/database/postgres.go`.
- Produces: `migrations.All() []Migration`, `migrations.Up(context.Context, *pgxpool.Pool) error`, `migrations.Status(context.Context, *pgxpool.Pool) ([]MigrationStatus, error)`, `migrations.RequireCurrent(context.Context, *pgxpool.Pool) error`, and `migrations.LatestVersion = 4`.

- [ ] **Step 1: Write the failing migration manifest tests**

```go
func TestAllReturnsOrderedUniqueMigrations(t *testing.T) {
	items := All()
	if len(items) != 4 {
		t.Fatalf("expected 4 migrations, got %d", len(items))
	}
	for index, item := range items {
		expected := index + 1
		if item.Version != expected || strings.TrimSpace(item.SQL) == "" {
			t.Fatalf("migration %d is invalid: %#v", expected, item)
		}
	}
	if LatestVersion != items[len(items)-1].Version {
		t.Fatalf("latest version=%d, last migration=%d", LatestVersion, items[len(items)-1].Version)
	}
}

func TestParseVersionRejectsInvalidFilename(t *testing.T) {
	if _, err := parseVersion("license_requests.sql"); err == nil {
		t.Fatal("expected invalid migration filename")
	}
}
```

- [ ] **Step 2: Run the manifest test and verify red**

Run: `Set-Location backend; go test ./migrations -run 'TestAll|TestParseVersion' -count=1`

Expected: FAIL because package functions and migration `004` do not exist.

- [ ] **Step 3: Add the embedded migration manifest and runner**

Implement these public types and constants in `migrations.go`:

```go
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
```

`All` must read `*.sql`, parse the three-digit prefix, reject duplicate versions, and return ascending order. `Up` must:

1. obtain `pg_advisory_lock(720240824)` and defer `pg_advisory_unlock(720240824)` on the same acquired connection;
2. create `schema_migrations(version INTEGER PRIMARY KEY, name TEXT NOT NULL, applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`;
3. load applied versions;
4. execute each missing migration in its own `pgx.Tx`;
5. insert the version in `schema_migrations` before committing;
6. rollback and return `apply migration %03d %s: %w` on error.

`Status` returns every embedded migration with `AppliedAt=nil` when missing. `RequireCurrent` reads `MAX(version)` and returns `database schema is at version %d; expected %d; run go run ./cmd/migrate up` unless it equals `LatestVersion`.

The CLI accepts exactly `up` or `status`, requires `DATABASE_URL`, opens a 15-second startup context, prints one status line per migration, and exits nonzero on error.

- [ ] **Step 4: Add migration 004 with constraints and indexes**

Use the following table and index structure; keep every name stable because repositories map specific constraints:

```sql
CREATE TABLE license_requests (
    id UUID PRIMARY KEY,
    requester_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    requester_name VARCHAR(150) NOT NULL,
    software_product_id UUID NOT NULL REFERENCES software_products(id) ON DELETE RESTRICT,
    software_product_name VARCHAR(150) NOT NULL,
    priority VARCHAR(20) NOT NULL CHECK (priority IN ('normal', 'high', 'urgent')),
    reason TEXT NOT NULL CHECK (BTRIM(reason) <> ''),
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'approved', 'rejected', 'cancelled')),
    selected_license_id UUID REFERENCES licenses(id) ON DELETE RESTRICT,
    selected_license_name VARCHAR(150),
    assignment_id UUID REFERENCES license_assignments(id) ON DELETE RESTRICT,
    reviewed_by UUID REFERENCES users(id) ON DELETE RESTRICT,
    reviewed_by_name VARCHAR(150),
    decision_reason VARCHAR(30) CHECK (decision_reason IS NULL OR decision_reason IN ('out_of_stock', 'not_approved', 'other')),
    response_note TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    reviewed_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    CONSTRAINT ck_license_requests_state CHECK (
      (status = 'pending' AND selected_license_id IS NULL AND assignment_id IS NULL AND reviewed_by IS NULL AND reviewed_at IS NULL AND cancelled_at IS NULL)
      OR (status = 'approved' AND selected_license_id IS NOT NULL AND assignment_id IS NOT NULL AND reviewed_by IS NOT NULL AND reviewed_at IS NOT NULL AND cancelled_at IS NULL)
      OR (status = 'rejected' AND selected_license_id IS NULL AND assignment_id IS NULL AND reviewed_by IS NOT NULL AND decision_reason IS NOT NULL AND BTRIM(COALESCE(response_note, '')) <> '' AND reviewed_at IS NOT NULL AND cancelled_at IS NULL)
      OR (status = 'cancelled' AND selected_license_id IS NULL AND assignment_id IS NULL AND reviewed_by IS NULL AND reviewed_at IS NULL AND cancelled_at IS NOT NULL)
    )
);

CREATE UNIQUE INDEX uq_pending_license_request
    ON license_requests (requester_id, software_product_id)
    WHERE status = 'pending';
CREATE INDEX idx_license_requests_requester_created
    ON license_requests (requester_id, created_at DESC);
CREATE INDEX idx_license_requests_status_created
    ON license_requests (status, created_at DESC);
CREATE INDEX idx_license_requests_priority
    ON license_requests (priority);

CREATE TABLE notifications (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type VARCHAR(80) NOT NULL,
    title VARCHAR(200) NOT NULL CHECK (BTRIM(title) <> ''),
    message TEXT NOT NULL CHECK (BTRIM(message) <> ''),
    entity_type VARCHAR(80) NOT NULL,
    entity_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    read_at TIMESTAMPTZ
);

CREATE INDEX idx_notifications_user_created
    ON notifications (user_id, created_at DESC);
CREATE INDEX idx_notifications_user_unread
    ON notifications (user_id, created_at DESC)
    WHERE read_at IS NULL;
```

- [ ] **Step 5: Run migration unit tests and compile the CLI**

Run: `Set-Location backend; go test ./migrations ./cmd/migrate -count=1`

Expected: PASS; `cmd/migrate` reports `[no test files]` after compiling.

- [ ] **Step 6: Create the local application role and two databases**

Run PowerShell, then enter the PostgreSQL superuser password when prompted:

```powershell
psql -U postgres -h localhost -d postgres
```

Run these commands inside `psql`. `\password` prompts twice without storing the password in shell history:

```sql
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'license_admin') THEN
        CREATE ROLE license_admin LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE;
    END IF;
END
$$;
\password license_admin
SELECT 'CREATE DATABASE license_manager OWNER license_admin'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'license_manager')\gexec
SELECT 'CREATE DATABASE license_manager_test OWNER license_admin'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'license_manager_test')\gexec
\q
```

- [ ] **Step 7: Apply and verify all migrations locally**

```powershell
$databasePassword = Read-Host "Nhập password của license_admin"
$encodedPassword = [Uri]::EscapeDataString($databasePassword)
$env:DATABASE_URL = "postgres://license_admin:$encodedPassword@localhost:5432/license_manager?sslmode=disable"
$env:TEST_DATABASE_URL = "postgres://license_admin:$encodedPassword@localhost:5432/license_manager_test?sslmode=disable"
Set-Location "D:\Đồ án\backend"
go run ./cmd/migrate up
go run ./cmd/migrate status
psql $env:DATABASE_URL -c "SELECT version, name FROM schema_migrations ORDER BY version;"
```

Expected: versions `1`, `2`, `3`, `4` appear exactly once and both new tables exist.

- [ ] **Step 8: Commit migration infrastructure**

```powershell
git add backend/migrations backend/cmd/migrate
git commit -m "feat: add versioned PostgreSQL migrations"
```

---

### Task 2: Context-scoped PostgreSQL transactions

**Files:**
- Create: `backend/internal/platform/database/transaction.go`
- Create: `backend/internal/platform/database/transaction_test.go`
- Modify: `backend/internal/modules/auth/postgres_repository.go`
- Modify: `backend/internal/modules/licenses/postgres_repository.go`
- Modify: `backend/internal/modules/assignments/postgres_repository.go`
- Test: `backend/internal/platform/database/transaction_test.go`
- Test: `backend/internal/modules/assignments/service_test.go`

**Interfaces:**
- Consumes: `pgxpool.Pool`, `pgx.Tx`, and existing repository methods.
- Produces: `database.Transactor`, `database.DirectTransactor`, `database.NewPostgresTransactor(*pgxpool.Pool) Transactor`, `database.Querier(context.Context, *pgxpool.Pool) DBTX`.

- [ ] **Step 1: Write failing direct-transactor tests**

```go
func TestDirectTransactorRunsCallbackAndReturnsItsError(t *testing.T) {
	expected := errors.New("stop")
	called := false
	err := (DirectTransactor{}).WithinTransaction(t.Context(), func(context.Context) error {
		called = true
		return expected
	})
	if !called || !errors.Is(err, expected) {
		t.Fatalf("called=%v err=%v", called, err)
	}
}
```

- [ ] **Step 2: Run the transaction test and verify red**

Run: `Set-Location backend; go test ./internal/platform/database -run TestDirectTransactor -count=1`

Expected: FAIL because `DirectTransactor` does not exist.

- [ ] **Step 3: Implement the transaction interfaces**

```go
type DBTX interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type Transactor interface {
	WithinTransaction(context.Context, func(context.Context) error) error
}

type DirectTransactor struct{}

func (DirectTransactor) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
```

`PostgresTransactor.WithinTransaction` must reuse a transaction already stored in context; otherwise begin a transaction, call the callback with a private context key, rollback on callback/commit failure, and commit exactly once on success. `Querier` returns the transaction from context or the supplied pool.

- [ ] **Step 4: Make reads and assignment creation transaction-aware**

Replace direct `r.pool.QueryRow` calls in `auth.findUser` and `licenses.FindByID` with `database.Querier(ctx, r.pool).QueryRow`.

Refactor `assignments.PostgresRepository.Create` to call its `PostgresTransactor.WithinTransaction`. Inside the callback, use one `DBTX` for `SELECT ... FOR UPDATE`, seat count, insert, and `findByID`. Do not call `tx.Commit` inside the repository. When no outer transaction exists, `WithinTransaction` creates one and preserves the existing standalone behavior.

- [ ] **Step 5: Run focused and complete Go tests**

Run:

```powershell
Set-Location backend
go test ./internal/platform/database ./internal/modules/auth ./internal/modules/licenses ./internal/modules/assignments -count=1
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit transaction foundation**

```powershell
git add backend/internal/platform/database backend/internal/modules/auth/postgres_repository.go backend/internal/modules/licenses/postgres_repository.go backend/internal/modules/assignments/postgres_repository.go
git commit -m "refactor: support shared PostgreSQL transactions"
```

---

### Task 3: PostgreSQL notification repository

**Files:**
- Create: `backend/internal/testsupport/postgres.go`
- Create: `backend/internal/modules/notifications/postgres_repository.go`
- Create: `backend/internal/modules/notifications/postgres_repository_test.go`
- Test: `backend/internal/modules/notifications/postgres_repository_test.go`

**Interfaces:**
- Consumes: `database.Querier`, `migrations.Up`, `TEST_DATABASE_URL`, and `notifications.Repository`.
- Produces: `notifications.NewPostgresRepository(*pgxpool.Pool) *PostgresRepository` and reusable `testsupport.OpenPostgres(t testing.TB) *pgxpool.Pool`.

- [ ] **Step 1: Write failing PostgreSQL repository tests**

Create tests guarded by `TEST_DATABASE_URL`. Seed two users with fixed UUIDs and verify:

```go
func TestPostgresRepositoryScopesAndMarksNotifications(t *testing.T) {
	pool := testsupport.OpenPostgres(t)
	repository := NewPostgresRepository(pool)
	const userOneID = "20000000-0000-0000-0000-000000000001"
	const userTwoID = "20000000-0000-0000-0000-000000000002"
	now := time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC)
	first := Notification{
		ID: "10000000-0000-0000-0000-000000000001", UserID: userOneID,
		Type: TypeLicenseRequestApproved, Title: "Đã duyệt", Message: "Đã cấp license",
		EntityType: EntityLicenseRequest, EntityID: "30000000-0000-0000-0000-000000000001", CreatedAt: now,
	}
	second := Notification{
		ID: "10000000-0000-0000-0000-000000000002", UserID: userTwoID,
		Type: TypeLicenseRequestRejected, Title: "Đã phản hồi", Message: "Tạm hết license",
		EntityType: EntityLicenseRequest, EntityID: "30000000-0000-0000-0000-000000000002", CreatedAt: now.Add(time.Minute),
	}
	if _, err := repository.Create(t.Context(), first); err != nil { t.Fatal(err) }
	if _, err := repository.Create(t.Context(), second); err != nil { t.Fatal(err) }
	items, err := repository.ListByUser(t.Context(), userOneID)
	if err != nil || len(items) != 1 || items[0].ID != first.ID { t.Fatalf("items=%#v err=%v", items, err) }
	read, err := repository.MarkRead(t.Context(), userOneID, first.ID, time.Now().UTC())
	if err != nil || read.ReadAt == nil { t.Fatalf("read=%#v err=%v", read, err) }
	if _, err := repository.MarkRead(t.Context(), userTwoID, first.ID, time.Now().UTC()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ownership-safe not found, got %v", err)
	}
}
```

Add a separate test that creates three unread notifications, marks all for user one, expects count `2`, and confirms user two remains unread.

- [ ] **Step 2: Run the notification integration tests and verify red**

Run: `Set-Location backend; go test ./internal/modules/notifications -run Postgres -count=1`

Expected: FAIL because `NewPostgresRepository` and test support do not exist.

- [ ] **Step 3: Add reusable PostgreSQL test support**

`OpenPostgres` must:

1. read `TEST_DATABASE_URL` and call `t.Skip` when empty;
2. open a pool and call `migrations.Up`;
3. truncate all application tables except `schema_migrations` using one explicit `TRUNCATE ... RESTART IDENTITY CASCADE` statement;
4. register `pool.Close` with `t.Cleanup`;
5. never point at `license_manager`; reject a URL whose database name does not end in `_test`.

- [ ] **Step 4: Implement the notification queries**

Use `database.Querier(ctx, r.pool)` for every query. Required SQL behavior:

```sql
SELECT id::text, user_id::text, type, title, message, entity_type,
       entity_id::text, created_at, read_at
FROM notifications
WHERE user_id = $1
ORDER BY created_at DESC, id DESC;

UPDATE notifications
SET read_at = COALESCE(read_at, $3)
WHERE id = $1 AND user_id = $2
RETURNING id::text, user_id::text, type, title, message, entity_type,
          entity_id::text, created_at, read_at;

UPDATE notifications
SET read_at = $2
WHERE user_id = $1 AND read_at IS NULL;
```

Map no returned row to `ErrNotFound`; wrap operational errors as `list notifications`, `create notification`, `mark notification read`, or `mark all notifications read`.

- [ ] **Step 5: Run repository and service tests**

Run:

```powershell
Set-Location backend
go test ./internal/modules/notifications -count=1
go test ./... -count=1
```

Expected: PASS with integration tests executed when `TEST_DATABASE_URL` is set.

- [ ] **Step 6: Commit notification persistence**

```powershell
git add backend/internal/testsupport backend/internal/modules/notifications
git commit -m "feat: persist website notifications in PostgreSQL"
```

---

### Task 4: PostgreSQL license-request repository

**Files:**
- Modify: `backend/internal/modules/licenserequests/model.go`
- Modify: `backend/internal/modules/licenserequests/memory_repository.go`
- Create: `backend/internal/modules/licenserequests/postgres_repository.go`
- Create: `backend/internal/modules/licenserequests/postgres_repository_test.go`
- Test: `backend/internal/modules/licenserequests/postgres_repository_test.go`

**Interfaces:**
- Consumes: `database.Querier`, `testsupport.OpenPostgres`, and migration 004.
- Produces: `Repository.FindForUpdate(context.Context, string) (Request, error)` and `licenserequests.NewPostgresRepository(*pgxpool.Pool) *PostgresRepository`.

- [ ] **Step 1: Add failing repository contract and integration tests**

Extend `Repository` with:

```go
FindForUpdate(context.Context, string) (Request, error)
```

Add `MemoryRepository.FindForUpdate` as an alias to `FindByID` so existing service tests compile. Then write PostgreSQL tests for create/list/search, duplicate pending requests, owner-only cancel, approve/reject state transitions, and terminal-state conflicts.

The duplicate test must execute two goroutines with the same `(requester_id, software_product_id)` and assert one success plus one `ErrPendingDuplicate`.

- [ ] **Step 2: Run focused tests and verify red**

Run: `Set-Location backend; go test ./internal/modules/licenserequests -run Postgres -count=1`

Expected: FAIL because the PostgreSQL repository does not exist.

- [ ] **Step 3: Implement request scanning and list queries**

Define one `requestSelect` constant returning fields in the exact `Request` struct order. Use stored snapshot names and `COALESCE` for nullable strings. Implement filters as parameters:

```sql
WHERE ($1 = '' OR status = $1)
  AND ($2 = '' OR priority = $2)
  AND ($3 = '' OR requester_name ILIKE '%' || $3 || '%'
               OR software_product_name ILIKE '%' || $3 || '%')
ORDER BY created_at DESC, id DESC;
```

`FindForUpdate` appends `WHERE id = $1 FOR UPDATE` and uses `database.Querier`, so it locks only when a transaction is present.

- [ ] **Step 4: Implement state changes and error mapping**

`Create` inserts every required snapshot and timestamp field. Map PostgreSQL error code `23505` with constraint `uq_pending_license_request` to `ErrPendingDuplicate`.

Use conditional updates:

```sql
UPDATE license_requests
SET status = 'cancelled', cancelled_at = $3, updated_at = $3
WHERE id = $1 AND requester_id = $2 AND status = 'pending';
```

Approve and reject must update only `status='pending'`, set exactly the fields allowed by `ck_license_requests_state`, and return the updated row. If the ID exists but the conditional update affects zero rows, return `ErrInvalidState`; if the ID/owner does not exist, return `ErrNotFound`.

- [ ] **Step 5: Run repository, module, and full Go tests**

Run:

```powershell
Set-Location backend
go test ./internal/modules/licenserequests -count=1
go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit license-request persistence**

```powershell
git add backend/internal/modules/licenserequests
git commit -m "feat: persist license requests in PostgreSQL"
```

---

### Task 5: Atomic request decisions and PostgreSQL runtime wiring

**Files:**
- Modify: `backend/internal/modules/licenserequests/model.go`
- Modify: `backend/internal/modules/licenserequests/service.go`
- Modify: `backend/internal/modules/licenserequests/service_test.go`
- Create: `backend/internal/modules/licenserequests/postgres_workflow_test.go`
- Modify: `backend/cmd/api/main.go`
- Modify: `backend/internal/httpapi/router_test.go`
- Test: `backend/internal/modules/licenserequests/postgres_workflow_test.go`

**Interfaces:**
- Consumes: `database.Transactor`, PostgreSQL repositories from Tasks 3-4, and transaction-aware assignment/auth/license repositories from Task 2.
- Produces: `licenserequests.TransactionManager` compatibility through `database.Transactor` injection and fully populated handlers for both storage drivers.

- [ ] **Step 1: Write failing service transaction tests**

Change `NewService` to require the final argument `transactions TransactionManager`, where:

```go
type TransactionManager interface {
	WithinTransaction(context.Context, func(context.Context) error) error
}
```

Update memory fixtures to pass `database.DirectTransactor{}`. Add a recording transaction fake and assert `Approve` and `Reject` each call `WithinTransaction` exactly once while validation errors before a decision do not open a transaction.

- [ ] **Step 2: Write failing PostgreSQL rollback and concurrency tests**

Build a PostgreSQL fixture using the test database and real repositories. Add:

- `TestPostgresApproveRollsBackAssignmentWhenNotificationFails`: wrap notification repository so `Create` returns `errors.New("notification unavailable")`; expect request still pending, zero assignments, zero notifications.
- `TestPostgresConcurrentApproveCreatesExactlyOneAssignment`: start two goroutines against the same request; expect one success, one `ErrInvalidState`, one assignment, one approved notification.
- `TestPostgresRejectCommitsRequestAndNotificationTogether`: expect one rejected request and one notification after reopening a new pool.

- [ ] **Step 3: Run focused tests and verify red**

Run: `Set-Location backend; go test ./internal/modules/licenserequests -run 'Transaction|PostgresApprove|PostgresConcurrent|PostgresReject' -count=1`

Expected: FAIL because the service does not use the transaction manager.

- [ ] **Step 4: Wrap decision workflows in one transaction**

Keep input validation outside the transaction. Inside `WithinTransaction`:

```go
var approved Request
err := s.transactions.WithinTransaction(ctx, func(txCtx context.Context) error {
	item, err := s.pendingRequestForUpdate(txCtx, requestID)
	if err != nil { return err }
	reviewer, err := s.reviewer(txCtx, reviewerID)
	if err != nil { return err }
	license, err := s.licenses.FindByID(txCtx, input.LicenseID)
	if errors.Is(err, licenses.ErrNotFound) { return ErrLicenseNotFound }
	if err != nil { return err }
	if license.SoftwareProductID != item.SoftwareProductID { return ErrLicenseProductMismatch }
	assignment, err := s.assignments.Create(txCtx, reviewer.ID, assignments.CreateInput{
		LicenseID: license.ID, UserID: item.RequesterID,
		Notes: fmt.Sprintf("Cấp từ yêu cầu license %s", item.ID),
	})
	if err != nil { return err }
	now := s.now().UTC()
	updated, err := s.repository.Approve(txCtx, ApprovalUpdate{
		RequestID: item.ID, LicenseID: license.ID, LicenseName: license.Name,
		AssignmentID: assignment.ID, ReviewerID: reviewer.ID, ReviewerName: reviewer.FullName,
		ResponseNote: input.ResponseNote, ReviewedAt: now,
	})
	if err != nil { return err }
	message := input.ResponseNote
	if message == "" { message = fmt.Sprintf("Yêu cầu %s đã được duyệt và license đã được cấp cho bạn.", item.SoftwareProductName) }
	if _, err := s.notifications.Create(txCtx, notifications.CreateInput{
		UserID: item.RequesterID, Type: notifications.TypeLicenseRequestApproved,
		Title: "Yêu cầu license đã được duyệt", Message: message,
		EntityType: notifications.EntityLicenseRequest, EntityID: item.ID,
	}); err != nil { return err }
	approved = updated
	return nil
})
return approved, err
```

Add the lock-aware helper used above:

```go
func (s *Service) pendingRequestForUpdate(ctx context.Context, requestID string) (Request, error) {
	item, err := s.repository.FindForUpdate(ctx, requestID)
	if err != nil { return Request{}, err }
	if item.Status != StatusPending { return Request{}, ErrInvalidState }
	return item, nil
}
```

Use `Repository.FindForUpdate` for approve/reject. Pass `txCtx` to reviewer, license, assignment, request, and notification dependencies. Preserve the service mutex for memory execution; PostgreSQL correctness must come from row locks and constraints, not the mutex.

- [ ] **Step 5: Wire both repositories for both storage drivers**

In `cmd/api/main.go`, declare before the storage switch:

```go
var notificationRepository notifications.Repository
var licenseRequestRepository licenserequests.Repository
var transactionManager database.Transactor
```

Memory assigns memory repositories plus `database.DirectTransactor{}`. PostgreSQL assigns `notifications.NewPostgresRepository(pool)`, `licenserequests.NewPostgresRepository(pool)`, and `database.NewPostgresTransactor(pool)`. Construct both services and handlers once after the switch; remove `if cfg.StorageDriver == "memory"` around handler creation.

- [ ] **Step 6: Run focused, HTTP, and full backend tests**

Run:

```powershell
Set-Location backend
go test ./internal/modules/licenserequests ./internal/modules/notifications ./internal/httpapi -count=1
go test ./... -count=1
go vet ./...
```

Expected: PASS. Router tests must show license-request and notification routes exist with PostgreSQL handlers.

- [ ] **Step 7: Commit atomic runtime wiring**

```powershell
git add backend/internal/modules/licenserequests backend/cmd/api/main.go backend/internal/httpapi/router_test.go
git commit -m "feat: make license request decisions transactional"
```

---

### Task 6: Idempotent Admin bootstrap and schema startup guard

**Files:**
- Create: `backend/internal/bootstrap/admin.go`
- Create: `backend/internal/bootstrap/admin_test.go`
- Create: `backend/cmd/seed/main.go`
- Modify: `backend/cmd/api/main.go`
- Test: `backend/internal/bootstrap/admin_test.go`
- Test: `backend/migrations/migrations_test.go`

**Interfaces:**
- Consumes: `auth.PasswordHasher`, `auth.PostgresRepository.FindByEmail`, `auth.PostgresRepository.CreateUser`, `migrations.RequireCurrent`.
- Produces: `bootstrap.EnsureAdmin(context.Context, AdminStore, auth.PasswordHasher, AdminInput) (Result, error)` and `go run ./cmd/seed`.

- [ ] **Step 1: Write failing Admin bootstrap tests**

```go
func TestEnsureAdminCreatesOnceAndIsIdempotent(t *testing.T) {
	store := newFakeAdminStore()
	input := AdminInput{Email: "admin@local.test", Password: "ChangeMe123!", FullName: "Development Admin", EmployeeCode: "DEV-ADMIN"}
	first, err := EnsureAdmin(t.Context(), store, auth.NewPasswordHasher(4), input)
	if err != nil || !first.Created { t.Fatalf("first=%#v err=%v", first, err) }
	second, err := EnsureAdmin(t.Context(), store, auth.NewPasswordHasher(4), input)
	if err != nil || second.Created || len(store.usersByEmail) != 1 { t.Fatalf("second=%#v err=%v", second, err) }
}
```

Define the test fake completely in `admin_test.go`:

```go
type fakeAdminStore struct {
	usersByEmail map[string]auth.User
}

func newFakeAdminStore() *fakeAdminStore {
	return &fakeAdminStore{usersByEmail: make(map[string]auth.User)}
}

func (s *fakeAdminStore) FindByEmail(_ context.Context, email string) (auth.User, error) {
	user, exists := s.usersByEmail[strings.ToLower(email)]
	if !exists { return auth.User{}, auth.ErrUserNotFound }
	return user, nil
}

func (s *fakeAdminStore) CreateUser(_ context.Context, user auth.User) (auth.User, error) {
	for _, existing := range s.usersByEmail {
		if existing.EmployeeCode == user.EmployeeCode { return auth.User{}, auth.ErrCodeAlreadyExists }
	}
	s.usersByEmail[strings.ToLower(user.Email)] = user
	return user, nil
}
```

Also test blank email/password, an existing non-admin account using the configured email, and duplicate employee code owned by another account.

- [ ] **Step 2: Run bootstrap tests and verify red**

Run: `Set-Location backend; go test ./internal/bootstrap -count=1`

Expected: FAIL because package bootstrap does not exist.

- [ ] **Step 3: Implement Admin bootstrap**

Implement these types, then use fixed ID `00000000-0000-0000-0000-000000000001`, normalized lowercase email, role `admin`, status `active`, and bcrypt cost 12 in the CLI:

```go
type AdminStore interface {
	FindByEmail(context.Context, string) (auth.User, error)
	CreateUser(context.Context, auth.User) (auth.User, error)
}

type AdminInput struct {
	Email        string
	Password     string
	FullName     string
	EmployeeCode string
}

type Result struct {
	User    auth.User
	Created bool
}
```

If `FindByEmail` finds an active Admin with employee code `DEV-ADMIN`, return `Created:false` without changing its password. Do not log the password or hash.

The CLI reads only `DATABASE_URL`, `DEV_ADMIN_EMAIL`, and `DEV_ADMIN_PASSWORD`; it does not call `config.Load`, because migration/seed commands do not need JWT or license-encryption settings. It must call `migrations.RequireCurrent` before inserting.

- [ ] **Step 4: Add schema version guard to API startup**

After `database.Open` succeeds in the PostgreSQL switch, call:

```go
if migrationErr := migrations.RequireCurrent(startupCtx, pool); migrationErr != nil {
	logger.Error("database migration required", "error", migrationErr)
	pool.Close()
	os.Exit(1)
}
```

Add a migration test proving an empty or version-3 database returns an error naming expected version 4.

- [ ] **Step 5: Run seed twice against the local database**

```powershell
Set-Location backend
$env:DEV_ADMIN_EMAIL = "admin@local.test"
$env:DEV_ADMIN_PASSWORD = Read-Host "Nhập mật khẩu Admin phát triển"
go run ./cmd/seed
go run ./cmd/seed
psql $env:DATABASE_URL -c "SELECT email, employee_code, role, status FROM users WHERE email='admin@local.test';"
```

Expected: first run reports created; second run reports already exists; query returns one active Admin.

- [ ] **Step 6: Run all backend checks and commit**

Run: `Set-Location backend; go test ./... -count=1; go vet ./...`

Expected: PASS.

```powershell
git add backend/internal/bootstrap backend/cmd/seed backend/cmd/api/main.go backend/migrations
git commit -m "feat: add PostgreSQL Admin bootstrap"
```

---

### Task 7: End-to-end PostgreSQL verification and operating documentation

**Files:**
- Create: `backend/internal/integration/postgres_persistence_test.go`
- Create: `docs/postgresql-local-setup.md`
- Create: `docs/postgresql-persistence-testing.md`
- Modify: `backend/.env.example`
- Modify: `README.md`
- Modify: `docs/getting-started-without-docker.md`
- Modify: `docs/license-request-notification-testing.md`
- Test: `backend/internal/integration/postgres_persistence_test.go`

**Interfaces:**
- Consumes: all production services/repositories, `TEST_DATABASE_URL`, migration and seed commands.
- Produces: one automated full workflow test and the exact local operating procedure for future development and Android work.

- [ ] **Step 1: Write the failing end-to-end persistence test**

Create a real PostgreSQL fixture, then execute this sequence through services:

1. seed Admin;
2. create one Employee, software product, user-assignable license with two seats;
3. create a license request as Employee;
4. approve it as Admin;
5. close the pool and open a new pool;
6. assert the request is approved, one active assignment exists, and one unread approved notification exists;
7. mark the notification read and confirm unread count becomes zero.

Name the test `TestPostgresWorkflowSurvivesReconnect`. It skips only when `TEST_DATABASE_URL` is absent.

- [ ] **Step 2: Run the end-to-end test and fix only integration defects**

Run: `Set-Location backend; go test ./internal/integration -run TestPostgresWorkflowSurvivesReconnect -count=1 -v`

Expected: PASS after correcting any wiring/query issue revealed by the real database. Do not weaken assertions or replace the reconnect with an in-memory check.

- [ ] **Step 3: Update environment examples without secrets**

`backend/.env.example` must contain:

```dotenv
APP_ENV=development
HTTP_ADDRESS=:8080
STORAGE_DRIVER=postgres
DATABASE_URL=postgres://license_admin:replace-with-local-password@localhost:5432/license_manager?sslmode=disable
TEST_DATABASE_URL=postgres://license_admin:replace-with-local-password@localhost:5432/license_manager_test?sslmode=disable
SHUTDOWN_TIMEOUT=10s
JWT_SECRET=replace-this-with-at-least-32-random-characters
LICENSE_ENCRYPTION_KEY=replace-with-a-base64-encoded-32-byte-key
JWT_ISSUER=license-manager
ACCESS_TOKEN_TTL=15m
REFRESH_TOKEN_TTL=168h
DEV_ADMIN_EMAIL=admin@local.test
DEV_ADMIN_PASSWORD=replace-with-a-local-admin-password
SEED_DEMO_DATA=false
```

- [ ] **Step 4: Write the local PostgreSQL operating guide**

Document, in order:

- verified prerequisite commands `psql --version`, `Get-Service postgresql*`, `pg_isready`;
- role/database creation from Task 1;
- session-only environment variables and safe secret generation;
- `go run ./cmd/migrate up`, `status`, and `go run ./cmd/seed`;
- backend start with `STORAGE_DRIVER=postgres`;
- expected live/ready responses;
- resolving `psql not recognized`, password authentication failure, port 5432 conflict, missing migration, and invalid encryption key;
- backup command `pg_dump $env:DATABASE_URL -Fc -f license_manager.backup` and a warning not to commit the backup.

Remove README statements saying request/notification work only in memory. Preserve the separate memory setup guide but label it temporary and not suitable for Android integration.

- [ ] **Step 5: Run the manual persistence smoke test**

Start backend:

```powershell
Set-Location "D:\Đồ án\backend"
$env:STORAGE_DRIVER = "postgres"
$env:JWT_SECRET = "dev-only-secret-change-before-production-123456"
$keyBytes = New-Object byte[] 32
$generator = [Security.Cryptography.RandomNumberGenerator]::Create()
$generator.GetBytes($keyBytes)
$generator.Dispose()
$env:LICENSE_ENCRYPTION_KEY = [Convert]::ToBase64String($keyBytes)
go run ./cmd/api
```

Verify in another terminal:

```powershell
Invoke-RestMethod http://localhost:8080/health/live
Invoke-RestMethod http://localhost:8080/health/ready
```

Use the existing web UI to create data and complete one approve/reject flow, stop API with `Ctrl+C`, run it again with the same `LICENSE_ENCRYPTION_KEY`, and confirm data and readable license keys remain intact.

- [ ] **Step 6: Run the complete verification matrix**

```powershell
Set-Location "D:\Đồ án\backend"
go test ./... -count=1
go vet ./...
Set-Location "D:\Đồ án\web"
npm.cmd test
npm.cmd run lint
npm.cmd run build
Set-Location "D:\Đồ án"
git diff --check
```

Expected: all commands exit `0`; PostgreSQL integration tests execute rather than skip because `TEST_DATABASE_URL` is set.

- [ ] **Step 7: Commit documentation and end-to-end verification**

```powershell
git add backend/internal/integration backend/.env.example README.md docs/postgresql-local-setup.md docs/postgresql-persistence-testing.md docs/getting-started-without-docker.md docs/license-request-notification-testing.md
git commit -m "docs: add PostgreSQL development workflow"
```

---

### Task 8: Final branch review and integration handoff

**Files:**
- Review: all files changed since `924b7d0`
- Review: `docs/superpowers/specs/2026-08-24-postgresql-persistence-design.md`
- Review: `docs/superpowers/plans/2026-08-24-postgresql-persistence.md`

**Interfaces:**
- Consumes: all deliverables from Tasks 1-7.
- Produces: a clean, verified feature branch ready for local merge or Pull Request.

- [ ] **Step 1: Inspect scope and secrets**

```powershell
git status --short
git diff 924b7d0 --stat
git diff 924b7d0 -- . ':!web/package-lock.json' | Select-String -Pattern 'postgres://|JWT_SECRET|LICENSE_ENCRYPTION_KEY|password' -CaseSensitive:$false
```

Expected: only intended files; connection strings contain placeholders only; no real secret or backup file is tracked.

- [ ] **Step 2: Re-run fresh verification**

Run the exact Task 7 Step 6 matrix again immediately before declaring completion.

Expected: every command exits `0` and the test output shows PostgreSQL workflow tests passing.

- [ ] **Step 3: Confirm branch history and working tree**

```powershell
git status
git log --oneline --decorate -10
```

Expected: working tree clean except the pre-existing untracked `.superpowers/` directory, and focused commits for migrations, transaction support, repositories, bootstrap, and documentation.

- [ ] **Step 4: Present integration options**

Use `superpowers:finishing-a-development-branch`; do not merge or push without the user's explicit choice.
