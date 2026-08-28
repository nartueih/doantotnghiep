package maintenancerequests

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"license-manager/backend/internal/modules/audit"
	"license-manager/backend/internal/modules/auth"
	"license-manager/backend/internal/modules/devices"
	"license-manager/backend/internal/modules/notifications"
	"license-manager/backend/internal/platform/database"
	"license-manager/backend/internal/testsupport"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresCreateAndAuditUseSingleConnectionTransaction(t *testing.T) {
	setupPool := testsupport.OpenPostgres(t)
	seedMaintenanceReferences(t, setupPool)
	config, err := pgxpool.ParseConfig(os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	config.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(t.Context(), config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	requestRepository := NewPostgresRepository(pool)
	service := NewService(requestRepository, devices.NewPostgresRepository(pool), auth.NewPostgresRepository(pool), notifications.NewService(notifications.NewPostgresRepository(pool)), database.NewPostgresTransactor(pool))
	auditService := audit.NewService(audit.NewPostgresRepository(pool))
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	err = database.NewPostgresTransactor(pool).WithinTransaction(ctx, func(txCtx context.Context) error {
		item, createErr := service.Create(txCtx, maintenanceEmployeeID, CreateInput{DeviceID: maintenanceDeviceID, Category: CategoryHardware, Priority: PriorityNormal, Title: "Kiểm tra một kết nối", Description: "Yêu cầu phải dùng cùng transaction."})
		if createErr != nil {
			return createErr
		}
		_, createErr = auditService.Record(txCtx, audit.RecordInput{ActorID: maintenanceEmployeeID, Action: audit.ActionRequest, EntityType: audit.EntityMaintenanceRequest, EntityID: item.ID})
		return createErr
	})
	if err != nil {
		t.Fatalf("single-connection transaction failed: %v", err)
	}
}

func TestPostgresAcceptRollsBackWhenNotificationFails(t *testing.T) {
	pool := testsupport.OpenPostgres(t)
	seedMaintenanceReferences(t, pool)
	requests := NewPostgresRepository(pool)
	notificationRepository := notifications.NewPostgresRepository(pool)
	notificationError := errors.New("notification unavailable")
	service := NewService(requests, devices.NewPostgresRepository(pool), auth.NewPostgresRepository(pool), failingMaintenanceNotificationCreator{err: notificationError}, database.NewPostgresTransactor(pool))

	created, err := service.Create(t.Context(), maintenanceEmployeeID, CreateInput{DeviceID: maintenanceDeviceID, Category: CategoryHardware, Priority: PriorityHigh, Title: "Máy không khởi động", Description: "Không phản hồi khi nhấn nút nguồn"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Accept(t.Context(), maintenanceAdminID, created.ID); !errors.Is(err, notificationError) {
		t.Fatalf("expected notification error, got %v", err)
	}

	items, err := requests.ListByRequester(t.Context(), maintenanceEmployeeID)
	if err != nil || len(items) != 1 || items[0].Status != StatusPending {
		t.Fatalf("requests=%#v err=%v", items, err)
	}
	notificationItems, err := notificationRepository.ListByUser(t.Context(), maintenanceEmployeeID)
	if err != nil || len(notificationItems) != 0 {
		t.Fatalf("notifications=%#v err=%v", notificationItems, err)
	}
}

func TestPostgresConcurrentAcceptCommitsExactlyOnce(t *testing.T) {
	pool := testsupport.OpenPostgres(t)
	seedMaintenanceReferences(t, pool)
	requests := NewPostgresRepository(pool)
	users := auth.NewPostgresRepository(pool)
	deviceRepository := devices.NewPostgresRepository(pool)
	notificationRepository := notifications.NewPostgresRepository(pool)
	notificationService := notifications.NewService(notificationRepository)
	transactor := database.NewPostgresTransactor(pool)
	first := NewService(requests, deviceRepository, users, notificationService, transactor)
	second := NewService(requests, deviceRepository, users, notificationService, transactor)
	created, err := first.Create(t.Context(), maintenanceEmployeeID, CreateInput{DeviceID: maintenanceDeviceID, Category: CategoryHardware, Priority: PriorityUrgent, Title: "Màn hình nhấp nháy", Description: "Màn hình mất tín hiệu liên tục"})
	if err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	results := make(chan error, 2)
	var group sync.WaitGroup
	for _, service := range []*Service{first, second} {
		group.Add(1)
		go func(current *Service) {
			defer group.Done()
			<-start
			_, err := current.Accept(context.Background(), maintenanceAdminID, created.ID)
			results <- err
		}(service)
	}
	close(start)
	group.Wait()
	close(results)
	successes, conflicts := 0, 0
	for err := range results {
		if err == nil {
			successes++
		} else if errors.Is(err, ErrInvalidState) {
			conflicts++
		} else {
			t.Fatalf("unexpected accept error: %v", err)
		}
	}
	notificationItems, err := notificationRepository.ListByUser(t.Context(), maintenanceEmployeeID)
	if err != nil {
		t.Fatal(err)
	}
	if successes != 1 || conflicts != 1 || len(notificationItems) != 1 {
		t.Fatalf("successes=%d conflicts=%d notifications=%d", successes, conflicts, len(notificationItems))
	}
}

type failingMaintenanceNotificationCreator struct{ err error }

func (creator failingMaintenanceNotificationCreator) Create(context.Context, notifications.CreateInput) (notifications.Notification, error) {
	return notifications.Notification{}, creator.err
}
