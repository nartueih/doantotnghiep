package audit

import (
	"context"
	"testing"

	"license-manager/backend/internal/modules/auth"
)

func TestRecordRemovesSensitiveMetadata(t *testing.T) {
	repository := NewMemoryRepository()
	service := NewService(repository)

	created, err := service.Record(context.Background(), RecordInput{
		ActorID:    "actor-id",
		Action:     " CREATE ",
		EntityType: " LICENSE ",
		EntityID:   "entity-id",
		Metadata: map[string]any{
			"name":          "Photoshop Teams",
			"license_key":   "TOP-SECRET",
			"refresh-token": "TOKEN",
			"nested": map[string]any{
				"password": "Password123",
				"safe":     true,
			},
		},
	})
	if err != nil {
		t.Fatalf("record audit log: %v", err)
	}
	if created.Action != ActionCreate || created.EntityType != EntityLicense {
		t.Fatalf("audit fields were not normalized: %#v", created)
	}
	if _, exists := created.Metadata["license_key"]; exists {
		t.Fatal("license_key was retained in audit metadata")
	}
	if _, exists := created.Metadata["refresh-token"]; exists {
		t.Fatal("refresh token was retained in audit metadata")
	}
	nested := created.Metadata["nested"].(map[string]any)
	if _, exists := nested["password"]; exists || nested["safe"] != true {
		t.Fatalf("nested metadata was not sanitized: %#v", nested)
	}
}

func TestListFiltersAndEnrichesActor(t *testing.T) {
	repository := NewMemoryRepository()
	actorRepository := auth.NewMemoryRepository([]auth.User{{
		ID: "actor-id", Email: "admin@example.com", FullName: "Admin User",
	}})
	service := NewService(repository, actorRepository)
	ctx := context.Background()
	if _, err := service.Record(ctx, RecordInput{ActorID: "actor-id", Action: ActionCreate, EntityType: EntityDevice}); err != nil {
		t.Fatalf("record device audit: %v", err)
	}
	if _, err := service.Record(ctx, RecordInput{ActorID: "actor-id", Action: ActionUpdate, EntityType: EntityLicense}); err != nil {
		t.Fatalf("record license audit: %v", err)
	}

	items, err := service.List(ctx, Filter{EntityType: " LICENSE ", Limit: 10})
	if err != nil {
		t.Fatalf("list audit logs: %v", err)
	}
	if len(items) != 1 || items[0].Action != ActionUpdate {
		t.Fatalf("unexpected filtered items: %#v", items)
	}
	if items[0].ActorName != "Admin User" || items[0].ActorEmail != "admin@example.com" {
		t.Fatalf("actor was not enriched: %#v", items[0])
	}
}
