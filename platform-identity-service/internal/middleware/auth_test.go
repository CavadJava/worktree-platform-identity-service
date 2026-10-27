package middleware

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"

	"platform-identity-service/internal/auth"
	"platform-identity-service/internal/database"
	"platform-identity-service/internal/models"
	"platform-identity-service/internal/repository"
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DSN")
	if dsn == "" {
		dsn = "host=localhost port=5433 user=postgres password=1 dbname=platform_identity sslmode=disable"
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if _, err := db.Exec(`CREATE SCHEMA IF NOT EXISTS test_middleware`); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	if _, err := db.Exec(`SET search_path TO test_middleware`); err != nil {
		t.Fatalf("set search_path: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestRequireAuth_RejectsRevokedToken(t *testing.T) {
	db := testDB(t)
	userRepo := repository.NewUserRepository(db)
	revocationRepo := repository.NewTokenRevocationRepository(db)
	jwtManager := auth.NewJWTManager("middleware-test-secret", 60)

	now := time.Now().UTC()
	u := &models.User{
		ID: uuid.NewString(), Name: "Revoke Me", Username: "revoke-" + uuid.NewString(),
		Email: uuid.NewString() + "@example.com", PasswordHash: "hash",
		SystemRoleID: 3, Status: models.UserStatusActive, CreatedAt: now, UpdatedAt: now,
	}
	if err := userRepo.Create(context.Background(), u); err != nil {
		t.Fatalf("create user: %v", err)
	}

	token, _, err := jwtManager.Generate(u.ID, "user")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	claims, err := jwtManager.Verify(token)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}

	var reached bool
	handler := RequireAuth(jwtManager, userRepo, revocationRepo)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	}))

	// Sanity check: the token works before it is revoked.
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 before revocation, got %d", rec.Code)
	}
	if !reached {
		t.Fatal("expected the protected handler to run before revocation")
	}

	if err := revocationRepo.Revoke(req.Context(), claims.JTI, claims.ExpiresAt.Time); err != nil {
		t.Fatalf("revoke: %v", err)
	}

	reached = false
	req = httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 after revocation, got %d: %s", rec.Code, rec.Body.String())
	}
	if reached {
		t.Fatal("expected the protected handler NOT to run after revocation")
	}
}
