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
		dsn = "host=localhost port=5433 user=postgres password=1 dbname=postgres sslmode=disable"
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func newAuthAndProjectServiceFromRepos(t *testing.T, projectRepo *repository.ProjectRepository, userRepo *repository.UserRepository) (*AuthService, *ProjectService) {
	t.Helper()
	jwt := auth.NewJWTManager("test-secret", 60)
	return NewAuthService(projectRepo, userRepo, jwt), NewProjectService(projectRepo)
}
