package notifications

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestServiceListsOnlyCurrentUsersNotificationsNewestFirst(t *testing.T) {
	service := NewService(NewMemoryRepository())
	times := []time.Time{
		time.Date(2026, time.August, 23, 8, 0, 0, 0, time.UTC),
		time.Date(2026, time.August, 23, 9, 0, 0, 0, time.UTC),
		time.Date(2026, time.August, 23, 10, 0, 0, 0, time.UTC),
	}
	service.now = func() time.Time {
		current := times[0]
		times = times[1:]
		return current
	}

	first, err := service.Create(context.Background(), CreateInput{
		UserID: "user-1", Type: TypeLicenseRequestApproved, Title: "Đã duyệt",
		Message: "Adobe đã được cấp", EntityType: EntityLicenseRequest, EntityID: "request-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Create(context.Background(), CreateInput{
		UserID: "user-2", Type: TypeLicenseRequestRejected, Title: "Đã từ chối",
		Message: "Office tạm hết license", EntityType: EntityLicenseRequest, EntityID: "request-2",
	}); err != nil {
		t.Fatal(err)
	}
	latest, err := service.Create(context.Background(), CreateInput{
		UserID: "user-1", Type: TypeLicenseRequestRejected, Title: "Đã từ chối",
		Message: "Figma chưa được duyệt", EntityType: EntityLicenseRequest, EntityID: "request-3",
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := service.List(context.Background(), "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if result.Total != 2 || result.UnreadCount != 2 || len(result.Items) != 2 {
		t.Fatalf("unexpected counts: %#v", result)
	}
	if result.Items[0].ID != latest.ID || result.Items[1].ID != first.ID {
		t.Fatalf("expected newest first, got %#v", result.Items)
	}
	for _, item := range result.Items {
		if item.UserID != "user-1" {
			t.Fatalf("leaked another user's notification: %#v", item)
		}
	}
}

func TestServiceMarkReadHidesOtherUsersNotificationAndIsIdempotent(t *testing.T) {
	service := NewService(NewMemoryRepository())
	createdAt := time.Date(2026, time.August, 23, 8, 0, 0, 0, time.UTC)
	readAt := time.Date(2026, time.August, 23, 9, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return createdAt }
	item, err := service.Create(context.Background(), CreateInput{
		UserID: "user-1", Type: TypeLicenseRequestApproved, Title: "Đã duyệt",
		Message: "Adobe đã được cấp", EntityType: EntityLicenseRequest, EntityID: "request-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := service.MarkRead(context.Background(), "user-2", item.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected hidden ownership error, got %v", err)
	}

	service.now = func() time.Time { return readAt }
	first, err := service.MarkRead(context.Background(), "user-1", item.ID)
	if err != nil || first.ReadAt == nil || !first.ReadAt.Equal(readAt) {
		t.Fatalf("expected read notification at fixed time: %#v, %v", first, err)
	}
	service.now = func() time.Time { return readAt.Add(time.Hour) }
	second, err := service.MarkRead(context.Background(), "user-1", item.ID)
	if err != nil || second.ReadAt == nil || !second.ReadAt.Equal(readAt) {
		t.Fatalf("expected idempotent read timestamp: %#v, %v", second, err)
	}
}

func TestServiceMarkAllReadOnlyChangesCurrentUsersUnreadNotifications(t *testing.T) {
	service := NewService(NewMemoryRepository())
	createdAt := time.Date(2026, time.August, 23, 8, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return createdAt }
	for _, userID := range []string{"user-1", "user-1", "user-2"} {
		if _, err := service.Create(context.Background(), CreateInput{
			UserID: userID, Type: TypeLicenseRequestApproved, Title: "Đã duyệt",
			Message: "License đã được cấp", EntityType: EntityLicenseRequest, EntityID: "request-1",
		}); err != nil {
			t.Fatal(err)
		}
	}

	service.now = func() time.Time { return createdAt.Add(time.Hour) }
	updated, err := service.MarkAllRead(context.Background(), "user-1")
	if err != nil || updated != 2 {
		t.Fatalf("expected two updates, got %d, %v", updated, err)
	}
	userOne, _ := service.List(context.Background(), "user-1")
	userTwo, _ := service.List(context.Background(), "user-2")
	if userOne.UnreadCount != 0 || userTwo.UnreadCount != 1 {
		t.Fatalf("unexpected unread counts: user-1=%d user-2=%d", userOne.UnreadCount, userTwo.UnreadCount)
	}
}

func TestServiceRejectsIncompleteNotificationData(t *testing.T) {
	service := NewService(NewMemoryRepository())
	_, err := service.Create(context.Background(), CreateInput{UserID: "user-1", Title: "Thiếu dữ liệu"})
	if !errors.Is(err, ErrInvalidData) {
		t.Fatalf("expected invalid data, got %v", err)
	}
}
