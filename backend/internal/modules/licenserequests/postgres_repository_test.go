package licenserequests

import (
	"errors"
	"sync"
	"testing"
	"time"

	"license-manager/backend/internal/testsupport"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	requestUserOneID  = "41000000-0000-0000-0000-000000000001"
	requestUserTwoID  = "41000000-0000-0000-0000-000000000002"
	requestReviewerID = "41000000-0000-0000-0000-000000000003"
	requestProductOne = "42000000-0000-0000-0000-000000000001"
	requestProductTwo = "42000000-0000-0000-0000-000000000002"
	requestLicenseOne = "43000000-0000-0000-0000-000000000001"
)

func TestPostgresRepositoryCreatesListsSearchesAndCancels(t *testing.T) {
	pool := testsupport.OpenPostgres(t)
	seedLicenseRequestReferences(t, pool)
	repository := NewPostgresRepository(pool)
	createdAt := time.Date(2026, time.August, 24, 9, 0, 0, 0, time.UTC)

	first := pendingRequest(
		"40000000-0000-0000-0000-000000000001",
		requestUserOneID,
		"Employee One",
		requestProductOne,
		"Adobe Photoshop",
		PriorityHigh,
		createdAt,
	)
	second := pendingRequest(
		"40000000-0000-0000-0000-000000000002",
		requestUserTwoID,
		"Employee Two",
		requestProductTwo,
		"Microsoft Office",
		PriorityNormal,
		createdAt.Add(time.Minute),
	)
	if _, err := repository.Create(t.Context(), first); err != nil {
		t.Fatalf("create first request: %v", err)
	}
	if _, err := repository.Create(t.Context(), second); err != nil {
		t.Fatalf("create second request: %v", err)
	}

	items, err := repository.List(t.Context(), Filter{
		Status: StatusPending, Priority: PriorityHigh, Search: "photo",
	})
	if err != nil || len(items) != 1 || items[0].ID != first.ID {
		t.Fatalf("filtered items=%#v err=%v", items, err)
	}
	owned, err := repository.ListByRequester(t.Context(), requestUserOneID)
	if err != nil || len(owned) != 1 || owned[0].ID != first.ID {
		t.Fatalf("owned items=%#v err=%v", owned, err)
	}
	locked, err := repository.FindForUpdate(t.Context(), first.ID)
	if err != nil || locked.ID != first.ID {
		t.Fatalf("find for update=%#v err=%v", locked, err)
	}

	cancelledAt := createdAt.Add(2 * time.Hour)
	if _, err := repository.Cancel(t.Context(), first.ID, requestUserTwoID, cancelledAt); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ownership-safe not found, got %v", err)
	}
	cancelled, err := repository.Cancel(t.Context(), first.ID, requestUserOneID, cancelledAt)
	if err != nil || cancelled.Status != StatusCancelled || cancelled.CancelledAt == nil || !cancelled.CancelledAt.Equal(cancelledAt) {
		t.Fatalf("cancelled=%#v err=%v", cancelled, err)
	}
	if _, err := repository.Cancel(t.Context(), first.ID, requestUserOneID, cancelledAt.Add(time.Hour)); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected terminal-state conflict, got %v", err)
	}
}

func TestPostgresRepositoryPreventsConcurrentPendingDuplicates(t *testing.T) {
	pool := testsupport.OpenPostgres(t)
	seedLicenseRequestReferences(t, pool)
	repository := NewPostgresRepository(pool)
	createdAt := time.Date(2026, time.August, 24, 9, 0, 0, 0, time.UTC)

	start := make(chan struct{})
	results := make(chan error, 2)
	var waitGroup sync.WaitGroup
	for index, requestID := range []string{
		"40000000-0000-0000-0000-000000000011",
		"40000000-0000-0000-0000-000000000012",
	} {
		waitGroup.Add(1)
		go func(index int, requestID string) {
			defer waitGroup.Done()
			<-start
			_, err := repository.Create(t.Context(), pendingRequest(
				requestID,
				requestUserOneID,
				"Employee One",
				requestProductOne,
				"Adobe Photoshop",
				PriorityNormal,
				createdAt.Add(time.Duration(index)*time.Second),
			))
			results <- err
		}(index, requestID)
	}
	close(start)
	waitGroup.Wait()
	close(results)

	successes := 0
	duplicates := 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrPendingDuplicate):
			duplicates++
		default:
			t.Fatalf("unexpected create error: %v", err)
		}
	}
	if successes != 1 || duplicates != 1 {
		t.Fatalf("successes=%d duplicates=%d", successes, duplicates)
	}
}

