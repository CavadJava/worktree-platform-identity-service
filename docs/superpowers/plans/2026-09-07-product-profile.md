# Product Profile Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give every Product (Teslahubs, ESound, Xerite, Vault, ...) a superadmin-editable profile — free-text description, comma-separated tech stack, and a list of named subprojects — visible and editable from `platform-identity-admin`'s Products page.

**Architecture:** Additive Postgres migration (two `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` on `products`, one new `product_subprojects` table) + a new `SubprojectRepository` mirroring `ShopMembershipRepository`'s CRUD shape + `ProductService`/`ProductHandler` additions gated by the existing `SystemRoleSuperadmin` check + three frontend file edits (`types.ts`, `products.ts`, `ProductsPage.tsx`) adding a "Profil" modal to the existing Products table.

**Tech Stack:** Go (chi router, database/sql + pgx), Postgres, React + Vite + AntDesign + TanStack Query, axios.

## Global Constraints

- Every migration statement MUST be idempotent/additive: `ADD COLUMN IF NOT EXISTS`, `CREATE TABLE IF NOT EXISTS`, `CREATE INDEX IF NOT EXISTS` — no `DROP`, ever (see `platform-identity-service/internal/database/postgres.go`'s `Migrate` doc comment; this branch had a real data-loss incident from a non-idempotent migration).
- All new profile/subproject endpoints are superadmin-only, using the exact same check already in `ProductService.Create`-adjacent code: `if caller.SystemRole != models.SystemRoleSuperadmin { return nil, ErrForbidden }`.
- `tech_stack` is stored as the raw typed string (e.g. `"React, Go, Postgres"`) — split/trim into tags happens client-side at render time, never persisted as a separate structure.
- Subprojects support add + list + remove only — no in-place edit (explicitly deferred in the spec).
- Tests use the existing `TEST_DSN`-or-default-DSN + `CREATE SCHEMA IF NOT EXISTS test_...` + `SET search_path` + `database.Migrate(db)` pattern already used in `internal/service/helpers_test.go` and `internal/handlers/auth_handler_test.go` — reuse `testDB(t)` / `testRouter(t)`, do not reinvent it.

---

### Task 1: Migration — `products` columns + `product_subprojects` table

**Files:**
- Modify: `platform-identity-service/internal/database/postgres.go` (inside `Migrate`'s SQL block, after the existing `products` table statement, before `user_product_subscriptions`)
- Test: `platform-identity-service/internal/database/postgres_test.go` (new file)

**Interfaces:**
- Produces: `products.description TEXT NOT NULL DEFAULT ''`, `products.tech_stack TEXT NOT NULL DEFAULT ''` columns; `product_subprojects(id UUID PK, product_id UUID FK, name TEXT, description TEXT, created_at TIMESTAMPTZ)` table with index `idx_subprojects_product_id`.

- [ ] **Step 1: Write the failing test**

Create `platform-identity-service/internal/database/postgres_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd platform-identity-service && go test ./internal/database/... -run TestMigrate_ProductProfileColumnsAndSubprojectsTable -v`
Expected: FAIL — `pq: column "description" of relation "products" does not exist` (or `product_subprojects` relation does not exist).

- [ ] **Step 3: Write minimal implementation**

In `platform-identity-service/internal/database/postgres.go`, edit the `Migrate` function's SQL block — insert this immediately after the `products` table's closing `);` and before the `CREATE TABLE IF NOT EXISTS user_product_subscriptions` block:

```go
		ALTER TABLE products ADD COLUMN IF NOT EXISTS description TEXT NOT NULL DEFAULT '';
		ALTER TABLE products ADD COLUMN IF NOT EXISTS tech_stack TEXT NOT NULL DEFAULT '';

		CREATE TABLE IF NOT EXISTS product_subprojects (
			id UUID PRIMARY KEY,
			product_id UUID NOT NULL REFERENCES products(id),
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		);
		CREATE INDEX IF NOT EXISTS idx_subprojects_product_id ON product_subprojects (product_id);
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd platform-identity-service && go test ./internal/database/... -run TestMigrate_ProductProfileColumnsAndSubprojectsTable -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add platform-identity-service/internal/database/postgres.go platform-identity-service/internal/database/postgres_test.go
git commit -m "feat(platform-identity-service): add product profile columns and subprojects table

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 2: Models — `Product` profile fields + `ProductSubproject`

**Files:**
- Modify: `platform-identity-service/internal/models/models.go`

**Interfaces:**
- Consumes: none (pure data types).
- Produces: `models.Product.Description string`, `models.Product.TechStack string`; new `models.ProductSubproject{ID, ProductID, Name, Description string; CreatedAt time.Time}` — used by Tasks 3-5.

- [ ] **Step 1: No test needed (pure struct fields) — write the change directly**

In `platform-identity-service/internal/models/models.go`, replace the existing `Product` struct:

```go
type Product struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	TechStack   string    `json:"tech_stack"`
	CreatedAt   time.Time `json:"created_at"`
}
```

Add a new type after `Product`:

```go
type ProductSubproject struct {
	ID          string    `json:"id"`
	ProductID   string    `json:"product_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}
```

- [ ] **Step 2: Verify the package still builds**

Run: `cd platform-identity-service && go build ./...`
Expected: FAILS at this point — `product_repository.go`'s `Create`/`List`/`GetByID` scan calls don't reference the new fields yet, but adding fields to a struct never breaks existing `Scan(&p.ID, &p.Name, &p.CreatedAt)` calls (Go structs tolerate unused fields). So this actually PASSES. Confirm with the build command; if it fails for an unrelated reason, stop and investigate before continuing.

- [ ] **Step 3: Commit**

```bash
git add platform-identity-service/internal/models/models.go
git commit -m "feat(platform-identity-service): add Product profile fields and ProductSubproject model

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 3: Repository — extend `ProductRepository`, add `SubprojectRepository`

**Files:**
- Modify: `platform-identity-service/internal/repository/product_repository.go`
- Create: `platform-identity-service/internal/repository/subproject_repository.go`
- Test: `platform-identity-service/internal/repository/repository_test.go` (append to existing file)

**Interfaces:**
- Consumes: `models.Product`, `models.ProductSubproject` (Task 2).
- Produces: `ProductRepository.GetByID`/`List`/`Create` now populate `Description`/`TechStack`; new `ProductRepository.Update(ctx, productID, description, techStack string) error`; new `SubprojectRepository` with `NewSubprojectRepository(db *sql.DB) *SubprojectRepository`, `Create(ctx, s *models.ProductSubproject) error`, `ListByProduct(ctx, productID string) ([]models.ProductSubproject, error)`, `Delete(ctx, id string) error`, and `var ErrSubprojectNotFound = errors.New("subproject not found")` — consumed by Task 4's `ProductService`.

- [ ] **Step 1: Write the failing tests**

Append to `platform-identity-service/internal/repository/repository_test.go` (open the file first to confirm the exact helper name used for a test DB in this file — it is `testRepoDB(t)` per the existing file's convention; use whatever helper that file already defines for its other `TestProductRepository_*` tests, matching its existing setup pattern exactly):

```go
func TestProductRepository_UpdateProfile(t *testing.T) {
	db := testRepoDB(t)
	repo := repository.NewProductRepository(db)

	p := &models.Product{ID: uuid.NewString(), Name: "Teslahubs", CreatedAt: time.Now().UTC()}
	if err := repo.Create(context.Background(), p); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := repo.Update(context.Background(), p.ID, "A Tesla platform", "React, Go, Postgres"); err != nil {
		t.Fatalf("update: %v", err)
	}

	got, err := repo.GetByID(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Description != "A Tesla platform" || got.TechStack != "React, Go, Postgres" {
		t.Fatalf("expected profile to be updated, got description=%q tech_stack=%q", got.Description, got.TechStack)
	}
}

func TestSubprojectRepository_CreateListDelete(t *testing.T) {
	db := testRepoDB(t)
	productRepo := repository.NewProductRepository(db)
	subRepo := repository.NewSubprojectRepository(db)

	p := &models.Product{ID: uuid.NewString(), Name: "Teslahubs", CreatedAt: time.Now().UTC()}
	if err := productRepo.Create(context.Background(), p); err != nil {
		t.Fatalf("create product: %v", err)
	}

	sub := &models.ProductSubproject{ID: uuid.NewString(), ProductID: p.ID, Name: "auth-service", Description: "handles login", CreatedAt: time.Now().UTC()}
	if err := subRepo.Create(context.Background(), sub); err != nil {
		t.Fatalf("create subproject: %v", err)
	}

	list, err := subRepo.ListByProduct(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 1 || list[0].Name != "auth-service" {
		t.Fatalf("expected one subproject named auth-service, got %+v", list)
	}

	if err := subRepo.Delete(context.Background(), sub.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	list, err = subRepo.ListByProduct(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("list after delete: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected zero subprojects after delete, got %d", len(list))
	}

	if err := subRepo.Delete(context.Background(), uuid.NewString()); !errors.Is(err, repository.ErrSubprojectNotFound) {
		t.Fatalf("expected ErrSubprojectNotFound for unknown id, got %v", err)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd platform-identity-service && go test ./internal/repository/... -run 'TestProductRepository_UpdateProfile|TestSubprojectRepository_CreateListDelete' -v`
Expected: FAIL to compile — `repo.Update` and `repository.NewSubprojectRepository` undefined.

- [ ] **Step 3: Write minimal implementation**

Replace `platform-identity-service/internal/repository/product_repository.go` in full:

```go
package repository

import (
	"context"
	"database/sql"
	"errors"

	"platform-identity-service/internal/models"
)

var ErrProductNotFound = errors.New("product not found")

type ProductRepository struct {
	db *sql.DB
}

func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

func (r *ProductRepository) Create(ctx context.Context, p *models.Product) error {
	const q = `INSERT INTO products (id, name, created_at) VALUES ($1, $2, $3)`
	_, err := r.db.ExecContext(ctx, q, p.ID, p.Name, p.CreatedAt)
	return err
}

func (r *ProductRepository) List(ctx context.Context) ([]models.Product, error) {
	const q = `SELECT id, name, description, tech_stack, created_at FROM products ORDER BY created_at DESC`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := []models.Product{}
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.TechStack, &p.CreatedAt); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

func (r *ProductRepository) GetByID(ctx context.Context, id string) (*models.Product, error) {
	const q = `SELECT id, name, description, tech_stack, created_at FROM products WHERE id = $1`
	var p models.Product
	err := r.db.QueryRowContext(ctx, q, id).Scan(&p.ID, &p.Name, &p.Description, &p.TechStack, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProductNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// Update sets a product's profile fields. Both fields are always submitted
// together by the admin panel's profile form, so there is no partial-update
// variant here (unlike UserUpdate's nil-field pattern).
func (r *ProductRepository) Update(ctx context.Context, productID, description, techStack string) error {
	const q = `UPDATE products SET description = $2, tech_stack = $3 WHERE id = $1`
	result, err := r.db.ExecContext(ctx, q, productID, description, techStack)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrProductNotFound
	}
	return nil
}
```

Create `platform-identity-service/internal/repository/subproject_repository.go`:

```go
package repository

import (
	"context"
	"database/sql"
	"errors"

	"platform-identity-service/internal/models"
)

var ErrSubprojectNotFound = errors.New("subproject not found")

type SubprojectRepository struct {
	db *sql.DB
}

func NewSubprojectRepository(db *sql.DB) *SubprojectRepository {
	return &SubprojectRepository{db: db}
}

func (r *SubprojectRepository) Create(ctx context.Context, s *models.ProductSubproject) error {
	const q = `
		INSERT INTO product_subprojects (id, product_id, name, description, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.ExecContext(ctx, q, s.ID, s.ProductID, s.Name, s.Description, s.CreatedAt)
	return err
}

func (r *SubprojectRepository) ListByProduct(ctx context.Context, productID string) ([]models.ProductSubproject, error) {
	const q = `
		SELECT id, product_id, name, description, created_at
		FROM product_subprojects
		WHERE product_id = $1
		ORDER BY created_at
	`
	rows, err := r.db.QueryContext(ctx, q, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	subprojects := []models.ProductSubproject{}
	for rows.Next() {
		var s models.ProductSubproject
		if err := rows.Scan(&s.ID, &s.ProductID, &s.Name, &s.Description, &s.CreatedAt); err != nil {
			return nil, err
		}
		subprojects = append(subprojects, s)
	}
	return subprojects, rows.Err()
}

func (r *SubprojectRepository) Delete(ctx context.Context, id string) error {
	const q = `DELETE FROM product_subprojects WHERE id = $1`
	result, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrSubprojectNotFound
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd platform-identity-service && go test ./internal/repository/... -run 'TestProductRepository_UpdateProfile|TestSubprojectRepository_CreateListDelete' -v`
Expected: PASS

Also run the full repository package to confirm no existing test broke from the `List`/`GetByID` scan changes:

Run: `cd platform-identity-service && go test ./internal/repository/... -v`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add platform-identity-service/internal/repository/product_repository.go platform-identity-service/internal/repository/subproject_repository.go platform-identity-service/internal/repository/repository_test.go
git commit -m "feat(platform-identity-service): add ProductRepository.Update and SubprojectRepository

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 4: Service — `ProductService` profile + subproject methods

**Files:**
- Modify: `platform-identity-service/internal/service/product_service.go`
- Test: `platform-identity-service/internal/service/product_service_test.go` (new file, or append if one already exists — check first with `ls platform-identity-service/internal/service/product_service_test.go`)

**Interfaces:**
- Consumes: `repository.ProductRepository.Update` (Task 3), `repository.SubprojectRepository` (Task 3), `shopassign.Caller` (existing type, has `.SystemRole string` field per `SetSubscription`'s existing usage), `models.SystemRoleSuperadmin` (existing constant), `ErrForbidden` (existing package-level var in this service package).
- Produces: `ProductService.UpdateProfile(ctx, caller shopassign.Caller, productID, description, techStack string) (*models.Product, error)`, `ProductService.AddSubproject(ctx, caller shopassign.Caller, productID, name, description string) (*models.ProductSubproject, error)`, `ProductService.ListSubprojects(ctx, productID string) ([]models.ProductSubproject, error)` (read, no caller check — matches `List`'s existing no-auth pattern), `ProductService.RemoveSubproject(ctx, caller shopassign.Caller, subprojectID string) error` — consumed by Task 5's handler.

- [ ] **Step 1: Write the failing tests**

Create `platform-identity-service/internal/service/product_service_test.go`:

```go
package service_test

import (
	"context"
	"errors"
	"testing"

	"platform-identity-service/internal/models"
	"platform-identity-service/internal/repository"
	"platform-identity-service/internal/service"
	"platform-identity-service/internal/service/shopassign"
)

func newTestProductService(t *testing.T) (*service.ProductService, *repository.ProductRepository) {
	t.Helper()
	db := testDB(t) // reuses the existing testDB helper in this package's helpers_test.go
	productRepo := repository.NewProductRepository(db)
	subRepo := repository.NewSubprojectRepository(db)
	subscriptionRepo := repository.NewSubscriptionRepository(db)
	return service.NewProductService(productRepo, subscriptionRepo, subRepo), productRepo
}

func TestProductService_UpdateProfile_RequiresSuperadmin(t *testing.T) {
	svc, productRepo := newTestProductService(t)
	p, err := svc.Create(context.Background(), "Teslahubs")
	if err != nil {
		t.Fatalf("create product: %v", err)
	}

	_, err = svc.UpdateProfile(context.Background(), shopassign.Caller{SystemRole: models.SystemRoleUser}, p.ID, "desc", "React")
	if !errors.Is(err, service.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for non-superadmin, got %v", err)
	}

	updated, err := svc.UpdateProfile(context.Background(), shopassign.Caller{SystemRole: models.SystemRoleSuperadmin}, p.ID, "A Tesla platform", "React, Go, Postgres")
	if err != nil {
		t.Fatalf("update profile: %v", err)
	}
	if updated.Description != "A Tesla platform" || updated.TechStack != "React, Go, Postgres" {
		t.Fatalf("expected updated profile, got %+v", updated)
	}

	_ = productRepo
}

func TestProductService_SubprojectLifecycle(t *testing.T) {
	svc, _ := newTestProductService(t)
	p, err := svc.Create(context.Background(), "Teslahubs")
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	superadmin := shopassign.Caller{SystemRole: models.SystemRoleSuperadmin}
	nonAdmin := shopassign.Caller{SystemRole: models.SystemRoleUser}

	if _, err := svc.AddSubproject(context.Background(), nonAdmin, p.ID, "auth-service", "handles login"); !errors.Is(err, service.ErrForbidden) {
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

	if err := svc.RemoveSubproject(context.Background(), nonAdmin, sub.ID); !errors.Is(err, service.ErrForbidden) {
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd platform-identity-service && go test ./internal/service/... -run 'TestProductService_UpdateProfile_RequiresSuperadmin|TestProductService_SubprojectLifecycle' -v`
Expected: FAIL to compile — `service.NewProductService` takes 2 args not 3, `UpdateProfile`/`AddSubproject`/`ListSubprojects`/`RemoveSubproject` undefined.

- [ ] **Step 3: Write minimal implementation**

Replace `platform-identity-service/internal/service/product_service.go` in full:

```go
package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"platform-identity-service/internal/models"
	"platform-identity-service/internal/repository"
	"platform-identity-service/internal/service/shopassign"
)

var ErrProductNotFound = repository.ErrProductNotFound

const (
	AccessFull = "full"
	AccessDemo = "demo"
)

type ProductService struct {
	productRepo *repository.ProductRepository
	subRepo     *repository.SubscriptionRepository
	subprojRepo *repository.SubprojectRepository
}

func NewProductService(productRepo *repository.ProductRepository, subRepo *repository.SubscriptionRepository, subprojRepo *repository.SubprojectRepository) *ProductService {
	return &ProductService{productRepo: productRepo, subRepo: subRepo, subprojRepo: subprojRepo}
}

func (s *ProductService) Create(ctx context.Context, name string) (*models.Product, error) {
	p := &models.Product{ID: uuid.NewString(), Name: name, CreatedAt: time.Now().UTC()}
	if err := s.productRepo.Create(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *ProductService) List(ctx context.Context) ([]models.Product, error) {
	return s.productRepo.List(ctx)
}

// UpdateProfile lets a superadmin set a product's description and tech
// stack. Both fields are always submitted together by the admin panel form.
func (s *ProductService) UpdateProfile(ctx context.Context, caller shopassign.Caller, productID, description, techStack string) (*models.Product, error) {
	if caller.SystemRole != models.SystemRoleSuperadmin {
		return nil, ErrForbidden
	}
	if err := s.productRepo.Update(ctx, productID, description, techStack); err != nil {
		return nil, err
	}
	return s.productRepo.GetByID(ctx, productID)
}

// AddSubproject lets a superadmin record a named internal module of a
// product (e.g. Teslahubs -> auth-service). Manually entered, no external
// discovery.
func (s *ProductService) AddSubproject(ctx context.Context, caller shopassign.Caller, productID, name, description string) (*models.ProductSubproject, error) {
	if caller.SystemRole != models.SystemRoleSuperadmin {
		return nil, ErrForbidden
	}
	if _, err := s.productRepo.GetByID(ctx, productID); err != nil {
		return nil, err
	}
	sub := &models.ProductSubproject{
		ID: uuid.NewString(), ProductID: productID, Name: name, Description: description,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.subprojRepo.Create(ctx, sub); err != nil {
		return nil, err
	}
	return sub, nil
}

// ListSubprojects has no caller-role check, matching List's existing
// no-auth read pattern — subprojects are visible to anyone who can see the
// product.
func (s *ProductService) ListSubprojects(ctx context.Context, productID string) ([]models.ProductSubproject, error) {
	return s.subprojRepo.ListByProduct(ctx, productID)
}

func (s *ProductService) RemoveSubproject(ctx context.Context, caller shopassign.Caller, subprojectID string) error {
	if caller.SystemRole != models.SystemRoleSuperadmin {
		return ErrForbidden
	}
	return s.subprojRepo.Delete(ctx, subprojectID)
}

// CheckAccess reports "full" if userID has an active subscription to
// productID, "demo" otherwise (including when no subscription row exists
// at all — a user who never subscribed still gets demo access, not an
// error).
func (s *ProductService) CheckAccess(ctx context.Context, userID, productID string) (string, error) {
	if _, err := s.productRepo.GetByID(ctx, productID); err != nil {
		return "", err
	}

	sub, err := s.subRepo.GetByUserAndProduct(ctx, userID, productID)
	if errors.Is(err, repository.ErrSubscriptionNotFound) {
		return AccessDemo, nil
	}
	if err != nil {
		return "", err
	}
	if sub.Subscripted {
		return AccessFull, nil
	}
	return AccessDemo, nil
}

// SetSubscription is called by an admin/superadmin to manually toggle a
// user's subscription — there is no payment gateway integration in this
// scope.
func (s *ProductService) SetSubscription(ctx context.Context, caller shopassign.Caller, targetUserID, productID string, subscripted, renewed bool) (*models.Subscription, error) {
	if caller.SystemRole != models.SystemRoleSuperadmin {
		return nil, ErrForbidden
	}
	if _, err := s.productRepo.GetByID(ctx, productID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	existing, err := s.subRepo.GetByUserAndProduct(ctx, targetUserID, productID)
	id := uuid.NewString()
	createdAt := now
	if err == nil {
		id = existing.ID
		createdAt = existing.CreatedAt
	} else if !errors.Is(err, repository.ErrSubscriptionNotFound) {
		return nil, err
	}

	sub := &models.Subscription{
		ID: id, UserID: targetUserID, ProductID: productID,
		Subscripted: subscripted, Renewed: renewed,
		CreatedAt: createdAt, UpdatedAt: now,
	}
	if err := s.subRepo.Upsert(ctx, sub); err != nil {
		return nil, err
	}
	return sub, nil
}
```

**Note:** `NewProductService`'s signature changes from 2 params to 3 — Task 6 updates the one call site in `cmd/api/main.go`. Grep for any other call sites before moving on: `grep -rn "NewProductService(" platform-identity-service/` and update every match found (there should be exactly one production call site plus the test helper this task just added).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd platform-identity-service && go test ./internal/service/... -v`
Expected: PASS (this will also fail to compile at first if `cmd/api/main.go` isn't updated yet — but `main.go` is a separate package from `internal/service`, so `go test ./internal/service/...` compiles independently and should pass here; `go build ./...` will still fail until Task 6, which is expected and fixed there).

- [ ] **Step 5: Commit**

```bash
git add platform-identity-service/internal/service/product_service.go platform-identity-service/internal/service/product_service_test.go
git commit -m "feat(platform-identity-service): add ProductService profile and subproject methods

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 5: Handler — profile + subproject routes

**Files:**
- Modify: `platform-identity-service/internal/handlers/product_handler.go`
- Test: `platform-identity-service/internal/handlers/product_handler_test.go` (new file)

**Interfaces:**
- Consumes: `service.ProductService.UpdateProfile`/`AddSubproject`/`ListSubprojects`/`RemoveSubproject` (Task 4), `middleware.CallerFromContext` (existing), `service.ErrForbidden` (existing), `repository.ErrProductNotFound`/`ErrSubprojectNotFound` (existing/Task 3).
- Produces: `productResponse` gains `description`, `tech_stack` fields; new `subprojectResponse{id, product_id, name, description, created_at}`; new handler methods `UpdateProfile`, `AddSubproject`, `ListSubprojects`, `RemoveSubproject` on `ProductHandler` — wired to routes in Task 6.

- [ ] **Step 1: Write the failing test**

Create `platform-identity-service/internal/handlers/product_handler_test.go`:

```go
package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	_ "github.com/jackc/pgx/v5/stdlib"

	"platform-identity-service/internal/auth"
	"platform-identity-service/internal/database"
	appmiddleware "platform-identity-service/internal/middleware"
	"platform-identity-service/internal/models"
	"platform-identity-service/internal/repository"
	"platform-identity-service/internal/service"
)

func testProductRouter(t *testing.T) (chi.Router, *sql.DB, *auth.JWTManager) {
	t.Helper()
	dsn := os.Getenv("TEST_DSN")
	if dsn == "" {
		dsn = "host=localhost port=5433 user=postgres password=1 dbname=platform_identity sslmode=disable"
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if _, err := db.Exec(`CREATE SCHEMA IF NOT EXISTS test_handlers_product`); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	if _, err := db.Exec(`SET search_path TO test_handlers_product`); err != nil {
		t.Fatalf("set search_path: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	productRepo := repository.NewProductRepository(db)
	subRepo := repository.NewSubscriptionRepository(db)
	subprojRepo := repository.NewSubprojectRepository(db)
	jwtManager := auth.NewJWTManager("product-handler-test-secret", 60)

	productService := service.NewProductService(productRepo, subRepo, subprojRepo)
	productHandler := NewProductHandler(productService)

	r := chi.NewRouter()
	r.Get("/api/v1/products", productHandler.List)
	r.Get("/api/v1/products/{id}/subprojects", productHandler.ListSubprojects)
	r.Group(func(r chi.Router) {
		r.Use(appmiddleware.RequireAuth(jwtManager, userRepo))
		r.Post("/api/v1/products", productHandler.Create)
		r.Post("/api/v1/products/{id}/profile", productHandler.UpdateProfile)
		r.Post("/api/v1/products/{id}/subprojects", productHandler.AddSubproject)
		r.Delete("/api/v1/products/{id}/subprojects/{subId}", productHandler.RemoveSubproject)
	})

	t.Cleanup(func() { db.Close() })
	return r, db, jwtManager
}

func createTestSuperadmin(t *testing.T, db *sql.DB, jwtManager *auth.JWTManager) string {
	t.Helper()
	userRepo := repository.NewUserRepository(db)
	u := &models.User{
		ID: "super-" + t.Name(), Name: "Super", Username: "super-" + t.Name(), Email: "super-" + t.Name() + "@example.com",
		PasswordHash: "x", SystemRoleID: 1, Status: models.UserStatusActive,
	}
	_ = userRepo // repository package imported for type only if Create signature needs it; see below.
	if err := userRepo.Create(context.Background(), u); err != nil {
		t.Fatalf("create superadmin: %v", err)
	}
	token, err := jwtManager.Generate(u.ID)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	return token
}

func TestProductHandler_ProfileAndSubprojects(t *testing.T) {
	r, db, jwtManager := testProductRouter(t)
	token := createTestSuperadmin(t, db, jwtManager)

	rec := doJSON(t, r, http.MethodPost, "/api/v1/products", token, map[string]string{"name": "Teslahubs"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create product: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var created productResponse
	json.NewDecoder(rec.Body).Decode(&created)

	rec = doJSON(t, r, http.MethodPost, "/api/v1/products/"+created.ID+"/profile", token, map[string]string{
		"description": "A Tesla platform", "tech_stack": "React, Go, Postgres",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("update profile: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var profile productResponse
	json.NewDecoder(rec.Body).Decode(&profile)
	if profile.Description != "A Tesla platform" || profile.TechStack != "React, Go, Postgres" {
		t.Fatalf("expected updated profile in response, got %+v", profile)
	}

	rec = doJSON(t, r, http.MethodPost, "/api/v1/products/"+created.ID+"/subprojects", token, map[string]string{
		"name": "auth-service", "description": "handles login",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("add subproject: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var sub subprojectResponse
	json.NewDecoder(rec.Body).Decode(&sub)
	if sub.Name != "auth-service" {
		t.Fatalf("expected subproject name auth-service, got %+v", sub)
	}

	rec = doJSON(t, r, http.MethodGet, "/api/v1/products/"+created.ID+"/subprojects", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list subprojects: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var list []subprojectResponse
	json.NewDecoder(rec.Body).Decode(&list)
	if len(list) != 1 {
		t.Fatalf("expected 1 subproject, got %d", len(list))
	}

	rec = doJSON(t, r, http.MethodDelete, "/api/v1/products/"+created.ID+"/subprojects/"+sub.ID, token, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("remove subproject: expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}
```

Note: this test file uses `doJSON` and relies on `context` — add `"context"` to the import block above (it was omitted for brevity in this listing; include it when creating the file, since `createTestSuperadmin` calls `context.Background()`). It reuses `doJSON` from `auth_handler_test.go`, which is in the same `handlers` package, so no re-declaration is needed.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd platform-identity-service && go test ./internal/handlers/... -run TestProductHandler_ProfileAndSubprojects -v`
Expected: FAIL to compile — `productHandler.UpdateProfile`/`AddSubproject`/`ListSubprojects`/`RemoveSubproject` undefined, `productResponse`/new `subprojectResponse` missing fields.

- [ ] **Step 3: Write minimal implementation**

Replace `platform-identity-service/internal/handlers/product_handler.go` in full:

```go
package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"platform-identity-service/internal/middleware"
	"platform-identity-service/internal/models"
	"platform-identity-service/internal/repository"
	"platform-identity-service/internal/service"
)

type ProductHandler struct {
	svc *service.ProductService
}

func NewProductHandler(svc *service.ProductService) *ProductHandler {
	return &ProductHandler{svc: svc}
}

type createProductRequest struct {
	Name string `json:"name"`
}

type productResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	TechStack   string `json:"tech_stack"`
	CreatedAt   string `json:"created_at"`
}

func toProductResponse(p *models.Product) productResponse {
	return productResponse{
		ID: p.ID, Name: p.Name, Description: p.Description, TechStack: p.TechStack,
		CreatedAt: p.CreatedAt.Format(timeFormat),
	}
}

// Create godoc
// @Summary      Create a product
// @Description  Superadmin only.
// @Tags         products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        request body createProductRequest true "Product payload"
// @Success      201 {object} productResponse
// @Failure      403 {object} map[string]string
// @Router       /products [post]
func (h *ProductHandler) Create(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if caller.SystemRole != models.SystemRoleSuperadmin {
		writeError(w, http.StatusForbidden, "superadmin role required")
		return
	}

	var req createProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	p, err := h.svc.Create(r.Context(), req.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create product")
		return
	}
	writeJSON(w, http.StatusCreated, toProductResponse(p))
}

// List godoc
// @Summary      List all products
// @Tags         products
// @Produce      json
// @Success      200 {array} productResponse
// @Router       /products [get]
func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	products, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list products")
		return
	}

	response := make([]productResponse, len(products))
	for i, p := range products {
		response[i] = toProductResponse(&p)
	}
	writeJSON(w, http.StatusOK, response)
}

type accessResponse struct {
	Access string `json:"access"`
}

// CheckAccess godoc
// @Summary      Check the caller's access level to a product
// @Description  Returns "full" if subscribed, "demo" otherwise.
// @Tags         products
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Product ID"
// @Success      200 {object} accessResponse
// @Failure      404 {object} map[string]string
// @Router       /products/{id}/access [get]
func (h *ProductHandler) CheckAccess(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	productID := chi.URLParam(r, "id")
	access, err := h.svc.CheckAccess(r.Context(), caller.UserID, productID)
	if err != nil {
		if errors.Is(err, repository.ErrProductNotFound) {
			writeError(w, http.StatusNotFound, "product not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to check access")
		return
	}
	writeJSON(w, http.StatusOK, accessResponse{Access: access})
}

type setSubscriptionRequest struct {
	Subscripted bool `json:"subscripted"`
	Renewed     bool `json:"renewed"`
}

type subscriptionResponse struct {
	UserID      string `json:"user_id"`
	ProductID   string `json:"product_id"`
	Subscripted bool   `json:"subscripted"`
	Renewed     bool   `json:"renewed"`
}

// SetSubscription godoc
// @Summary      Manually set a user's subscription to a product
// @Description  Superadmin only. No payment gateway integration.
// @Tags         products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        userId path string true "User ID"
// @Param        productId path string true "Product ID"
// @Param        request body setSubscriptionRequest true "Subscription payload"
// @Success      200 {object} subscriptionResponse
// @Failure      403 {object} map[string]string
// @Router       /users/{userId}/products/{productId}/subscribe [post]
func (h *ProductHandler) SetSubscription(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req setSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	targetUserID := chi.URLParam(r, "userId")
	productID := chi.URLParam(r, "productId")
	sub, err := h.svc.SetSubscription(r.Context(), caller, targetUserID, productID, req.Subscripted, req.Renewed)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			writeError(w, http.StatusForbidden, "superadmin or admin role required")
		case errors.Is(err, repository.ErrProductNotFound):
			writeError(w, http.StatusNotFound, "product not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to set subscription")
		}
		return
	}
	writeJSON(w, http.StatusOK, subscriptionResponse{
		UserID: sub.UserID, ProductID: sub.ProductID, Subscripted: sub.Subscripted, Renewed: sub.Renewed,
	})
}

type updateProductProfileRequest struct {
	Description string `json:"description"`
	TechStack   string `json:"tech_stack"`
}

// UpdateProfile godoc
// @Summary      Set a product's description and tech stack
// @Description  Superadmin only.
// @Tags         products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Product ID"
// @Param        request body updateProductProfileRequest true "Profile payload"
// @Success      200 {object} productResponse
// @Failure      403 {object} map[string]string
// @Router       /products/{id}/profile [post]
func (h *ProductHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req updateProductProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	productID := chi.URLParam(r, "id")
	p, err := h.svc.UpdateProfile(r.Context(), caller, productID, req.Description, req.TechStack)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			writeError(w, http.StatusForbidden, "superadmin role required")
		case errors.Is(err, repository.ErrProductNotFound):
			writeError(w, http.StatusNotFound, "product not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to update profile")
		}
		return
	}
	writeJSON(w, http.StatusOK, toProductResponse(p))
}

type addSubprojectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type subprojectResponse struct {
	ID          string `json:"id"`
	ProductID   string `json:"product_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
}

// AddSubproject godoc
// @Summary      Add a named subproject/module to a product
// @Description  Superadmin only.
// @Tags         products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Product ID"
// @Param        request body addSubprojectRequest true "Subproject payload"
// @Success      201 {object} subprojectResponse
// @Failure      403 {object} map[string]string
// @Router       /products/{id}/subprojects [post]
func (h *ProductHandler) AddSubproject(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req addSubprojectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	productID := chi.URLParam(r, "id")
	sub, err := h.svc.AddSubproject(r.Context(), caller, productID, req.Name, req.Description)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			writeError(w, http.StatusForbidden, "superadmin role required")
		case errors.Is(err, repository.ErrProductNotFound):
			writeError(w, http.StatusNotFound, "product not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to add subproject")
		}
		return
	}
	writeJSON(w, http.StatusCreated, subprojectResponse{
		ID: sub.ID, ProductID: sub.ProductID, Name: sub.Name, Description: sub.Description,
		CreatedAt: sub.CreatedAt.Format(timeFormat),
	})
}

// ListSubprojects godoc
// @Summary      List a product's subprojects
// @Tags         products
// @Produce      json
// @Param        id path string true "Product ID"
// @Success      200 {array} subprojectResponse
// @Router       /products/{id}/subprojects [get]
func (h *ProductHandler) ListSubprojects(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "id")
	subprojects, err := h.svc.ListSubprojects(r.Context(), productID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list subprojects")
		return
	}

	response := make([]subprojectResponse, len(subprojects))
	for i, s := range subprojects {
		response[i] = subprojectResponse{
			ID: s.ID, ProductID: s.ProductID, Name: s.Name, Description: s.Description,
			CreatedAt: s.CreatedAt.Format(timeFormat),
		}
	}
	writeJSON(w, http.StatusOK, response)
}

// RemoveSubproject godoc
// @Summary      Remove a subproject
// @Description  Superadmin only.
// @Tags         products
// @Security     BearerAuth
// @Param        id path string true "Product ID"
// @Param        subId path string true "Subproject ID"
// @Success      204
// @Failure      403 {object} map[string]string
// @Router       /products/{id}/subprojects/{subId} [delete]
func (h *ProductHandler) RemoveSubproject(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	subID := chi.URLParam(r, "subId")
	err := h.svc.RemoveSubproject(r.Context(), caller, subID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			writeError(w, http.StatusForbidden, "superadmin role required")
		case errors.Is(err, repository.ErrSubprojectNotFound):
			writeError(w, http.StatusNotFound, "subproject not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to remove subproject")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd platform-identity-service && go test ./internal/handlers/... -run TestProductHandler_ProfileAndSubprojects -v`
Expected: PASS

Also run the full handlers package to confirm no regression:

Run: `cd platform-identity-service && go test ./internal/handlers/... -v`
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add platform-identity-service/internal/handlers/product_handler.go platform-identity-service/internal/handlers/product_handler_test.go
git commit -m "feat(platform-identity-service): add product profile and subproject HTTP routes

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 6: Wire new routes and repo in `cmd/api/main.go`

**Files:**
- Modify: `platform-identity-service/cmd/api/main.go`

**Interfaces:**
- Consumes: `repository.NewSubprojectRepository` (Task 3), `service.NewProductService(productRepo, subRepo, subprojRepo)` (Task 4, now 3 args), `productHandler.UpdateProfile`/`AddSubproject`/`ListSubprojects`/`RemoveSubproject` (Task 5).
- Produces: whole binary builds and runs with the new routes live.

- [ ] **Step 1: No new test — this task is pure wiring, verified by full build + full test suite**

Open `platform-identity-service/cmd/api/main.go` and locate the existing lines (from the earlier grep):

```
58:	productRepo := repository.NewProductRepository(db)
68:	productService := service.NewProductService(productRepo, subscriptionRepo)
76:	productHandler := handlers.NewProductHandler(productService)
104:		r.Get("/products", productHandler.List)
127:			r.Post("/products", productHandler.Create)
128:			r.Get("/products/{id}/access", productHandler.CheckAccess)
129:			r.Post("/users/{userId}/products/{productId}/subscribe", productHandler.SetSubscription)
```

- [ ] **Step 2: Add the subproject repository next to `productRepo`**

Immediately after the line `productRepo := repository.NewProductRepository(db)`, add:

```go
	subprojectRepo := repository.NewSubprojectRepository(db)
```

- [ ] **Step 3: Update the `NewProductService` call site**

Change:

```go
	productService := service.NewProductService(productRepo, subscriptionRepo)
```

to:

```go
	productService := service.NewProductService(productRepo, subscriptionRepo, subprojectRepo)
```

- [ ] **Step 4: Add the new public route**

Immediately after the line `r.Get("/products", productHandler.List)`, add:

```go
		r.Get("/products/{id}/subprojects", productHandler.ListSubprojects)
```

(This mirrors `List`'s existing no-auth-required placement — subprojects are readable by anyone who can see products, per Task 4's `ListSubprojects` design.)

- [ ] **Step 5: Add the new authenticated routes**

Immediately after the line `r.Post("/users/{userId}/products/{productId}/subscribe", productHandler.SetSubscription)`, add:

```go
			r.Post("/products/{id}/profile", productHandler.UpdateProfile)
			r.Post("/products/{id}/subprojects", productHandler.AddSubproject)
			r.Delete("/products/{id}/subprojects/{subId}", productHandler.RemoveSubproject)
```

- [ ] **Step 6: Verify the full build and full test suite pass**

Run: `cd platform-identity-service && go build ./...`
Expected: builds cleanly.

Run: `cd platform-identity-service && go test ./... -v`
Expected: all PASS (this is the first point where every package compiles together — confirms Tasks 1-6 are consistent end to end).

- [ ] **Step 7: Commit**

```bash
git add platform-identity-service/cmd/api/main.go
git commit -m "feat(platform-identity-service): wire product profile and subproject routes into main

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 7: Frontend — `types.ts` and `products.ts` API client

**Files:**
- Modify: `platform-identity-admin/src/api/types.ts`
- Modify: `platform-identity-admin/src/api/products.ts`

**Interfaces:**
- Produces: `Product.description?: string`, `Product.tech_stack?: string`; new `Subproject{id, product_id, name, description, created_at}` type; new functions `updateProductProfile(productId, description, techStack) => Promise<Product>`, `listSubprojects(productId) => Promise<Subproject[]>`, `addSubproject(productId, name, description) => Promise<Subproject>`, `removeSubproject(productId, subId) => Promise<void>` — consumed by Task 8's `ProductsPage.tsx`.

There is no automated frontend test harness referenced anywhere in this codebase's prior work (no `*.test.tsx` files were found for `ProductsPage` or `ShopMembersPage` in prior sessions) — this task and Task 8 are verified by `npm run build`/`tsc` type-checking plus a manual browser smoke test at the end of Task 8, matching this project's established practice for frontend changes.

- [ ] **Step 1: Locate and read the current full `types.ts` to confirm the exact `Product` interface location before editing**

Run: `grep -n "interface Product" platform-identity-admin/src/api/types.ts`

- [ ] **Step 2: Edit `types.ts`**

In `platform-identity-admin/src/api/types.ts`, replace:

```ts
export interface Product {
  id: string;
  name: string;
  created_at: string;
}
```

with:

```ts
export interface Product {
  id: string;
  name: string;
  description: string;
  tech_stack: string;
  created_at: string;
}

export interface Subproject {
  id: string;
  product_id: string;
  name: string;
  description: string;
  created_at: string;
}
```

- [ ] **Step 3: Edit `products.ts`**

Append to `platform-identity-admin/src/api/products.ts`:

```ts
import type { Subproject } from './types';

export async function updateProductProfile(
  productId: string,
  description: string,
  techStack: string
): Promise<Product> {
  const response = await platformIdentityApi.post<Product>(`/products/${productId}/profile`, {
    description,
    tech_stack: techStack,
  });
  return response.data;
}

export async function listSubprojects(productId: string): Promise<Subproject[]> {
  const response = await platformIdentityApi.get<Subproject[]>(`/products/${productId}/subprojects`);
  return response.data;
}

export async function addSubproject(productId: string, name: string, description: string): Promise<Subproject> {
  const response = await platformIdentityApi.post<Subproject>(`/products/${productId}/subprojects`, {
    name,
    description,
  });
  return response.data;
}

export async function removeSubproject(productId: string, subId: string): Promise<void> {
  await platformIdentityApi.delete(`/products/${productId}/subprojects/${subId}`);
}
```

Note: move the `import type { Subproject } from './types';` line to the top of the file next to the existing `import type { Product } from './types';` import rather than leaving it mid-file — combine both into one import statement: `import type { Product, Subproject } from './types';`.

- [ ] **Step 4: Type-check the frontend**

Run: `cd platform-identity-admin && npx tsc --noEmit`
Expected: no errors from `types.ts` or `products.ts`. (There will likely be a pre-existing error from `ProductsPage.tsx` referencing `product.created_at`-only usage still working fine since old fields aren't removed — the new required `description`/`tech_stack` fields on `Product` may cause a type error anywhere a `Product` literal is constructed without them; check for that with `grep -rn "created_at:" platform-identity-admin/src` and fix any bare object literals found, e.g. in test fixtures, by adding `description: '', tech_stack: ''`.)

- [ ] **Step 5: Commit**

```bash
git add platform-identity-admin/src/api/types.ts platform-identity-admin/src/api/products.ts
git commit -m "feat(platform-identity-admin): add product profile and subproject API client functions

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 8: Frontend — `ProductsPage.tsx` profile modal

**Files:**
- Modify: `platform-identity-admin/src/pages/ProductsPage.tsx`

**Interfaces:**
- Consumes: `updateProductProfile`, `listSubprojects`, `addSubproject`, `removeSubproject` (Task 7), `Product`, `Subproject` types (Task 7).
- Produces: a "Profil" button per row opening a modal with description textarea, tech-stack input rendered as Tags, and a subprojects add/remove mini-table — the final user-visible deliverable of this plan.

- [ ] **Step 1: Replace `platform-identity-admin/src/pages/ProductsPage.tsx` in full**

```tsx
import { useState } from 'react';
import { Button, Form, Input, List, Modal, Select, Space, Switch, Table, Tag, message } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { Product, Subproject } from '../api/types';
import {
  addSubproject,
  createProduct,
  listProducts,
  listSubprojects,
  removeSubproject,
  setSubscription,
  updateProductProfile,
} from '../api/products';
import { listAllUsers } from '../api/users';
import { useQueryErrorToast } from '../hooks/useQueryErrorToast';

interface ProductFormValues {
  name: string;
}

interface SubprojectFormValues {
  name: string;
  description: string;
}

export function ProductsPage() {
  const queryClient = useQueryClient();
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [subModalProduct, setSubModalProduct] = useState<Product | null>(null);
  const [subUserId, setSubUserId] = useState<string | null>(null);
  const [subscripted, setSubscripted] = useState(false);
  const [renewed, setRenewed] = useState(false);
  const [profileModalProduct, setProfileModalProduct] = useState<Product | null>(null);
  const [form] = Form.useForm<ProductFormValues>();
  const [profileForm] = Form.useForm<{ description: string; techStack: string }>();
  const [subprojectForm] = Form.useForm<SubprojectFormValues>();

  const { data: products, isLoading, isError, error } = useQuery({ queryKey: ['products'], queryFn: () => listProducts() });
  useQueryErrorToast(isError, error);

  const { data: allUsers } = useQuery({ queryKey: ['all-users-for-sub'], queryFn: () => listAllUsers(), enabled: !!subModalProduct });

  const { data: subprojects } = useQuery({
    queryKey: ['product-subprojects', profileModalProduct?.id],
    queryFn: () => listSubprojects(profileModalProduct!.id),
    enabled: !!profileModalProduct,
  });

  const createMutation = useMutation({
    mutationFn: (values: ProductFormValues) => createProduct(values.name),
    onSuccess: () => {
      message.success('Product yaradıldı');
      setCreateModalOpen(false);
      form.resetFields();
      queryClient.invalidateQueries({ queryKey: ['products'] });
    },
    onError: (err) => message.error(err instanceof Error ? err.message : 'Xəta baş verdi'),
  });

  const subMutation = useMutation({
    mutationFn: () => setSubscription(subUserId!, subModalProduct!.id, subscripted, renewed),
    onSuccess: () => {
      message.success('Subscription yeniləndi');
      setSubModalProduct(null);
      setSubUserId(null);
      setSubscripted(false);
      setRenewed(false);
    },
    onError: (err) => message.error(err instanceof Error ? err.message : 'Xəta baş verdi'),
  });

  const profileMutation = useMutation({
    mutationFn: (values: { description: string; techStack: string }) =>
      updateProductProfile(profileModalProduct!.id, values.description, values.techStack),
    onSuccess: () => {
      message.success('Profil yeniləndi');
      queryClient.invalidateQueries({ queryKey: ['products'] });
    },
    onError: (err) => message.error(err instanceof Error ? err.message : 'Xəta baş verdi'),
  });

  const addSubprojectMutation = useMutation({
    mutationFn: (values: SubprojectFormValues) =>
      addSubproject(profileModalProduct!.id, values.name, values.description),
    onSuccess: () => {
      subprojectForm.resetFields();
      queryClient.invalidateQueries({ queryKey: ['product-subprojects', profileModalProduct?.id] });
    },
    onError: (err) => message.error(err instanceof Error ? err.message : 'Xəta baş verdi'),
  });

  const removeSubprojectMutation = useMutation({
    mutationFn: (subId: string) => removeSubproject(profileModalProduct!.id, subId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['product-subprojects', profileModalProduct?.id] });
    },
    onError: (err) => message.error(err instanceof Error ? err.message : 'Xəta baş verdi'),
  });

  const openProfileModal = (product: Product) => {
    setProfileModalProduct(product);
    profileForm.setFieldsValue({ description: product.description, techStack: product.tech_stack });
  };

  const columns = [
    { title: 'Ad', dataIndex: 'name' },
    { title: 'ID', dataIndex: 'id' },
    {
      title: 'Əməliyyat',
      render: (_: unknown, record: Product) => (
        <Space>
          <Button size="small" onClick={() => openProfileModal(record)}>
            Profil
          </Button>
          <Button size="small" onClick={() => setSubModalProduct(record)}>
            Subscription idarə et
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <>
      <Button type="primary" onClick={() => setCreateModalOpen(true)} style={{ marginBottom: 16 }}>
        Yeni product
      </Button>
      <Table rowKey="id" loading={isLoading} dataSource={products} columns={columns} />
      <Modal
        title="Yeni product"
        open={createModalOpen}
        onCancel={() => setCreateModalOpen(false)}
        onOk={() => form.submit()}
        confirmLoading={createMutation.isPending}
      >
        <Form form={form} layout="vertical" onFinish={(values) => createMutation.mutate(values)}>
          <Form.Item name="name" label="Ad" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
        </Form>
      </Modal>
      <Modal
        title={subModalProduct ? `${subModalProduct.name} — Subscription` : ''}
        open={!!subModalProduct}
        onCancel={() => setSubModalProduct(null)}
        onOk={() => subMutation.mutate()}
        confirmLoading={subMutation.isPending}
        okButtonProps={{ disabled: !subUserId }}
      >
        <Space direction="vertical" style={{ width: '100%' }}>
          <Select
            style={{ width: '100%' }}
            placeholder="İstifadəçi seç"
            value={subUserId ?? undefined}
            onChange={setSubUserId}
            options={allUsers?.map((u) => ({ label: `${u.name} (${u.username})`, value: u.id }))}
          />
          <Space>
            <span>Subscripted:</span>
            <Switch checked={subscripted} onChange={setSubscripted} />
          </Space>
          <Space>
            <span>Renewed:</span>
            <Switch checked={renewed} onChange={setRenewed} />
          </Space>
        </Space>
      </Modal>
      <Modal
        title={profileModalProduct ? `${profileModalProduct.name} — Profil` : ''}
        open={!!profileModalProduct}
        onCancel={() => setProfileModalProduct(null)}
        footer={null}
        width={640}
      >
        <Form
          form={profileForm}
          layout="vertical"
          onFinish={(values) => profileMutation.mutate(values)}
        >
          <Form.Item name="description" label="Təsvir">
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item name="techStack" label="Tech stack (vergüllə ayrılmış, məs. React, Go, Postgres)">
            <Input />
          </Form.Item>
          <Form.Item shouldUpdate>
            {() => {
              const raw = profileForm.getFieldValue('techStack') as string | undefined;
              const tags = (raw ?? '').split(',').map((t) => t.trim()).filter(Boolean);
              return (
                <Space wrap>
                  {tags.map((tag) => (
                    <Tag key={tag}>{tag}</Tag>
                  ))}
                </Space>
              );
            }}
          </Form.Item>
          <Button type="primary" htmlType="submit" loading={profileMutation.isPending}>
            Profili yadda saxla
          </Button>
        </Form>

        <div style={{ marginTop: 24 }}>
          <h4>Daxili layihələr</h4>
          <List
            size="small"
            dataSource={subprojects ?? []}
            renderItem={(sub: Subproject) => (
              <List.Item
                actions={[
                  <Button
                    key="remove"
                    size="small"
                    danger
                    onClick={() => removeSubprojectMutation.mutate(sub.id)}
                  >
                    Sil
                  </Button>,
                ]}
              >
                <List.Item.Meta title={sub.name} description={sub.description} />
              </List.Item>
            )}
          />
          <Form
            form={subprojectForm}
            layout="inline"
            style={{ marginTop: 12 }}
            onFinish={(values) => addSubprojectMutation.mutate(values)}
          >
            <Form.Item name="name" rules={[{ required: true, message: 'Ad tələb olunur' }]}>
              <Input placeholder="Ad (məs. auth-service)" />
            </Form.Item>
            <Form.Item name="description">
              <Input placeholder="Təsvir" />
            </Form.Item>
            <Form.Item>
              <Button htmlType="submit" loading={addSubprojectMutation.isPending}>
                Əlavə et
              </Button>
            </Form.Item>
          </Form>
        </div>
      </Modal>
    </>
  );
}
```

- [ ] **Step 2: Type-check and build the frontend**

Run: `cd platform-identity-admin && npx tsc --noEmit && npm run build`
Expected: no type errors, build succeeds.

- [ ] **Step 3: Manual browser smoke test**

Start both services (per this project's established local-run commands from earlier in this session) and in the browser:
1. Log in as superadmin.
2. Go to Products page, click "Profil" on an existing product (e.g. Teslahubs).
3. Enter a description and tech stack (`React, Go, Postgres`), confirm Tags render live below the input.
4. Click "Profili yadda saxla", confirm the success message and that re-opening the modal shows the saved values.
5. Add a subproject (name `auth-service`, description `handles login`), confirm it appears in the list immediately.
6. Click "Sil" on it, confirm it disappears.
7. Refresh the page and re-open the same product's Profil modal — confirm the profile and (now empty) subprojects list persisted correctly across a reload.

- [ ] **Step 4: Commit**

```bash
git add platform-identity-admin/src/pages/ProductsPage.tsx
git commit -m "feat(platform-identity-admin): add product profile and subprojects modal to Products page

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

## Self-Review Notes

- **Spec coverage:** migration (Task 1), models (Task 2), repository incl. `Update`/`SubprojectRepository` (Task 3), service incl. all 4 superadmin-gated methods (Task 4), handler incl. all 4 routes + extended `productResponse` (Task 5), main.go wiring (Task 6), frontend types+API client (Task 7), frontend modal with description/tech-stack/subprojects list UI (Task 8) — every section of `2026-09-07-product-profile-design.md` has a corresponding task.
- **Explicitly out of scope per spec, correctly NOT implemented:** external tech-stack/subproject discovery, in-place subproject editing, tech-stack vocabulary constraints — none of the tasks above add these.
- **Type consistency check:** `NewProductService(productRepo, subRepo, subprojRepo)` signature is introduced in Task 4 and its only two call sites (Task 4's own test helper, Task 6's `main.go`) both use the 3-arg form. `subprojectResponse`/`Subproject` field names (`id`, `product_id`, `name`, `description`, `created_at`) are identical across Task 5 (Go JSON tags) and Task 7 (TS interface) — required for the frontend to deserialize correctly.
