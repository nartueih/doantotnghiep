# License Request and Website Notification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Xây dựng luồng nhân viên yêu cầu cấp license, Admin/IT duyệt hoặc từ chối, tự động cấp phát khi duyệt và gửi thông báo trong website.

**Architecture:** Thêm hai module Go độc lập `licenserequests` và `notifications`, dùng repository memory có mutex và service nghiệp vụ. License-request service tái sử dụng assignment service để giữ nguyên quy tắc seat/cấp phát; web thêm API client, khu vực self-service cho Employee và màn hình quản trị riêng.

**Tech Stack:** Go 1.25, Gin, repository memory, Go `testing`, React 19, TypeScript 6, Vite 8, Oxlint, Node 24 test runner.

**Spec:** `docs/superpowers/specs/2026-08-23-license-request-notification-design.md`

## Global Constraints

- Chỉ triển khai lưu trữ `memory`; không thêm PostgreSQL migration hoặc PostgreSQL repository trong kế hoạch này.
- User ID của Employee luôn lấy từ JWT access token, không nhận từ request body hoặc query.
- Không trả về hoặc ghi log license key trong request, notification hay audit metadata.
- Chỉ cho phép một request `pending` trên mỗi cặp requester/phần mềm.
- Trạng thái chỉ chuyển từ `pending` sang `approved`, `rejected` hoặc `cancelled`.
- Duyệt phải tái sử dụng assignment service để giữ kiểm tra seat, thời hạn, assignment type, target và cấp trùng.
- Form bắt buộc dùng placeholder/hint “không được bỏ trống”, không thêm dấu sao đỏ.
- Mọi thay đổi phải giữ toàn bộ backend tests, web lint và web production build hiện có chạy thành công.

---

## File Structure

### Backend files to create

- `backend/internal/modules/notifications/model.go`: notification model, errors và repository contract.
- `backend/internal/modules/notifications/memory_repository.go`: lưu, lọc và cập nhật read state an toàn đồng thời.
- `backend/internal/modules/notifications/service.go`: tạo/list/read/read-all notifications theo user.
- `backend/internal/modules/notifications/service_test.go`: service và ownership tests.
- `backend/internal/modules/notifications/http.go`: `/api/v1/me/notifications` routes.
- `backend/internal/modules/notifications/http_test.go`: authentication và HTTP contract tests.
- `backend/internal/modules/licenserequests/model.go`: request model, filter, errors và dependency interfaces.
- `backend/internal/modules/licenserequests/memory_repository.go`: persistence, duplicate check và state transitions.
- `backend/internal/modules/licenserequests/service.go`: validation, concurrency guard, approval orchestration và notification creation.
- `backend/internal/modules/licenserequests/service_test.go`: business-flow và concurrent approval tests.
- `backend/internal/modules/licenserequests/http.go`: Employee và Admin/IT routes cùng audit recording.
- `backend/internal/modules/licenserequests/http_test.go`: authorization, status mapping và audit tests.

### Backend files to modify

- `backend/internal/modules/audit/model.go`: thêm request/approve/reject/cancel actions và license-request entity.
- `backend/internal/httpapi/router.go`: đăng ký hai handler mới.
- `backend/internal/httpapi/router_test.go`: cập nhật constructor calls.
- `backend/internal/httpapi/openapi.json`: mô tả endpoints và schemas mới.
- `backend/internal/httpapi/openapi_test.go`: bắt buộc các path mới và cập nhật constructor.
- `backend/cmd/api/main.go`: chỉ khởi tạo memory repositories/services/handlers cho feature mới và truyền vào router.

### Web files to create

- `web/src/lib/license-request-api.ts`: request/notification models và API functions.
- `web/src/features/requests/request-view-model.ts`: label, filter và license eligibility helpers thuần.
- `web/src/features/requests/request-view-model.test.ts`: Node tests cho helpers.
- `web/src/features/requests/LicenseRequestManagementScreen.tsx`: trang Admin/IT xử lý requests.
- `web/src/features/requests/LicenseRequestManagementScreen.css`: layout responsive cho trang quản trị.

