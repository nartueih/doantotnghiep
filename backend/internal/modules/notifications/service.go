package notifications

import (
	"context"
	"strings"
	"time"

	"license-manager/backend/internal/platform/id"
)

type CreateInput struct {
	UserID     string
	Type       string
	Title      string
	Message    string
	EntityType string
	EntityID   string
}

type ListResult struct {
	Items       []Notification `json:"items"`
	Total       int            `json:"total"`
	UnreadCount int            `json:"unread_count"`
}

type Service struct {
	repository Repository
	now        func() time.Time
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository, now: time.Now}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (Notification, error) {
	input.UserID = strings.TrimSpace(input.UserID)
	input.Type = strings.TrimSpace(input.Type)
	input.Title = strings.TrimSpace(input.Title)
	input.Message = strings.TrimSpace(input.Message)
	input.EntityType = strings.TrimSpace(input.EntityType)
	input.EntityID = strings.TrimSpace(input.EntityID)
	if input.UserID == "" || input.Type == "" || input.Title == "" || input.Message == "" || input.EntityType == "" || input.EntityID == "" {
		return Notification{}, ErrInvalidData
	}

	notificationID, err := id.NewUUID()
	if err != nil {
		return Notification{}, err
	}
	return s.repository.Create(ctx, Notification{
		ID:         notificationID,
		UserID:     input.UserID,
		Type:       input.Type,
		Title:      input.Title,
		Message:    input.Message,
		EntityType: input.EntityType,
		EntityID:   input.EntityID,
		CreatedAt:  s.now().UTC(),
	})
}

func (s *Service) List(ctx context.Context, userID string) (ListResult, error) {
	items, err := s.repository.ListByUser(ctx, strings.TrimSpace(userID))
	if err != nil {
		return ListResult{}, err
	}
	unreadCount := 0
	for _, item := range items {
		if item.ReadAt == nil {
			unreadCount++
		}
	}
	return ListResult{Items: items, Total: len(items), UnreadCount: unreadCount}, nil
}

func (s *Service) MarkRead(ctx context.Context, userID, notificationID string) (Notification, error) {
	return s.repository.MarkRead(ctx, strings.TrimSpace(userID), strings.TrimSpace(notificationID), s.now().UTC())
}

func (s *Service) MarkAllRead(ctx context.Context, userID string) (int, error) {
	return s.repository.MarkAllRead(ctx, strings.TrimSpace(userID), s.now().UTC())
}
