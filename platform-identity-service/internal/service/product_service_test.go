package service

import (
	"context"
	"errors"
	"testing"

	"platform-identity-service/internal/models"
	"platform-identity-service/internal/repository"
	"platform-identity-service/internal/service/shopassign"
)

func newTestProductService(t *testing.T) *ProductService {
	t.Helper()
	db := testDB(t)
	productRepo := repository.NewProductRepository(db)
	subRepo := repository.NewSubscriptionRepository(db)
	subprojRepo := repository.NewSubprojectRepository(db)
	return NewProductService(productRepo, subRepo, subprojRepo)
}

func TestProductService_UpdateProfile_RequiresSuperadmin(t *testing.T) {
	svc := newTestProductService(t)
	p, err := svc.Create(context.Background(), "Teslahubs")
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	_, err = svc.UpdateProfile(context.Background(), shopassign.Caller{SystemRole: models.SystemRoleUser}, p.ID, "desc", "React")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for non-superadmin, got %v", err)
	}

	updated, err := svc.UpdateProfile(context.Background(), shopassign.Caller{SystemRole: models.SystemRoleSuperadmin}, p.ID, "A Tesla platform", "React, Go, Postgres")
	if err != nil {
		t.Fatalf("update profile: %v", err)
	}
	if updated.Description != "A Tesla platform" || updated.TechStack != "React, Go, Postgres" {
		t.Fatalf("expected updated profile, got %+v", updated)
	}
}

func TestProductService_SubprojectLifecycle(t *testing.T) {
	svc := newTestProductService(t)
	p, err := svc.Create(context.Background(), "Teslahubs")
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	superadmin := shopassign.Caller{SystemRole: models.SystemRoleSuperadmin}
	nonAdmin := shopassign.Caller{SystemRole: models.SystemRoleUser}

	if _, err := svc.AddSubproject(context.Background(), nonAdmin, p.ID, "auth-service", "handles login"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for non-superadmin add, got %v", err)
	}

	sub, err := svc.AddSubproject(context.Background(), superadmin, p.ID, "auth-service", "handles login")
	if err != nil {
		t.Fatalf("add subproject: %v", err)
	}

	list, err := svc.ListSubprojects(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("list subprojects: %v", err)
	}
	if len(list) != 1 || list[0].Name != "auth-service" {
		t.Fatalf("expected one subproject, got %+v", list)
	}

	if err := svc.RemoveSubproject(context.Background(), nonAdmin, sub.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for non-superadmin remove, got %v", err)
	}
	if err := svc.RemoveSubproject(context.Background(), superadmin, sub.ID); err != nil {
		t.Fatalf("remove subproject: %v", err)
	}

	list, err = svc.ListSubprojects(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("list subprojects after remove: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected zero subprojects after remove, got %d", len(list))
	}
}