### Web files to modify

- `web/package.json`: thêm script `test` dùng Node TypeScript stripping.
- `web/src/App.tsx`: thêm hash route và render màn hình requests.
- `web/src/components/layout/AdminShell.tsx`: thêm `requests` page, navigation item và icon.
- `web/src/features/employee/EmployeePortalScreen.tsx`: tải request/notification, form tạo/hủy, notification panel.
- `web/src/features/employee/EmployeePortalScreen.css`: styles cho request history, modal và notification panel.

### Documentation files to create/modify

- `docs/license-request-notification-testing.md`: kịch bản PowerShell và giao diện memory.
- `README.md`: đánh dấu module và dẫn tới hướng dẫn kiểm thử.

---

### Task 1: Notification Memory Module

**Files:**
- Create: `backend/internal/modules/notifications/model.go`
- Create: `backend/internal/modules/notifications/memory_repository.go`
- Create: `backend/internal/modules/notifications/service.go`
- Create: `backend/internal/modules/notifications/service_test.go`
- Create: `backend/internal/modules/notifications/http.go`
- Create: `backend/internal/modules/notifications/http_test.go`

**Interfaces:**
- Produces: `notifications.Repository`, `notifications.Service`, `notifications.HTTPHandler`.
- Produces: `CreateInput{UserID, Type, Title, Message, EntityType, EntityID}` and `Service.Create(ctx, input)` for license-request service.
- Produces routes: `GET /api/v1/me/notifications`, `PATCH /api/v1/me/notifications/:id/read`, `PATCH /api/v1/me/notifications/read-all`.

- [ ] **Step 1: Write failing service tests**

Create tests that pin ownership, ordering and idempotent read behavior:

```go
func TestServiceListsOnlyCurrentUsersNotifications(t *testing.T) {
    service := NewService(NewMemoryRepository())
    _, _ = service.Create(context.Background(), CreateInput{UserID: "user-1", Type: TypeLicenseRequestApproved, Title: "Đã duyệt", Message: "Adobe", EntityType: EntityLicenseRequest, EntityID: "request-1"})
    _, _ = service.Create(context.Background(), CreateInput{UserID: "user-2", Type: TypeLicenseRequestRejected, Title: "Đã từ chối", Message: "Office", EntityType: EntityLicenseRequest, EntityID: "request-2"})

    result, err := service.List(context.Background(), "user-1")
    if err != nil || len(result.Items) != 1 || result.UnreadCount != 1 || result.Items[0].UserID != "user-1" {
        t.Fatalf("unexpected result: %#v, %v", result, err)
    }
}

func TestServiceMarkReadIsOwnedAndIdempotent(t *testing.T) {
    service := NewService(NewMemoryRepository())
    item, _ := service.Create(context.Background(), CreateInput{UserID: "user-1", Type: TypeLicenseRequestApproved, Title: "Đã duyệt", Message: "Adobe", EntityType: EntityLicenseRequest, EntityID: "request-1"})
    if _, err := service.MarkRead(context.Background(), "user-2", item.ID); !errors.Is(err, ErrNotFound) {
        t.Fatalf("expected not found, got %v", err)
    }
    first, err := service.MarkRead(context.Background(), "user-1", item.ID)
    if err != nil || first.ReadAt == nil {
        t.Fatalf("expected read notification: %#v, %v", first, err)
    }
    second, err := service.MarkRead(context.Background(), "user-1", item.ID)
    if err != nil || !second.ReadAt.Equal(*first.ReadAt) {
        t.Fatalf("expected idempotent read: %#v, %v", second, err)
    }
}
```

- [ ] **Step 2: Run the notification tests and verify RED**

Run: `cd backend; go test ./internal/modules/notifications`

