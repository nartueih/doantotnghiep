package assignments

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"license-manager/backend/internal/modules/auth"
	"license-manager/backend/internal/modules/devices"
	"license-manager/backend/internal/modules/licenses"
	"license-manager/backend/internal/modules/software"
	"license-manager/backend/internal/platform/securevalue"
)

func TestAssignUserConsumesSeatAndRejectsDuplicate(t *testing.T) {
	fixture := newAssignmentFixture(t, 2, licenses.AssignmentUser)
	ctx := context.Background()

	created, err := fixture.service.Create(ctx, fixture.admin.ID, CreateInput{
		LicenseID: fixture.license.ID,
		UserID:    fixture.employee1.ID,
	})
	if err != nil {
		t.Fatalf("create assignment: %v", err)
	}
	if created.TargetName != fixture.employee1.FullName || created.Status != StatusActive {
		t.Fatalf("unexpected assignment: %#v", created)
	}
	updatedLicense, err := fixture.licenseRepository.FindByID(ctx, fixture.license.ID)
	if err != nil {
		t.Fatalf("find license: %v", err)
	}
	if updatedLicense.UsedSeats != 1 || updatedLicense.AvailableSeats != 1 {
		t.Fatalf("unexpected seat counts: used=%d available=%d", updatedLicense.UsedSeats, updatedLicense.AvailableSeats)
	}

	_, err = fixture.service.Create(ctx, fixture.admin.ID, CreateInput{
		LicenseID: fixture.license.ID,
		UserID:    fixture.employee1.ID,
	})
	if !errors.Is(err, ErrDuplicate) {
		t.Fatalf("expected ErrDuplicate, got %v", err)
	}
}

func TestConcurrentAssignmentsCannotExceedSeatCount(t *testing.T) {
	fixture := newAssignmentFixture(t, 1, licenses.AssignmentUser)
	ctx := context.Background()
	inputs := []CreateInput{
		{LicenseID: fixture.license.ID, UserID: fixture.employee1.ID},
		{LicenseID: fixture.license.ID, UserID: fixture.employee2.ID},
	}

	var waitGroup sync.WaitGroup
	errorsChannel := make(chan error, len(inputs))
	for _, input := range inputs {
		waitGroup.Add(1)
		go func(input CreateInput) {
			defer waitGroup.Done()
			_, err := fixture.service.Create(ctx, fixture.admin.ID, input)
			errorsChannel <- err
		}(input)
	}
	waitGroup.Wait()
	close(errorsChannel)

	var successes, noSeats int
	for err := range errorsChannel {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrNoAvailableSeats):
			noSeats++
		default:
			t.Fatalf("unexpected assignment error: %v", err)
		}
	}
	if successes != 1 || noSeats != 1 {
		t.Fatalf("expected one success and one no-seat error, got successes=%d noSeats=%d", successes, noSeats)
	}
}

func TestRevokeReleasesSeatAndPreservesHistory(t *testing.T) {
	fixture := newAssignmentFixture(t, 1, licenses.AssignmentUser)
	ctx := context.Background()
	created, err := fixture.service.Create(ctx, fixture.admin.ID, CreateInput{
		LicenseID: fixture.license.ID, UserID: fixture.employee1.ID,
	})
	if err != nil {
		t.Fatalf("create assignment: %v", err)
	}

	revoked, err := fixture.service.Revoke(ctx, fixture.admin.ID, created.ID)
	if err != nil {
		t.Fatalf("revoke assignment: %v", err)
	}
	if revoked.Status != StatusRevoked || revoked.RevokedAt == nil || revoked.RevokedBy != fixture.admin.ID {
		t.Fatalf("unexpected revoked assignment: %#v", revoked)
	}
	item, _ := fixture.licenseRepository.FindByID(ctx, fixture.license.ID)
	if item.UsedSeats != 0 || item.AvailableSeats != 1 {
		t.Fatalf("seat was not released: used=%d available=%d", item.UsedSeats, item.AvailableSeats)
	}
	items, err := fixture.service.List(ctx)
	if err != nil || len(items) != 1 || items[0].Status != StatusRevoked {
		t.Fatalf("revoked history was not preserved: items=%#v error=%v", items, err)
	}

	if _, err := fixture.service.Create(ctx, fixture.admin.ID, CreateInput{
		LicenseID: fixture.license.ID, UserID: fixture.employee1.ID,
	}); err != nil {
		t.Fatalf("target should be assignable again after revoke: %v", err)
	}
}