func TestPostgresRepositoryApprovesRejectsAndProtectsTerminalStates(t *testing.T) {
	pool := testsupport.OpenPostgres(t)
	seedLicenseRequestReferences(t, pool)
	repository := NewPostgresRepository(pool)
	createdAt := time.Date(2026, time.August, 24, 9, 0, 0, 0, time.UTC)

	approvedRequest := pendingRequest(
		"40000000-0000-0000-0000-000000000021",
		requestUserOneID,
		"Employee One",
		requestProductOne,
		"Adobe Photoshop",
		PriorityUrgent,
		createdAt,
	)
	if _, err := repository.Create(t.Context(), approvedRequest); err != nil {
		t.Fatal(err)
	}
	const assignmentID = "44000000-0000-0000-0000-000000000001"
	if _, err := pool.Exec(t.Context(), `
		INSERT INTO license_assignments (
			id, license_id, user_id, assigned_by, assigned_at, status
		) VALUES ($1, $2, $3, $4, $5, 'active')
	`, assignmentID, requestLicenseOne, requestUserOneID, requestReviewerID, createdAt); err != nil {
		t.Fatalf("seed assignment: %v", err)
	}
	reviewedAt := createdAt.Add(time.Hour)
	approved, err := repository.Approve(t.Context(), ApprovalUpdate{
		RequestID: approvedRequest.ID, LicenseID: requestLicenseOne,
		LicenseName: "Adobe Business", AssignmentID: assignmentID,
		ReviewerID: requestReviewerID, ReviewerName: "Admin User",
		ResponseNote: "Đã cấp", ReviewedAt: reviewedAt,
	})
	if err != nil || approved.Status != StatusApproved || approved.AssignmentID != assignmentID {
		t.Fatalf("approved=%#v err=%v", approved, err)
	}
	if _, err := repository.Approve(t.Context(), ApprovalUpdate{
		RequestID: approvedRequest.ID, LicenseID: requestLicenseOne,
		LicenseName: "Adobe Business", AssignmentID: assignmentID,
		ReviewerID: requestReviewerID, ReviewerName: "Admin User", ReviewedAt: reviewedAt,
	}); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected approval conflict, got %v", err)
	}

	rejectedRequest := pendingRequest(
		"40000000-0000-0000-0000-000000000022",
		requestUserTwoID,
		"Employee Two",
		requestProductTwo,
		"Microsoft Office",
		PriorityNormal,
		createdAt.Add(time.Minute),
	)
	if _, err := repository.Create(t.Context(), rejectedRequest); err != nil {
		t.Fatal(err)
	}
	rejected, err := repository.Reject(t.Context(), RejectionUpdate{
		RequestID: rejectedRequest.ID, ReviewerID: requestReviewerID,
		ReviewerName: "Admin User", DecisionReason: DecisionOutOfStock,
		ResponseNote: "Tạm hết license", ReviewedAt: reviewedAt,
	})
	if err != nil || rejected.Status != StatusRejected || rejected.DecisionReason != DecisionOutOfStock {
		t.Fatalf("rejected=%#v err=%v", rejected, err)
	}
	if _, err := repository.Cancel(t.Context(), rejectedRequest.ID, requestUserTwoID, reviewedAt); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected rejection terminal-state conflict, got %v", err)
	}
}

func pendingRequest(
	id string,
	requesterID string,
	requesterName string,
	softwareProductID string,
	softwareProductName string,
	priority string,
	createdAt time.Time,
) Request {
	return Request{
		ID: id, RequesterID: requesterID, RequesterName: requesterName,
		SoftwareProductID: softwareProductID, SoftwareProductName: softwareProductName,
		Priority: priority, Reason: "Cần dùng cho công việc", Status: StatusPending,
		CreatedAt: createdAt, UpdatedAt: createdAt,
	}
}

func seedLicenseRequestReferences(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(t.Context(), `
		INSERT INTO users (id, email, password_hash, full_name, employee_code, role, status)
		VALUES
			($1, 'request-one@local.test', 'hash', 'Employee One', 'REQUEST-1', 'employee', 'active'),
			($2, 'request-two@local.test', 'hash', 'Employee Two', 'REQUEST-2', 'employee', 'active'),
			($3, 'request-admin@local.test', 'hash', 'Admin User', 'REQUEST-ADMIN', 'admin', 'active')
	`, requestUserOneID, requestUserTwoID, requestReviewerID); err != nil {
		t.Fatalf("seed license request users: %v", err)
	}
	if _, err := pool.Exec(t.Context(), `
		INSERT INTO software_products (id, name, publisher, version)
		VALUES
			($1, 'Adobe Photoshop', 'Adobe', ''),
			($2, 'Microsoft Office', 'Microsoft', '')
	`, requestProductOne, requestProductTwo); err != nil {
		t.Fatalf("seed license request software: %v", err)
	}
	if _, err := pool.Exec(t.Context(), `
		INSERT INTO licenses (
			id, software_product_id, name, license_type, assignment_type, seat_count, expires_at
		) VALUES ($1, $2, 'Adobe Business', 'subscription', 'user', 2, DATE '2099-01-01')
	`, requestLicenseOne, requestProductOne); err != nil {
		t.Fatalf("seed license request license: %v", err)
	}
}