Expected: FAIL because package/types do not exist.

- [ ] **Step 3: Implement notification model, repository and service**

Define these exact contracts in `model.go`:

```go
const (
    TypeLicenseRequestApproved = "license_request_approved"
    TypeLicenseRequestRejected = "license_request_rejected"
    EntityLicenseRequest       = "license_request"
)

var (
    ErrNotFound    = errors.New("notification not found")
    ErrInvalidData = errors.New("notification data is required")
)

type Notification struct {
    ID, UserID, Type, Title, Message, EntityType, EntityID string
    CreatedAt time.Time
    ReadAt *time.Time
}

type Repository interface {
    ListByUser(context.Context, string) ([]Notification, error)
    Create(context.Context, Notification) (Notification, error)
    MarkRead(context.Context, string, string, time.Time) (Notification, error)
    MarkAllRead(context.Context, string, time.Time) (int, error)
}
```

Implement `MemoryRepository` with `sync.RWMutex`, newest-first sorting and ownership checks. Implement `Service` with `now func() time.Time`, UUID generation through `platform/id`, trimmed validation, `ListResult{Items, Total, UnreadCount}`, idempotent `MarkRead` and `MarkAllRead`.

- [ ] **Step 4: Run service tests and verify GREEN**

Run: `cd backend; go test ./internal/modules/notifications`

Expected: PASS for service tests; HTTP tests are not added yet.

- [ ] **Step 5: Write failing HTTP tests**

Cover no token (`401`), employee list (`200` with `unread_count`), marking another user’s notification (`404`), read-one (`200`) and read-all (`200` with `updated`). Use the same low-cost password hasher/token fixture pattern as `selfservice/http_test.go`.

- [ ] **Step 6: Implement notification HTTP routes**

Register under `/me/notifications` with `Authenticate()` and `RequireRoles(auth.RoleEmployee)`. Always derive `userID := auth.CurrentUserID(c)`. Map malformed IDs/ownership to `404`, repository failures to `500`, and successful read-all to `{ "updated": number }`.

- [ ] **Step 7: Run tests and commit**

Run: `cd backend; go test ./internal/modules/notifications`

Expected: PASS.

```powershell
git add backend/internal/modules/notifications
git commit -m "feat: add memory website notifications"
```

---

### Task 2: License Request Domain and Memory Workflow

**Files:**
- Create: `backend/internal/modules/licenserequests/model.go`
- Create: `backend/internal/modules/licenserequests/memory_repository.go`
- Create: `backend/internal/modules/licenserequests/service.go`
- Create: `backend/internal/modules/licenserequests/service_test.go`

**Interfaces:**
- Consumes: `software.Repository`, `licenses.Repository`, `auth.Repository` through narrow finder/catalog interfaces.
- Consumes: `AssignmentCreator.Create(context.Context, actorID string, assignments.CreateInput) (assignments.Assignment, error)`.
- Consumes: `NotificationCreator.Create(context.Context, notifications.CreateInput) (notifications.Notification, error)`.
- Produces: `Service.ListMine`, `ListAdmin`, `RequestableSoftware`, `Create`, `Cancel`, `Approve`, `Reject`.

- [ ] **Step 1: Write failing creation and state-transition tests**

Create fixtures with one Employee, one Admin, one software product, an active user-assignable license with seats, memory assignment repository and notification service. Pin these behaviors:

