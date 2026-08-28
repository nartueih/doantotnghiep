package licenserequests

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	"license-manager/backend/internal/modules/assignments"
	"license-manager/backend/internal/modules/auth"
	"license-manager/backend/internal/modules/devices"
	"license-manager/backend/internal/modules/licenses"
	"license-manager/backend/internal/modules/notifications"
	"license-manager/backend/internal/modules/software"
	"license-manager/backend/internal/platform/database"
	"license-manager/backend/internal/testsupport"
)

func TestPostgresApproveRollsBackAssignmentWhenNotificationFails(t *testing.T) {
	fixture := newPostgresWorkflowFixture(t)
	notificationError := errors.New("notification unavailable")
	service := fixture.service(failingNotificationCreator{err: notificationError})
	item := createPostgresLicenseRequest(t, service)

	_, err := service.Approve(t.Context(), requestReviewerID, item.ID, ApproveInput{
		LicenseID: requestLicenseOne,
	})
	if !errors.Is(err, notificationError) {
		t.Fatalf("expected notification error, got %v", err)
	}

	stored, err := fixture.requests.FindByID(t.Context(), item.ID)
	if err != nil || stored.Status != StatusPending {
		t.Fatalf("request after rollback=%#v err=%v", stored, err)
	}
	assignmentItems, err := fixture.assignments.List(t.Context())
	if err != nil || len(assignmentItems) != 0 {
		t.Fatalf("assignments after rollback=%#v err=%v", assignmentItems, err)
	}
	notificationItems, err := fixture.notifications.ListByUser(t.Context(), requestUserOneID)
	if err != nil || len(notificationItems) != 0 {
		t.Fatalf("notifications after rollback=%#v err=%v", notificationItems, err)
	}
}

func TestPostgresConcurrentApproveCreatesExactlyOneAssignment(t *testing.T) {
	fixture := newPostgresWorkflowFixture(t)
	firstService := fixture.service(fixture.notificationService)
	secondService := fixture.service(fixture.notificationService)
	item := createPostgresLicenseRequest(t, firstService)

	start := make(chan struct{})
	results := make(chan error, 2)
	var waitGroup sync.WaitGroup
	for _, service := range []*Service{firstService, secondService} {
		waitGroup.Add(1)
		go func(service *Service) {
			defer waitGroup.Done()
			<-start
			_, err := service.Approve(context.Background(), requestReviewerID, item.ID, ApproveInput{
				LicenseID: requestLicenseOne,
			})
			results <- err
		}(service)
	}
	close(start)
	waitGroup.Wait()
	close(results)

	successes := 0
	conflicts := 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrInvalidState):
			conflicts++
		default:
			t.Fatalf("unexpected approval error: %v", err)
		}
	}
	assignmentItems, err := fixture.assignments.List(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	notificationItems, err := fixture.notifications.ListByUser(t.Context(), requestUserOneID)
	if err != nil {
		t.Fatal(err)
	}
	if successes != 1 || conflicts != 1 || len(assignmentItems) != 1 || len(notificationItems) != 1 {
		t.Fatalf(
			"successes=%d conflicts=%d assignments=%d notifications=%d",
			successes,
			conflicts,
			len(assignmentItems),
			len(notificationItems),
		)
	}
}

func TestPostgresRejectCommitsRequestAndNotificationTogether(t *testing.T) {
	fixture := newPostgresWorkflowFixture(t)
	service := fixture.service(fixture.notificationService)
	item := createPostgresLicenseRequest(t, service)

	if _, err := service.Reject(t.Context(), requestReviewerID, item.ID, RejectInput{
		DecisionReason: DecisionOutOfStock,
		ResponseNote:   "Tạm hết license",
	}); err != nil {
		t.Fatalf("reject request: %v", err)
	}
	fixture.pool.Close()

	reopened, err := database.Open(t.Context(), os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatalf("reopen PostgreSQL test database: %v", err)
	}
	t.Cleanup(reopened.Close)
	reopenedRequest, err := NewPostgresRepository(reopened).FindByID(t.Context(), item.ID)
	if err != nil || reopenedRequest.Status != StatusRejected {
		t.Fatalf("reopened request=%#v err=%v", reopenedRequest, err)
	}
	reopenedNotifications, err := notifications.NewPostgresRepository(reopened).ListByUser(t.Context(), requestUserOneID)
	if err != nil || len(reopenedNotifications) != 1 || reopenedNotifications[0].Type != notifications.TypeLicenseRequestRejected {
		t.Fatalf("reopened notifications=%#v err=%v", reopenedNotifications, err)
	}
}

type postgresWorkflowFixture struct {
	pool                databasePool
	requests            *PostgresRepository
	assignments         *assignments.PostgresRepository
	notifications       *notifications.PostgresRepository
	notificationService *notifications.Service
	software            *software.PostgresRepository
	licenses            *licenses.PostgresRepository
	users               *auth.PostgresRepository
	assignmentService   *assignments.Service
	transactions        database.Transactor
}

type databasePool interface {
	Close()
}

func newPostgresWorkflowFixture(t *testing.T) postgresWorkflowFixture {
	pool := testsupport.OpenPostgres(t)
	seedLicenseRequestReferences(t, pool)
	requestRepository := NewPostgresRepository(pool)
	assignmentRepository := assignments.NewPostgresRepository(pool)
	notificationRepository := notifications.NewPostgresRepository(pool)
	softwareRepository := software.NewPostgresRepository(pool)
	licenseRepository := licenses.NewPostgresRepository(pool)
	userRepository := auth.NewPostgresRepository(pool)
	assignmentService := assignments.NewService(
		assignmentRepository,
		licenseRepository,
		userRepository,
		devices.NewPostgresRepository(pool),
	)
	notificationService := notifications.NewService(notificationRepository)
	return postgresWorkflowFixture{
		pool: pool, requests: requestRepository, assignments: assignmentRepository,
		notifications: notificationRepository, notificationService: notificationService,
		software: softwareRepository, licenses: licenseRepository, users: userRepository,
		assignmentService: assignmentService, transactions: database.NewPostgresTransactor(pool),
	}
}

func (f postgresWorkflowFixture) service(notificationCreator NotificationCreator) *Service {
	return NewService(
		f.requests,
		f.software,
		f.licenses,
		f.users,
		f.assignmentService,
		notificationCreator,
		f.transactions,
	)
}

func createPostgresLicenseRequest(t *testing.T, service *Service) Request {
	t.Helper()
	item, err := service.Create(t.Context(), requestUserOneID, CreateInput{
		SoftwareProductID: requestProductOne,
		Priority:          PriorityUrgent,
		Reason:            "Cần dùng cho dự án",
	})
	if err != nil {
		t.Fatalf("create license request: %v", err)
	}
	return item
}

type failingNotificationCreator struct {
	err error
}

func (c failingNotificationCreator) Create(context.Context, notifications.CreateInput) (notifications.Notification, error) {
	return notifications.Notification{}, c.err
}
