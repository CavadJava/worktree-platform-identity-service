package service

import (
	"context"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"platform-identity-service/internal/repository"
)

func newTestProjectService(t *testing.T) *ProjectService {
	t.Helper()
	db := testDB(t)
	return NewProjectService(repository.NewProjectRepository(db))
}

func TestProjectService_CreateAndGet(t *testing.T) {
	svc := newTestProjectService(t)

	p, err := svc.Create(context.Background(), "Widget Co")
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if p.ID == "" {
		t.Error("expected generated ID")
	}

	got, err := svc.Get(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Name != "Widget Co" {
		t.Errorf("expected name 'Widget Co', got %q", got.Name)
	}
}