```go
func TestCreateRejectsDuplicatePendingRequest(t *testing.T) {
    fixture := newRequestFixture(t)
    input := CreateInput{SoftwareProductID: fixture.product.ID, Priority: PriorityHigh, Reason: "Cần cho dự án thiết kế"}
    if _, err := fixture.service.Create(context.Background(), fixture.employee.ID, input); err != nil {
        t.Fatal(err)
    }
    if _, err := fixture.service.Create(context.Background(), fixture.employee.ID, input); !errors.Is(err, ErrPendingDuplicate) {
        t.Fatalf("expected duplicate, got %v", err)
    }
}

func TestCancelOnlyAllowsOwnerOfPendingRequest(t *testing.T) {
    fixture := newRequestFixture(t)
    item, _ := fixture.service.Create(context.Background(), fixture.employee.ID, validCreateInput(fixture))
    if _, err := fixture.service.Cancel(context.Background(), "another-user", item.ID); !errors.Is(err, ErrNotFound) {
        t.Fatalf("expected hidden ownership error, got %v", err)
    }
    cancelled, err := fixture.service.Cancel(context.Background(), fixture.employee.ID, item.ID)
    if err != nil || cancelled.Status != StatusCancelled {
        t.Fatalf("unexpected cancel: %#v, %v", cancelled, err)
    }
}
```

- [ ] **Step 2: Run package tests and verify RED**

Run: `cd backend; go test ./internal/modules/licenserequests`

Expected: FAIL because the package is not implemented.

- [ ] **Step 3: Implement model and memory repository**

Define constants exactly:

```go
const (
    PriorityNormal = "normal"
    PriorityHigh   = "high"
    PriorityUrgent = "urgent"
    StatusPending   = "pending"
    StatusApproved  = "approved"
    StatusRejected  = "rejected"
    StatusCancelled = "cancelled"
    DecisionOutOfStock  = "out_of_stock"
    DecisionNotApproved = "not_approved"
    DecisionOther       = "other"
)
```

Repository contract:

```go
type Repository interface {
    List(context.Context, Filter) ([]Request, error)
    ListByRequester(context.Context, string) ([]Request, error)
    FindByID(context.Context, string) (Request, error)
    Create(context.Context, Request) (Request, error)
    Cancel(context.Context, string, string, time.Time) (Request, error)
    Approve(context.Context, ApprovalUpdate) (Request, error)
    Reject(context.Context, RejectionUpdate) (Request, error)
}
```

Use `sync.RWMutex`, newest-first ordering, case-insensitive search and exact status/priority filtering. Enforce pending duplicate in `Create` while holding the write lock. Enforce owner and pending state atomically in `Cancel`; enforce pending state in `Approve` and `Reject`.

- [ ] **Step 4: Implement minimal service validation**

Use a service-wide `transitionMu sync.Mutex` around `Create`, `Cancel`, `Approve` and `Reject`. Trim all inputs. Require non-empty software ID/reason, allowed priority, active requester, existing software, allowed decision reason and non-empty rejection response. `ListMine` must only call `ListByRequester` with token user ID.

- [ ] **Step 5: Run creation/state tests and verify GREEN**

Run: `cd backend; go test ./internal/modules/licenserequests -run 'TestCreate|TestCancel|TestReject'`

Expected: PASS.

- [ ] **Step 6: Write failing approval and concurrency tests**

Pin license/product matching, assignment creation, notification creation, no-seat behavior remaining pending and concurrent approval:

```go
func TestApproveCreatesAssignmentAndNotification(t *testing.T) {
    fixture := newRequestFixture(t)
    item, _ := fixture.service.Create(context.Background(), fixture.employee.ID, validCreateInput(fixture))
    approved, err := fixture.service.Approve(context.Background(), fixture.admin.ID, item.ID, ApproveInput{LicenseID: fixture.license.ID, ResponseNote: "Đã cấp"})
    if err != nil || approved.Status != StatusApproved || approved.AssignmentID == "" {
        t.Fatalf("unexpected approval: %#v, %v", approved, err)
    }
    notices, _ := fixture.notifications.List(context.Background(), fixture.employee.ID)
    if notices.UnreadCount != 1 || notices.Items[0].Type != notifications.TypeLicenseRequestApproved {
        t.Fatalf("missing approval notification: %#v", notices)
    }
}

func TestConcurrentApprovalCreatesOneAssignment(t *testing.T) {
    fixture := newRequestFixture(t)
    item, _ := fixture.service.Create(context.Background(), fixture.employee.ID, validCreateInput(fixture))
    errorsChannel := make(chan error, 2)
    for range 2 {
        go func() {
            _, err := fixture.service.Approve(context.Background(), fixture.admin.ID, item.ID, ApproveInput{LicenseID: fixture.license.ID})
            errorsChannel <- err
        }()
    }
    successes := 0
    for range 2 {
        if <-errorsChannel == nil { successes++ }
    }
    assignmentsList, _ := fixture.assignments.List(context.Background())
    if successes != 1 || len(assignmentsList) != 1 {
        t.Fatalf("successes=%d assignments=%d", successes, len(assignmentsList))
    }
}
```

