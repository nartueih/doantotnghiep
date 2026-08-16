package audit

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"license-manager/backend/internal/modules/auth"
	"license-manager/backend/internal/platform/id"
)

var ErrInvalidData = errors.New("audit action and entity type are required")

type Service struct {
	repository Repository
	actors     ActorFinder
	now        func() time.Time
}

type ActorFinder interface {
	FindByID(context.Context, string) (auth.User, error)
}

func NewService(repository Repository, actorFinders ...ActorFinder) *Service {
	service := &Service{repository: repository, now: time.Now}
	if len(actorFinders) > 0 {
		service.actors = actorFinders[0]
	}
	return service
}

func (s *Service) List(ctx context.Context, filter Filter) ([]Log, error) {
	filter.Action = strings.ToLower(strings.TrimSpace(filter.Action))
	filter.EntityType = strings.ToLower(strings.TrimSpace(filter.EntityType))
	filter.ActorID = strings.TrimSpace(filter.ActorID)
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Limit > 200 {
		filter.Limit = 200
	}
	items, err := s.repository.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	if s.actors != nil {
		for index := range items {
			if items[index].ActorID == "" || (items[index].ActorName != "" && items[index].ActorEmail != "") {
				continue
			}
			actor, findErr := s.actors.FindByID(ctx, items[index].ActorID)
			if findErr == nil {
				items[index].ActorName = actor.FullName
				items[index].ActorEmail = actor.Email
			}
		}
	}
	return items, nil
}

func (s *Service) Record(ctx context.Context, input RecordInput) (Log, error) {
	input.ActorID = strings.TrimSpace(input.ActorID)
	input.Action = strings.ToLower(strings.TrimSpace(input.Action))
	input.EntityType = strings.ToLower(strings.TrimSpace(input.EntityType))
	input.EntityID = strings.TrimSpace(input.EntityID)
	input.IPAddress = strings.TrimSpace(input.IPAddress)
	if input.Action == "" || input.EntityType == "" {
		return Log{}, ErrInvalidData
	}

	logID, err := id.NewUUID()
	if err != nil {
		return Log{}, fmt.Errorf("generate audit log id: %w", err)
	}
	return s.repository.Create(ctx, Log{
		ID:         logID,
		ActorID:    input.ActorID,
		Action:     input.Action,
		EntityType: input.EntityType,
		EntityID:   input.EntityID,
		Metadata:   sanitizeMetadata(input.Metadata),
		IPAddress:  input.IPAddress,
		CreatedAt:  s.now().UTC(),
	})
}

func sanitizeMetadata(metadata map[string]any) map[string]any {
	clean := make(map[string]any)
	for key, value := range metadata {
		if sensitiveKey(key) {
			continue
		}
		if nested, ok := value.(map[string]any); ok {
			clean[key] = sanitizeMetadata(nested)
			continue
		}
		clean[key] = value
	}
	return clean
}

func sensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(key), "-", "_"))
	for _, fragment := range []string{"password", "token", "secret", "license_key", "encrypted_key"} {
		if strings.Contains(normalized, fragment) {
			return true
		}
	}
	return false
}