func TestAssignmentTypeIsEnforced(t *testing.T) {
	fixture := newAssignmentFixture(t, 1, licenses.AssignmentDevice)

	_, err := fixture.service.Create(context.Background(), fixture.admin.ID, CreateInput{
		LicenseID: fixture.license.ID,
		UserID:    fixture.employee1.ID,
	})
	if !errors.Is(err, ErrAssignmentType) {
		t.Fatalf("expected ErrAssignmentType, got %v", err)
	}
}

func TestExactlyOneTargetIsRequired(t *testing.T) {
	fixture := newAssignmentFixture(t, 1, licenses.AssignmentMixed)

	_, err := fixture.service.Create(context.Background(), fixture.admin.ID, CreateInput{
		LicenseID: fixture.license.ID,
		UserID:    fixture.employee1.ID,
		DeviceID:  fixture.device.ID,
	})
	if !errors.Is(err, ErrInvalidTarget) {
		t.Fatalf("expected ErrInvalidTarget, got %v", err)
	}
}

func TestArchivedLicenseCannotBeAssigned(t *testing.T) {
	fixture := newAssignmentFixture(t, 1, licenses.AssignmentUser)
	ctx := context.Background()
	if _, err := fixture.licenseRepository.Archive(ctx, fixture.license.ID, time.Now()); err != nil {
		t.Fatalf("archive license: %v", err)
	}

	_, err := fixture.service.Create(ctx, fixture.admin.ID, CreateInput{
		LicenseID: fixture.license.ID,
		UserID:    fixture.employee1.ID,
	})
	if !errors.Is(err, ErrLicenseInactive) {
		t.Fatalf("expected ErrLicenseInactive, got %v", err)
	}
}

type assignmentFixture struct {
	service           *Service
	licenseRepository *licenses.MemoryRepository
	license           licenses.License
	device            devices.Device
	admin             auth.User
	employee1         auth.User
	employee2         auth.User
}

func newAssignmentFixture(t *testing.T, seats int, assignmentType string) assignmentFixture {
	t.Helper()
	admin := auth.User{ID: "admin-id", Email: "admin@example.com", FullName: "Admin User", Role: auth.RoleAdmin, Status: auth.StatusActive}
	employee1 := auth.User{ID: "employee-1", Email: "one@example.com", FullName: "Employee One", Role: auth.RoleEmployee, Status: auth.StatusActive}
	employee2 := auth.User{ID: "employee-2", Email: "two@example.com", FullName: "Employee Two", Role: auth.RoleEmployee, Status: auth.StatusActive}
	userRepository := auth.NewMemoryRepository([]auth.User{admin, employee1, employee2})

	softwareRepository := software.NewMemoryRepository()
	product, err := software.NewService(softwareRepository).Create(context.Background(), software.Input{Name: "Office", Publisher: "Microsoft"})
	if err != nil {
		t.Fatalf("create software: %v", err)
	}
	cipher, err := securevalue.NewCipher("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")
	if err != nil {
		t.Fatalf("create cipher: %v", err)
	}
	licenseRepository := licenses.NewMemoryRepository()
	license, err := licenses.NewService(licenseRepository, softwareRepository, cipher).Create(context.Background(), licenses.Input{
		SoftwareProductID: product.ID,
		Name:              "Office License",
		LicenseType:       licenses.TypeSubscription,
		AssignmentType:    assignmentType,
		SeatCount:         seats,
		ExpiresAt:         "2099-01-01",
	})
	if err != nil {
		t.Fatalf("create license: %v", err)
	}

	deviceRepository := devices.NewMemoryRepository()
	device, err := devices.NewService(deviceRepository, userRepository).Create(context.Background(), devices.Input{
		AssetCode: "LAP-001", Name: "Laptop", DeviceType: "laptop",
	})
	if err != nil {
		t.Fatalf("create device: %v", err)
	}
	assignmentRepository := NewMemoryRepository(licenseRepository)
	return assignmentFixture{
		service:           NewService(assignmentRepository, licenseRepository, userRepository, deviceRepository),
		licenseRepository: licenseRepository,
		license:           license,
		device:            device,
		admin:             admin,
		employee1:         employee1,
		employee2:         employee2,
	}
}