- [ ] **Step 7: Implement approval/rejection orchestration**

`Approve` must load the pending request, load the selected license, compare `license.SoftwareProductID`, then call assignment creator with `UserID: request.RequesterID` and notes that reference the request. Only after assignment succeeds may it call repository `Approve`, then notification `Create`. Map assignment errors without replacing their identities so HTTP can reuse current status semantics.

`Reject` must update the request first and create `license_request_rejected` notification whose message includes the human response but no license key. The memory dependencies do not return post-write failures; do not add distributed rollback or outbox logic in this phase.

- [ ] **Step 8: Run package and race tests, then commit**

Run: `cd backend; go test -race ./internal/modules/licenserequests ./internal/modules/notifications`

Expected: PASS with no race report.

```powershell
git add backend/internal/modules/licenserequests
git commit -m "feat: add memory license request workflow"
```

---

### Task 3: HTTP Contracts, Audit, Wiring and OpenAPI

**Files:**
- Create: `backend/internal/modules/licenserequests/http.go`
- Create: `backend/internal/modules/licenserequests/http_test.go`
- Modify: `backend/internal/modules/audit/model.go`
- Modify: `backend/internal/httpapi/router.go`
- Modify: `backend/internal/httpapi/router_test.go`
- Modify: `backend/internal/httpapi/openapi.json`
- Modify: `backend/internal/httpapi/openapi_test.go`
- Modify: `backend/cmd/api/main.go`

**Interfaces:**
- Consumes services from Tasks 1-2.
- Produces all Employee and Admin/IT endpoints defined by the spec.
- Produces audit actions `request`, `cancel`, `approve`, `reject` and entity `license_request`.

- [ ] **Step 1: Write failing HTTP authorization and flow tests**

Build a Gin test router with auth, audit, notifications, software/license/assignment fixtures and request handler. Cover:

```go
func TestEmployeeCanCreateAndListOwnLicenseRequest(t *testing.T) { /* POST returns 201; GET /me returns exactly that requester */ }
func TestEmployeeCannotUseAdminLicenseRequestList(t *testing.T) { /* GET /license-requests returns 403 */ }
func TestAdminCannotCreateEmployeeSelfServiceRequest(t *testing.T) { /* POST /me/license-requests returns 403 */ }
func TestAdminApprovalCreatesAuditEntry(t *testing.T) { /* PATCH approve returns 200 and audit ActionApprove exists */ }
func TestAdminRejectOutOfStockCreatesAuditWithoutKey(t *testing.T) { /* PATCH reject; serialized metadata excludes license_key */ }
func TestCancelAnotherUsersRequestReturnsNotFound(t *testing.T) { /* PATCH cancel returns 404 */ }
```

- [ ] **Step 2: Run HTTP tests and verify RED**

Run: `cd backend; go test ./internal/modules/licenserequests`

Expected: FAIL because HTTP handler/routes are missing.

- [ ] **Step 3: Implement Employee and Admin/IT HTTP groups**

Register Employee routes under `/me` with `Authenticate()` plus `RequireRoles(employee)`. Register admin routes under `/license-requests` with `Authenticate()` plus `RequireRoles(admin, it_manager)`. Parse bodies into service inputs and map errors:

