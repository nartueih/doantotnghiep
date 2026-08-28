package notifications

import (
	"errors"
	"testing"
	"time"

	"license-manager/backend/internal/testsupport"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	notificationUserOneID = "20000000-0000-0000-0000-000000000001"
	notificationUserTwoID = "20000000-0000-0000-0000-000000000002"
)

func TestPostgresRepositoryScopesAndMarksNotifications(t *testing.T) {
	pool := testsupport.OpenPostgres(t)
	seedNotificationUsers(t, pool)
	repository := NewPostgresRepository(pool)

	createdAt := time.Date(2026, time.August, 24, 10, 0, 0, 0, time.UTC)
	first := Notification{
		ID:         "10000000-0000-0000-0000-000000000001",
		UserID:     notificationUserOneID,
		Type:       TypeLicenseRequestApproved,
		Title:      "Đã duyệt",
		Message:    "Đã cấp license",
		EntityType: EntityLicenseRequest,
		EntityID:   "30000000-0000-0000-0000-000000000001",
		CreatedAt:  createdAt,
	}
	second := Notification{
		ID:         "10000000-0000-0000-0000-000000000002",
		UserID:     notificationUserTwoID,
		Type:       TypeLicenseRequestRejected,
		Title:      "Đã phản hồi",
		Message:    "Tạm hết license",
		EntityType: EntityLicenseRequest,
		EntityID:   "30000000-0000-0000-0000-000000000002",
		CreatedAt:  createdAt.Add(time.Minute),
	}

	if _, err := repository.Create(t.Context(), first); err != nil {
		t.Fatalf("create first notification: %v", err)
	}
	if _, err := repository.Create(t.Context(), second); err != nil {
		t.Fatalf("create second notification: %v", err)
	}

	items, err := repository.ListByUser(t.Context(), notificationUserOneID)
	if err != nil || len(items) != 1 || items[0].ID != first.ID {
		t.Fatalf("items=%#v err=%v", items, err)
	}

	readAt := createdAt.Add(2 * time.Hour)
	read, err := repository.MarkRead(t.Context(), notificationUserOneID, first.ID, readAt)
	if err != nil || read.ReadAt == nil || !read.ReadAt.Equal(readAt) {
		t.Fatalf("read=%#v err=%v", read, err)
	}
	readAgain, err := repository.MarkRead(t.Context(), notificationUserOneID, first.ID, readAt.Add(time.Hour))
	if err != nil || readAgain.ReadAt == nil || !readAgain.ReadAt.Equal(readAt) {
		t.Fatalf("idempotent read=%#v err=%v", readAgain, err)
	}
	if _, err := repository.MarkRead(t.Context(), notificationUserTwoID, first.ID, readAt); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ownership-safe not found, got %v", err)
	}
}

func TestPostgresRepositoryMarksAllUnreadForOneUser(t *testing.T) {
	pool := testsupport.OpenPostgres(t)
	seedNotificationUsers(t, pool)
	repository := NewPostgresRepository(pool)
	createdAt := time.Date(2026, time.August, 24, 10, 0, 0, 0, time.UTC)

	for index, item := range []Notification{
		{ID: "10000000-0000-0000-0000-000000000011", UserID: notificationUserOneID},
		{ID: "10000000-0000-0000-0000-000000000012", UserID: notificationUserOneID},
		{ID: "10000000-0000-0000-0000-000000000013", UserID: notificationUserTwoID},
	} {
		item.Type = TypeLicenseRequestApproved
		item.Title = "Thông báo"
		item.Message = "License đã được xử lý"
		item.EntityType = EntityLicenseRequest
		item.EntityID = "30000000-0000-0000-0000-000000000011"
		item.CreatedAt = createdAt.Add(time.Duration(index) * time.Minute)
		if _, err := repository.Create(t.Context(), item); err != nil {
			t.Fatalf("create notification %d: %v", index, err)
		}
	}

	updated, err := repository.MarkAllRead(t.Context(), notificationUserOneID, createdAt.Add(time.Hour))
	if err != nil || updated != 2 {
		t.Fatalf("updated=%d err=%v", updated, err)
	}
	userOne, err := repository.ListByUser(t.Context(), notificationUserOneID)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range userOne {
		if item.ReadAt == nil {
			t.Fatalf("notification %s remains unread", item.ID)
		}
	}
	userTwo, err := repository.ListByUser(t.Context(), notificationUserTwoID)
	if err != nil || len(userTwo) != 1 || userTwo[0].ReadAt != nil {
		t.Fatalf("user two notifications=%#v err=%v", userTwo, err)
	}
}

func seedNotificationUsers(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(t.Context(), `
		INSERT INTO users (id, email, password_hash, full_name, employee_code, role, status)
		VALUES
			($1, 'notification-one@local.test', 'hash', 'Notification One', 'NOTIFY-1', 'employee', 'active'),
			($2, 'notification-two@local.test', 'hash', 'Notification Two', 'NOTIFY-2', 'employee', 'active')
	`, notificationUserOneID, notificationUserTwoID)
	if err != nil {
		t.Fatalf("seed notification users: %v", err)
	}
}
