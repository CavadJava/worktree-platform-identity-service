package database

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func testMigrateDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DSN")
	if dsn == "" {
		dsn = "host=localhost port=5433 user=postgres password=1 dbname=platform_identity sslmode=disable"
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if _, err := db.Exec(`CREATE SCHEMA IF NOT EXISTS test_database`); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	if _, err := db.Exec(`SET search_path TO test_database`); err != nil {
		t.Fatalf("set search_path: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestMigrate_ProductProfileColumnsAndSubprojectsTable(t *testing.T) {
	db := testMigrateDB(t)

	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// Running it twice must stay a no-op (idempotency guarantee).
	if err := Migrate(db); err != nil {
		t.Fatalf("second migrate: %v", err)
	}

	var description, techStack string
	err := db.QueryRow(`
		INSERT INTO products (id, name, created_at) VALUES (gen_random_uuid(), 'Test Product', now())
		RETURNING description, tech_stack
	`).Scan(&description, &techStack)
	if err != nil {
		t.Fatalf("insert product: %v", err)
	}
	if description != "" || techStack != "" {
		t.Fatalf("expected empty defaults, got description=%q tech_stack=%q", description, techStack)
	}

	var productID string
	if err := db.QueryRow(`SELECT id FROM products WHERE name = 'Test Product'`).Scan(&productID); err != nil {
		t.Fatalf("select product id: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO product_subprojects (id, product_id, name, description, created_at)
		VALUES (gen_random_uuid(), $1, 'auth-service', 'handles login', now())
	`, productID)
	if err != nil {
		t.Fatalf("insert subproject: %v", err)
	}
}

func TestMigrate_ProductAdminRequestsTable(t *testing.T) {
	db := testMigrateDB(t)
	if err := Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	var productID, userID string
	if err := db.QueryRow(`INSERT INTO products (id, name, created_at) VALUES (gen_random_uuid(), 'Req Product', now()) RETURNING id`).Scan(&productID); err != nil {
		t.Fatalf("insert product: %v", err)
	}
	if err := db.QueryRow(`
		INSERT INTO users (id, name, username, email, password_hash, system_role_id, status, created_at, updated_at)
		VALUES (gen_random_uuid(), 'Req User', 'requser-'||gen_random_uuid(), 'requser-'||gen_random_uuid()||'@example.com', 'x', 3, 'ACTIVE', now(), now())
		RETURNING id
	`).Scan(&userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	var reqID, status string
	err := db.QueryRow(`
		INSERT INTO product_admin_requests (id, product_id, subject_user_id, requested_by_user_id, status, created_at)
		VALUES (gen_random_uuid(), $1, $2, $2, 'pending', now())
		RETURNING id, status
	`, productID, userID).Scan(&reqID, &status)
	if err != nil {
		t.Fatalf("insert request: %v", err)
	}
	if status != "pending" {
		t.Fatalf("expected default status 'pending', got %q", status)
	}
}