- invalid data/priority/rejection reason: `400`;
- hidden ownership or missing resource: `404`;
- duplicate pending, invalid state, inactive/no-seat/duplicate assignment: `409`;
- license-product or assignment-type mismatch: `422`;
- unexpected: `500`.

After successful create/cancel/approve/reject, call `audit.RecordRequest` with the new action/entity constants and IDs only.

- [ ] **Step 4: Run request HTTP tests and verify GREEN**

Run: `cd backend; go test ./internal/modules/licenserequests ./internal/modules/audit`

Expected: PASS.

- [ ] **Step 5: Wire memory-only feature into main and router**

Add `notificationsHandler` and `licenseRequestHandler` parameters to `httpapi.NewRouter` and update its three direct callers. In `main.go`, initialize both memory repositories only when `cfg.StorageDriver == "memory"`; leave the handlers nil in the PostgreSQL branch so the feature does not silently mix storage drivers. Construct request service after assignment service exists. Router registers non-nil handlers.

- [ ] **Step 6: Extend OpenAPI and its contract test**

Add all ten paths from the design and schemas `LicenseRequest`, `LicenseRequestCreate`, `LicenseRequestApprove`, `LicenseRequestReject`, `Notification`, `NotificationList`. Mark enum values exactly as the model constants and add `401/403/404/409/422` responses where applicable. Extend `requiredPaths` in `openapi_test.go`.

- [ ] **Step 7: Run backend formatting and full tests**

Run:

```powershell
cd backend
gofmt -w ./cmd/api ./internal/httpapi ./internal/modules/audit ./internal/modules/notifications ./internal/modules/licenserequests
go test ./...
go vet ./...
```

Expected: all packages PASS; `go vet` exits 0.

- [ ] **Step 8: Commit backend integration**

```powershell
git add backend/cmd/api backend/internal/httpapi backend/internal/modules/audit backend/internal/modules/notifications backend/internal/modules/licenserequests
git commit -m "feat: expose license request APIs"
```

---

### Task 4: Web API Client and Tested View-Model Helpers

**Files:**
- Create: `web/src/lib/license-request-api.ts`
- Create: `web/src/features/requests/request-view-model.ts`
- Create: `web/src/features/requests/request-view-model.test.ts`
- Modify: `web/package.json`

**Interfaces:**
- Produces API functions `listMyLicenseRequests`, `listRequestableSoftware`, `createLicenseRequest`, `cancelLicenseRequest`, `listNotifications`, `markNotificationRead`, `markAllNotificationsRead`, `listLicenseRequests`, `approveLicenseRequest`, `rejectLicenseRequest`.
- Produces pure helpers `requestStatusLabel`, `requestPriorityLabel`, `rejectReasonLabel`, `eligibleLicenses`.

- [ ] **Step 1: Add Node test script and failing helper tests**

Add script:

```json
"test": "node --test --experimental-strip-types src/features/requests/request-view-model.test.ts"
```

Test labels and eligibility using `node:test` and `node:assert/strict`:

```ts
test('eligibleLicenses keeps active matching user licenses with seats', () => {
  const result = eligibleLicenses(licenses, 'software-1')
  assert.deepEqual(result.map((item) => item.id), ['eligible-license'])
})

test('request labels are Vietnamese and deterministic', () => {
  assert.equal(requestStatusLabel('pending'), 'Đang chờ')
  assert.equal(requestPriorityLabel('urgent'), 'Khẩn cấp')
  assert.equal(rejectReasonLabel('out_of_stock'), 'Tạm hết license')
})
```

- [ ] **Step 2: Run web tests and verify RED**

Run: `cd web; npm.cmd test`

Expected: FAIL because helper module/exports are missing.

- [ ] **Step 3: Implement exact TypeScript models and API client**

