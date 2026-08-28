package audit

import (
	"context"
	"errors"
	"testing"
	"time"

	"license-manager/backend/internal/platform/database"
	"license-manager/backend/internal/testsupport"
)

func TestPostgresAuditCreateParticipatesInCallerTransaction(t *testing.T) {
	pool := testsupport.OpenPostgres(t)
	repository := NewPostgresRepository(pool)
	rollbackError := errors.New("force rollback")
	err := database.NewPostgresTransactor(pool).WithinTransaction(t.Context(), func(txCtx context.Context) error {
		_, createErr := repository.Create(txCtx, Log{
			ID: "55000000-0000-0000-0000-000000000001", Action: ActionCreate,
			EntityType: EntityMaintenanceRequest, Metadata: map[string]any{"status": "pending"}, CreatedAt: time.Now().UTC(),
		})
		if createErr != nil {
			return createErr
		}
		return rollbackError
	})
	if !errors.Is(err, rollbackError) {
		t.Fatalf("expected rollback error, got %v", err)
	}
	items, err := repository.List(t.Context(), Filter{Limit: 10})
	if err != nil || len(items) != 0 {
		t.Fatalf("audit transaction did not roll back: items=%#v err=%v", items, err)
	}
}
