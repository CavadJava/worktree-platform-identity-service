package repository

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"

	"platform-identity-service/internal/database"
	"platform-identity-service/internal/models"
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
	return db
}

func TestProjectRepository_CreateAndGet(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	repo := NewProjectRepository(db)

	p := &models.Project{ID: uuid.NewString(), Name: "Test Project " + uuid.NewString(), CreatedAt: time.Now().UTC()}
	if err := repo.Create(context.Background(), p); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := repo.GetByID(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.Name != p.Name {
		t.Errorf("expected name %q, got %q", p.Name, got.Name)
	}
}

func TestProjectRepository_GetByID_NotFound(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	repo := NewProjectRepository(db)

	_, err := repo.GetByID(context.Background(), uuid.NewString())
	if err != ErrProjectNotFound {
		t.Errorf("expected ErrProjectNotFound, got %v", err)
	}
}

func TestUserRepository_CreateAndCount(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	projectRepo := NewProjectRepository(db)
	userRepo := NewUserRepository(db)

	p := &models.Project{ID: uuid.NewString(), Name: "Count Test", CreatedAt: time.Now().UTC()}
	if err := projectRepo.Create(context.Background(), p); err != nil {
		t.Fatalf("Create project failed: %v", err)
	}

	count, err := userRepo.CountByProject(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("CountByProject failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 users for new project, got %d", count)
	}

	roleID := int16(2) // admin, seeded second
	u := &models.User{
		ID: uuid.NewString(), Name: "Cavad", Username: "cavad-" + uuid.NewString(),
		Email: uuid.NewString() + "@example.com", PasswordHash: "hash",
		ProjectID: &p.ID, RoleID: &roleID,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := userRepo.Create(context.Background(), u); err != nil {
		t.Fatalf("Create user failed: %v", err)
	}

	count, err = userRepo.CountByProject(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("CountByProject failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 user after create, got %d", count)
	}

	got, err := userRepo.GetByID(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.RoleName != "admin" {
		t.Errorf("expected joined RoleName 'admin', got %q", got.RoleName)
	}
}

func TestUserRepository_SetRole(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	projectRepo := NewProjectRepository(db)
	userRepo := NewUserRepository(db)

	p := &models.Project{ID: uuid.NewString(), Name: "SetRole Test", CreatedAt: time.Now().UTC()}
	_ = projectRepo.Create(context.Background(), p)

	roleID := int16(1) // user
	u := &models.User{
		ID: uuid.NewString(), Name: "Test", Username: "u-" + uuid.NewString(),
		Email: uuid.NewString() + "@example.com", PasswordHash: "hash",
		ProjectID: &p.ID, RoleID: &roleID,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	_ = userRepo.Create(context.Background(), u)

	if err := userRepo.SetRole(context.Background(), u.ID, 2); err != nil {
		t.Fatalf("SetRole failed: %v", err)
	}

	got, err := userRepo.GetByID(context.Background(), u.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.RoleName != "admin" {
		t.Errorf("expected role promoted to admin, got %q", got.RoleName)
	}
}

func TestUserRepository_GetByUsernameOrEmail(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	projectRepo := NewProjectRepository(db)
	userRepo := NewUserRepository(db)

	// Create a test project
	p := &models.Project{ID: uuid.NewString(), Name: "GetByUsernameOrEmail Test", CreatedAt: time.Now().UTC()}
	if err := projectRepo.Create(context.Background(), p); err != nil {
		t.Fatalf("Create project failed: %v", err)
	}

	// Create a test user
	roleID := int16(2) // admin
	testUsername := "testuser-" + uuid.NewString()
	testEmail := uuid.NewString() + "@example.com"
	u := &models.User{
		ID: uuid.NewString(), Name: "Test User", Username: testUsername,
		Email: testEmail, PasswordHash: "hash",
		ProjectID: &p.ID, RoleID: &roleID,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := userRepo.Create(context.Background(), u); err != nil {
		t.Fatalf("Create user failed: %v", err)
	}

	// Test 1: Find by username
	got, err := userRepo.GetByUsernameOrEmail(context.Background(), testUsername)
	if err != nil {
		t.Fatalf("GetByUsernameOrEmail with username failed: %v", err)
	}
	if got.ID != u.ID {
		t.Errorf("expected user ID %q, got %q", u.ID, got.ID)
	}
	if got.Username != testUsername {
		t.Errorf("expected username %q, got %q", testUsername, got.Username)
	}
	if got.RoleName != "admin" {
		t.Errorf("expected RoleName 'admin', got %q", got.RoleName)
	}

	// Test 2: Find by email
	got, err = userRepo.GetByUsernameOrEmail(context.Background(), testEmail)
	if err != nil {
		t.Fatalf("GetByUsernameOrEmail with email failed: %v", err)
	}
	if got.ID != u.ID {
		t.Errorf("expected user ID %q, got %q", u.ID, got.ID)
	}
	if got.Email != testEmail {
		t.Errorf("expected email %q, got %q", testEmail, got.Email)
	}
	if got.RoleName != "admin" {
		t.Errorf("expected RoleName 'admin', got %q", got.RoleName)
	}

	// Test 3: Nonexistent identifier should return ErrUserNotFound
	_, err = userRepo.GetByUsernameOrEmail(context.Background(), "nonexistent-"+uuid.NewString())
	if err != ErrUserNotFound {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestRoleRepository_List(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	repo := NewRoleRepository(db)

	roles, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(roles) != 2 {
		t.Fatalf("expected 2 seeded roles, got %d", len(roles))
	}
	byName := map[string]int16{}
	for _, r := range roles {
		byName[r.Name] = r.ID
	}
	if byName["user"] != 1 {
		t.Errorf("expected 'user' role id 1, got %d", byName["user"])
	}
	if byName["admin"] != 2 {
		t.Errorf("expected 'admin' role id 2, got %d", byName["admin"])
	}
}