Define literal unions matching backend enums, typed response wrappers and one shared request helper using `VITE_API_BASE_URL`, bearer token, JSON parsing and the project’s existing `APIError` behavior. `eligibleLicenses` must require matching `software_product_id`, `lifecycle_status === 'active'`, `available_seats > 0`, and assignment type `user` or `mixed`.

- [ ] **Step 4: Run helper tests and build**

Run:

```powershell
cd web
npm.cmd test
npm.cmd run build
```

Expected: tests PASS and Vite build succeeds.

- [ ] **Step 5: Commit API foundation**

```powershell
git add web/package.json web/src/lib/license-request-api.ts web/src/features/requests/request-view-model.ts web/src/features/requests/request-view-model.test.ts
git commit -m "feat: add web license request API client"
```

---

### Task 5: Employee Request History and Notification Panel

**Files:**
- Modify: `web/src/features/employee/EmployeePortalScreen.tsx`
- Modify: `web/src/features/employee/EmployeePortalScreen.css`

**Interfaces:**
- Consumes Employee API functions and view-model labels from Task 4.
- Produces Employee create/cancel/history/notification experience.

- [ ] **Step 1: Record the pre-change web verification baseline**

Run: `cd web; npm.cmd test; npm.cmd run lint; npm.cmd run build`

Expected: PASS before the UI edit.

- [ ] **Step 2: Add request/notification state and data loading**

Load devices, assigned licenses, requestable software, own requests and notifications together. Keep independent `isSubmittingRequest`, `cancellingRequestID`, `notificationAction` states so existing portal data remains visible during mutations. A `401` from any call must reuse the existing expired-session error flow.

- [ ] **Step 3: Add Employee request UI**

Add a `Yêu cầu license` section with count summary, newest-first cards/table and `Tạo yêu cầu` modal. Fields:

- software select placeholder `Chọn phần mềm — không được bỏ trống`;
- priority default `normal`;
- reason textarea placeholder `Nhập lý do sử dụng — không được bỏ trống`.

Disable submit until valid. After successful creation, close/reset modal and refresh request data. Pending items show a confirmation-gated `Hủy yêu cầu` action. Approved items link their assignment outcome text; rejected items display reason label and Admin/IT response.

- [ ] **Step 4: Add website notification panel**

Replace the decorative bell behavior with a button showing `unread_count`. Panel entries show title, message and Vietnamese date/time; unread entries have a visible indicator. Clicking an unread item calls read-one; `Đánh dấu tất cả đã đọc` calls read-all. Clicking outside/close button dismisses the panel without changing read state.

- [ ] **Step 5: Add responsive styles and accessibility**

Use the existing portal tokens, font scale and breakpoints. Add dialog `role="dialog"`, labels, focusable buttons, visible disabled states and `aria-live` for mutation errors/success. On narrow screens, render request rows as stacked cards and keep the notification panel within viewport width.

- [ ] **Step 6: Verify and commit Employee UI**

Run: `cd web; npm.cmd test; npm.cmd run lint; npm.cmd run build`

Expected: all commands exit 0.

```powershell
git add web/src/features/employee
git commit -m "feat: add employee license requests and notifications"
```

---

### Task 6: Admin/IT License Request Management Screen

**Files:**
- Create: `web/src/features/requests/LicenseRequestManagementScreen.tsx`
- Create: `web/src/features/requests/LicenseRequestManagementScreen.css`
- Modify: `web/src/App.tsx`
- Modify: `web/src/components/layout/AdminShell.tsx`

**Interfaces:**
- Consumes Admin API functions, `listLicenses` and Task 4 helpers.
- Produces hash route `#/requests` and `AdminPage = 'requests'`.

- [ ] **Step 1: Add route and navigation compile failure first**

Add `requests` to `AdminPage`, navigation with label `Yêu cầu`, a request/inbox icon, `pageFromHash()` mapping and screen import/render in `App.tsx` before creating the component.

Run: `cd web; npm.cmd run build`

Expected: FAIL because `LicenseRequestManagementScreen` does not exist.

