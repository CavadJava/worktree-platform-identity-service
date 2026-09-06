package service

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"platform-identity-service/internal/auth"
	"platform-identity-service/internal/database"
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
	if _, err := db.Exec(`CREATE SCHEMA IF NOT EXISTS test_service`); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	if _, err := db.Exec(`SET search_path TO test_service`); err != nil {
		t.Fatalf("set search_path: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func newTestAuthService(t *testing.T) *AuthService {
	t.Helper()
	db := testDB(t)
	userRepo := repository.NewUserRepository(db)
	jwt := auth.NewJWTManager("test-secret", 60)
	return NewAuthService(userRepo, jwt)
}

func newTestJWTManager() *auth.JWTManager {
	return auth.NewJWTManager("test-secret", 60)
}
