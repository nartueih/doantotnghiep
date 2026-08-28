package integration

import (
	"os"
	"testing"

	"license-manager/backend/internal/bootstrap"
	"license-manager/backend/internal/modules/assignments"
	"license-manager/backend/internal/modules/auth"
	"license-manager/backend/internal/modules/devices"
	"license-manager/backend/internal/modules/licenserequests"
	"license-manager/backend/internal/modules/licenses"
	"license-manager/backend/internal/modules/notifications"
	"license-manager/backend/internal/modules/software"
	"license-manager/backend/internal/modules/users"
	"license-manager/backend/internal/platform/database"
	"license-manager/backend/internal/platform/securevalue"
	"license-manager/backend/internal/testsupport"
)

func TestPostgresWorkflowSurvivesReconnect(t *testing.T) {
	pool := testsupport.OpenPostgres(t)
	userRepository := auth.NewPostgresRepository(pool)
	adminResult, err := bootstrap.EnsureAdmin(
		t.Context(),
		userRepository,
		auth.NewPasswordHasher(4),
		bootstrap.AdminInput{
			Email: "admin@local.test", Password: "ChangeMe123!",
			FullName: "Development Admin", EmployeeCode: "DEV-ADMIN",
		},
	)
	if err != nil {
		t.Fatalf("seed Admin: %v", err)
	}
	employee, err := users.NewService(userRepository, auth.NewPasswordHasher(4)).Create(
		t.Context(),
		users.CreateInput{
			Email: "employee@local.test", Password: "Employee123!",
			FullName: "PostgreSQL Employee", EmployeeCode: "PG-EMPLOYEE",
			Role: auth.RoleEmployee,
		},
	)
	if err != nil {
		t.Fatalf("create Employee: %v", err)
	}

	softwareRepository := software.NewPostgresRepository(pool)
	product, err := software.NewService(softwareRepository).Create(t.Context(), software.Input{
		Name: "PostgreSQL Test Product", Publisher: "License Manager",
	})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}
	cipher, err := securevalue.NewCipher("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")
	if err != nil {
		t.Fatal(err)
	}
	licenseRepository := licenses.NewPostgresRepository(pool)
	licenseItem, err := licenses.NewService(licenseRepository, softwareRepository, cipher).Create(
		t.Context(),
		licenses.Input{
			SoftwareProductID: product.ID, Name: "PostgreSQL Test License",
			LicenseType: licenses.TypeSubscription, AssignmentType: licenses.AssignmentUser,
			SeatCount: 2, ExpiresAt: "2099-01-01",
		},
	)
	if err != nil {
		t.Fatalf("create license: %v", err)
	}

	assignmentRepository := assignments.NewPostgresRepository(pool)
	assignmentService := assignments.NewService(
		assignmentRepository,
		licenseRepository,
		userRepository,
		devices.NewPostgresRepository(pool),
	)
	notificationRepository := notifications.NewPostgresRepository(pool)
	notificationService := notifications.NewService(notificationRepository)
	requestRepository := licenserequests.NewPostgresRepository(pool)
	requestService := licenserequests.NewService(
		requestRepository,
		softwareRepository,
		licenseRepository,
		userRepository,
		assignmentService,
		notificationService,
		database.NewPostgresTransactor(pool),
	)
	requestItem, err := requestService.Create(t.Context(), employee.ID, licenserequests.CreateInput{
		SoftwareProductID: product.ID,
		Priority:          licenserequests.PriorityHigh,
		Reason:            "Cần license cho công việc",
	})
	if err != nil {
		t.Fatalf("create license request: %v", err)
	}
	if _, err := requestService.Approve(t.Context(), adminResult.User.ID, requestItem.ID, licenserequests.ApproveInput{
		LicenseID: licenseItem.ID,
	}); err != nil {
		t.Fatalf("approve license request: %v", err)
	}
	pool.Close()

	reopened, err := database.Open(t.Context(), os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatalf("reopen PostgreSQL test database: %v", err)
	}
	t.Cleanup(reopened.Close)
	storedRequest, err := licenserequests.NewPostgresRepository(reopened).FindByID(t.Context(), requestItem.ID)
	if err != nil || storedRequest.Status != licenserequests.StatusApproved {
		t.Fatalf("stored request=%#v err=%v", storedRequest, err)
	}
	assignmentItems, err := assignments.NewPostgresRepository(reopened).List(t.Context())
	if err != nil || len(assignmentItems) != 1 || assignmentItems[0].UserID != employee.ID {
		t.Fatalf("stored assignments=%#v err=%v", assignmentItems, err)
	}
	reopenedNotifications := notifications.NewService(notifications.NewPostgresRepository(reopened))
	result, err := reopenedNotifications.List(t.Context(), employee.ID)
	if err != nil || result.Total != 1 || result.UnreadCount != 1 {
		t.Fatalf("stored notifications=%#v err=%v", result, err)
	}
	if _, err := reopenedNotifications.MarkRead(t.Context(), employee.ID, result.Items[0].ID); err != nil {
		t.Fatalf("mark notification read: %v", err)
	}
	afterRead, err := reopenedNotifications.List(t.Context(), employee.ID)
	if err != nil || afterRead.UnreadCount != 0 {
		t.Fatalf("notifications after read=%#v err=%v", afterRead, err)
	}
}