- [ ] **Step 2: Implement list, filters and summary**

Create the screen with `AdminShell activePage="requests"`. Load all requests and licenses, then implement controlled `search`, `status` and `priority` filters. Display summary counts for pending/approved/rejected and a responsive table with requester, software, priority, created time, status and actions.

- [ ] **Step 3: Implement approval modal**

Use `eligibleLicenses(allLicenses, request.software_product_id)`. Each option shows license name plus `available_seats/seat_count`. Require a selection, allow optional response note, show an explicit empty state when no license qualifies, and refresh both request list and license list after success. For `409` no-seat, keep the request pending and show: `License vừa hết seat. Hãy từ chối với lý do Tạm hết license để phản hồi nhân viên.`

- [ ] **Step 4: Implement rejection modal and quick out-of-stock action**

Require decision reason and non-blank response note. The quick `Tạm hết license` action opens the same modal preselected with `out_of_stock`; it must not submit without confirmation. After success, close/reset the modal and refresh requests.

- [ ] **Step 5: Add styles and verify RED-to-GREEN build**

Follow established management screen spacing, typography, status badges, modal overlay and mobile cards. Avoid red required asterisks. Ensure all loading, empty, API error and session-expired states render.

Run: `cd web; npm.cmd test; npm.cmd run lint; npm.cmd run build`

Expected: all commands exit 0.

- [ ] **Step 6: Commit Admin UI**

```powershell
git add web/src/App.tsx web/src/components/layout/AdminShell.tsx web/src/features/requests
git commit -m "feat: add license request management screen"
```

---

### Task 7: Documentation, Full Verification and Handoff

**Files:**
- Create: `docs/license-request-notification-testing.md`
- Modify: `README.md`

**Interfaces:**
- Consumes every route and UI flow from Tasks 1-6.
- Produces repeatable PowerShell and browser verification instructions for memory mode.

- [ ] **Step 1: Write the manual test guide**

Document exact commands to start backend with:

```powershell
$env:STORAGE_DRIVER = "memory"
$env:SEED_DEMO_DATA = "true"
$env:JWT_SECRET = "dev-only-secret-change-before-production-123456"
go run ./cmd/api
```

Document starting web with `npm.cmd run dev`, Employee login, create/cancel, Admin login in a separate browser profile, approve flow, assignment verification, reject `out_of_stock`, Employee notification/read flow, audit verification, duplicate pending and exhausted-seat negative cases. State that restarting backend clears requests and notifications.

- [ ] **Step 2: Update README module and testing links**

Add License Requests and Website Notifications under implemented modules, memory-only limitation, endpoint summary and link to `docs/license-request-notification-testing.md`.

- [ ] **Step 3: Run final backend verification**

Run:

```powershell
cd backend
go test ./...
go test -race ./internal/modules/licenserequests ./internal/modules/notifications ./internal/modules/assignments
go vet ./...
```

Expected: every command exits 0 and race detector reports no race.

- [ ] **Step 4: Run final web verification**

Run:

```powershell
cd web
npm.cmd test
npm.cmd run lint
npm.cmd run build
```

Expected: tests pass, Oxlint reports no errors and Vite production build succeeds.

- [ ] **Step 5: Inspect security and repository diff**

Run:

```powershell
cd "D:\Đồ án"
rg -n "license_key|encrypted_key" backend/internal/modules/licenserequests backend/internal/modules/notifications web/src/lib/license-request-api.ts web/src/features/requests
git diff --check
git status --short
```

Expected: request/notification code does not serialize, log or display a key; diff check is clean; only intended files are modified.

- [ ] **Step 6: Commit docs and final fixes**

```powershell
git add README.md docs/license-request-notification-testing.md
git commit -m "docs: add license request testing guide"
```

- [ ] **Step 7: Final handoff**

Report exact verification commands/results, commits created, memory data-loss limitation and provide the user with the browser test sequence. Do not merge or push without a separate user request.
