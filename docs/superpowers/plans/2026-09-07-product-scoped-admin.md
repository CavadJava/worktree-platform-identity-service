# Product-Scoped Admin + Admin-Request Workflow Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let an `admin` (not just `superadmin`) manage the products they hold a subscription to from the admin panel, narrow their panel view to Products-only, and add a request/approval workflow for promoting an existing `user` to `admin` for a specific product — gated so only `superadmin` can decide it.

**Architecture:** Reuse the existing `system_role` + `user_product_subscriptions` tables as the entire authorization model (no new role table). Loosen `ProductService`'s existing superadmin-only checks to "superadmin, or admin with a subscription to this product." Add one new table, `product_admin_requests`, backing a small request/approve/promote-directly subsystem inside `ProductService`. Add a narrow, admin-safe user-listing endpoint so an `admin` can pick a subject user without exposing full shop-membership data. Split the frontend's single `RequireSystemAdmin` gate into a superadmin-only gate (Shops, Users) and a looser admin-or-above gate (Products), and scope the Products page's data and actions to what the caller is allowed to do.

**Tech Stack:** Go (chi router, database/sql + pgx), Postgres, React + Vite + AntDesign + TanStack Query, axios.

## Global Constraints

- Every migration statement MUST be idempotent/additive: `CREATE TABLE IF NOT EXISTS`, no `DROP` — this branch had a real data-loss incident from a non-idempotent migration (see `platform-identity-service/internal/database/postgres.go`'s `Migrate` doc comment).
- No new role table — "product access" stays `user_product_subscriptions`; "can administer" is `system_role ∈ {admin, superadmin}` plus holding a subscription.
- Only `superadmin` may ever set `system_role = admin` on an *existing* user — directly, or by approving a request. An `admin` can never do this unilaterally for an existing account.
- Choosing `user` vs `admin` for a **brand-new** account (via "Yeni istifadəçi") is exempt from the request/approval path — it only affects an account that doesn't exist yet, so anyone who can open that modal (admin or superadmin, for a product they manage) can pick either role directly.
- A request can only target a product the subject does not already hold a subscription to.
- Rejecting a request leaves the subject's role/subscriptions untouched and does not block a future new request for the same pair.

---

### Task 1: Add a caller-scope helper to `ProductService` and loosen its existing superadmin-only gates

**Files:**
- Modify: `platform-identity-service/internal/service/product_service.go`
- Test: `platform-identity-service/internal/service/product_service_test.go`

**Interfaces:**
- Consumes: `repository.SubscriptionRepository.GetByUserAndProduct(ctx, userID, productID) (*models.Subscription, error)` (existing, returns `repository.ErrSubscriptionNotFound` if none).
- Produces: unexported `(s *ProductService) canManage(ctx context.Context, caller shopassign.Caller, productID string) error` — returns `nil` if caller is superadmin, or if caller is admin AND holds a subscription to productID; returns `ErrForbidden` otherwise. `UpdateProfile`, `AddSubproject`, `RemoveSubproject`, `SetSubscription`, `CreateUserAndSubscribe` all call this instead of their current `if caller.SystemRole != models.SystemRoleSuperadmin` check. Consumed by every later task that adds a new gated method.

- [ ] **Step 1: Write the failing test**

Add to `platform-identity-service/internal/service/product_service_test.go`:

```go
func TestProductService_AdminWithSubscriptionCanManageTheirProduct(t *testing.T) {
	svc := newTestProductService(t)
	p, err := svc.Create(context.Background(), "Teslahubs")
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	superadmin := shopassign.Caller{SystemRole: models.SystemRoleSuperadmin}
	admin := shopassign.Caller{UserID: "admin-" + p.ID, SystemRole: models.SystemRoleAdmin}

	// Admin has no subscription yet — must be forbidden.
	if _, err := svc.UpdateProfile(context.Background(), admin, p.ID, "desc", "Go"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden before subscription exists, got %v", err)
	}

	// Superadmin subscribes the admin to the product.
	if _, err := svc.SetSubscription(context.Background(), superadmin, admin.UserID, p.ID, true, false, ""); err != nil {
		t.Fatalf("subscribe admin: %v", err)
	}

	// Now the admin can manage it.
	updated, err := svc.UpdateProfile(context.Background(), admin, p.ID, "A Tesla platform", "Go, React")
	if err != nil {
		t.Fatalf("expected admin with subscription to manage product, got %v", err)
	}
	if updated.Description != "A Tesla platform" {
		t.Fatalf("expected profile updated, got %+v", updated)
	}

	// A different admin, still unsubscribed, is still forbidden.
	otherAdmin := shopassign.Caller{UserID: "other-" + p.ID, SystemRole: models.SystemRoleAdmin}
	if _, err := svc.UpdateProfile(context.Background(), otherAdmin, p.ID, "x", "y"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for unsubscribed admin, got %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd platform-identity-service && go test ./internal/service/... -run TestProductService_AdminWithSubscriptionCanManageTheirProduct -v`
Expected: FAIL — `UpdateProfile` still rejects the subscribed admin with `ErrForbidden` (current code only allows superadmin).

- [ ] **Step 3: Write minimal implementation**

In `platform-identity-service/internal/service/product_service.go`, add this method right after `NewProductService`:

```go
// canManage reports whether caller may administer productID: a superadmin
// always may; an admin may only for a product they hold a subscription to.
// A plain user is never allowed, and never reaches this far in practice
// since the HTTP layer requires an authenticated caller with a system role
// of admin or superadmin to reach any of these handlers at all.
func (s *ProductService) canManage(ctx context.Context, caller shopassign.Caller, productID string) error {
	if caller.SystemRole == models.SystemRoleSuperadmin {
		return nil
	}
	if caller.SystemRole != models.SystemRoleAdmin {
		return ErrForbidden
	}
	_, err := s.subRepo.GetByUserAndProduct(ctx, caller.UserID, productID)
	if errors.Is(err, repository.ErrSubscriptionNotFound) {
		return ErrForbidden
	}
	if err != nil {
		return err
	}
	return nil
}
```

Then replace each of these five `if caller.SystemRole != models.SystemRoleSuperadmin { return ..., ErrForbidden }` checks with a call to `canManage`:

In `UpdateProfile`:
```go
func (s *ProductService) UpdateProfile(ctx context.Context, caller shopassign.Caller, productID, description, techStack string) (*models.Product, error) {
	if err := s.canManage(ctx, caller, productID); err != nil {
		return nil, err
	}
	if err := s.productRepo.Update(ctx, productID, description, techStack); err != nil {
		return nil, err
	}
	return s.productRepo.GetByID(ctx, productID)
}
```

In `AddSubproject`:
```go
func (s *ProductService) AddSubproject(ctx context.Context, caller shopassign.Caller, productID, name, description string) (*models.ProductSubproject, error) {
	if err := s.canManage(ctx, caller, productID); err != nil {
		return nil, err
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
```

`RemoveSubproject` takes a subproject ID, not a product ID, so it needs the subproject's product looked up first:
```go
func (s *ProductService) RemoveSubproject(ctx context.Context, caller shopassign.Caller, subprojectID string) error {
	sub, err := s.subprojRepo.GetByID(ctx, subprojectID)
	if err != nil {
		return err
	}
	if err := s.canManage(ctx, caller, sub.ProductID); err != nil {
		return err
	}
	return s.subprojRepo.Delete(ctx, subprojectID)
}
```

This requires a new `SubprojectRepository.GetByID` method — add it to
`platform-identity-service/internal/repository/subproject_repository.go`:
```go
func (r *SubprojectRepository) GetByID(ctx context.Context, id string) (*models.ProductSubproject, error) {
	const q = `SELECT id, product_id, name, description, created_at FROM product_subprojects WHERE id = $1`
	var s models.ProductSubproject
	err := r.db.QueryRowContext(ctx, q, id).Scan(&s.ID, &s.ProductID, &s.Name, &s.Description, &s.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSubprojectNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}
```
(This file already imports `database/sql` and `errors` — check the top of the file before adding; if not, add them to the existing import block.)

In `SetSubscription`:
```go
func (s *ProductService) SetSubscription(ctx context.Context, caller shopassign.Caller, targetUserID, productID string, subscripted, renewed bool, notes string) (*models.Subscription, error) {
	if err := s.canManage(ctx, caller, productID); err != nil {
		return nil, err
	}
	if _, err := s.productRepo.GetByID(ctx, productID); err != nil {
		return nil, err
	}
	// ... rest unchanged
```

In `CreateUserAndSubscribe`:
```go
func (s *ProductService) CreateUserAndSubscribe(ctx context.Context, caller shopassign.Caller, productID string, in CreateUserInput) (*models.Subscription, error) {
	if err := s.canManage(ctx, caller, productID); err != nil {
		return nil, err
	}
	if _, err := s.productRepo.GetByID(ctx, productID); err != nil {
		return nil, err
	}
	// ... rest unchanged (in.SystemRole assignment covered by Task 6)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd platform-identity-service && go test ./internal/service/... -v`
Expected: all PASS, including the new test and every pre-existing `ProductService` test (`TestProductService_UpdateProfile_RequiresSuperadmin` and similar names must be renamed or updated if they assert plain-`ErrForbidden`-for-any-non-superadmin — check for tests asserting a plain `admin` caller is forbidden with NO subscription set up; those should still pass unchanged since an admin with no subscription is still forbidden).

- [ ] **Step 5: Commit**

```bash
git add platform-identity-service/internal/service/product_service.go platform-identity-service/internal/service/product_service_test.go platform-identity-service/internal/repository/subproject_repository.go
git commit -m "feat(platform-identity-service): let an admin manage products they hold a subscription to

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 2: Add a narrow, admin-safe user-picker endpoint

**Files:**
- Modify: `platform-identity-service/internal/service/user_service.go`
- Modify: `platform-identity-service/internal/handlers/user_handler.go`
- Modify: `platform-identity-service/cmd/api/main.go`
- Test: `platform-identity-service/internal/service/user_service_test.go`
- Test: `platform-identity-service/internal/handlers/auth_handler_test.go` (or a new `user_handler_test.go` — check whether one already exists with `ls platform-identity-service/internal/handlers/user_handler_test.go` first)

**Interfaces:**
- Consumes: `repository.UserRepository.ListAll` (existing).
- Produces: `UserService.ListBasic(ctx context.Context, caller shopassign.Caller) ([]models.User, error)` — allowed for `admin` or `superadmin` (unlike `ListAll`, which stays `superadmin`-only); `UserHandler.ListBasic` handler; route `GET /users/basic`. Consumed by Task 8 (frontend user-picker for the Subscription/admin-request modals).

- [ ] **Step 1: Write the failing test**

Add to `platform-identity-service/internal/service/user_service_test.go` (check the file's existing helper name for constructing a `UserService` — likely `newTestUserService(t)` matching the pattern in `product_service_test.go`; if it constructs the service differently, follow that file's own existing convention instead of introducing a new one):

```go
func TestUserService_ListBasic_AllowsAdminNotJustSuperadmin(t *testing.T) {
	db := testDB(t)
	repo := repository.NewUserRepository(db)
	svc := NewUserService(repo)

	admin := shopassign.Caller{SystemRole: models.SystemRoleAdmin}
	users, err := svc.ListBasic(context.Background(), admin)
	if err != nil {
		t.Fatalf("expected admin to list basic users, got %v", err)
	}
	_ = users

	plainUser := shopassign.Caller{SystemRole: models.SystemRoleUser}
	if _, err := svc.ListBasic(context.Background(), plainUser); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for plain user, got %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd platform-identity-service && go test ./internal/service/... -run TestUserService_ListBasic_AllowsAdminNotJustSuperadmin -v`
Expected: FAIL to compile — `ListBasic` undefined.

- [ ] **Step 3: Write minimal implementation**

In `platform-identity-service/internal/service/user_service.go`, add after `ListAll`:

```go
// ListBasic is a narrower listing than ListAll: no shop-membership data,
// available to admin as well as superadmin — an admin managing a product
// needs to pick a subject user (for a subscription or an admin-request)
// without gaining visibility into everyone's shop memberships, which
// ListAll's superadmin-only gate exists specifically to protect.
func (s *UserService) ListBasic(ctx context.Context, caller shopassign.Caller) ([]models.User, error) {
	if caller.SystemRole != models.SystemRoleAdmin && caller.SystemRole != models.SystemRoleSuperadmin {
		return nil, ErrForbidden
	}
	return s.repo.ListAll(ctx)
}
```

In `platform-identity-service/internal/handlers/user_handler.go`, add a new response type and handler after `ListAll`:

```go
type userBasicResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Username   string `json:"username"`
	Email      string `json:"email"`
	SystemRole string `json:"system_role"`
}

// ListBasic godoc
// @Summary      List every user, without shop-membership data
// @Description  Admin or superadmin.
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} userBasicResponse
// @Failure      403 {object} map[string]string
// @Router       /users/basic [get]
func (h *UserHandler) ListBasic(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	users, err := h.svc.ListBasic(r.Context(), caller)
	if err != nil {
		writeUserServiceError(w, err)
		return
	}

	response := make([]userBasicResponse, len(users))
	for i, u := range users {
		response[i] = userBasicResponse{ID: u.ID, Name: u.Name, Username: u.Username, Email: u.Email, SystemRole: u.SystemRoleName}
	}
	writeJSON(w, http.StatusOK, response)
}
```

In `platform-identity-service/cmd/api/main.go`, add the route right after the existing `r.Get("/users", userHandler.ListAll)` line (find it with `grep -n '"/users"' cmd/api/main.go` — it's inside the authenticated `r.Group`):

```go
r.Get("/users/basic", userHandler.ListBasic)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd platform-identity-service && go test ./... -v 2>&1 | tail -40`
Expected: all PASS, including the new `TestUserService_ListBasic_AllowsAdminNotJustSuperadmin`.

- [ ] **Step 5: Commit**

```bash
git add platform-identity-service/internal/service/user_service.go platform-identity-service/internal/service/user_service_test.go platform-identity-service/internal/handlers/user_handler.go platform-identity-service/cmd/api/main.go
git commit -m "feat(platform-identity-service): add admin-safe basic user list for product-scoped pickers

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 3: `product_admin_requests` migration + model

**Files:**
- Modify: `platform-identity-service/internal/database/postgres.go`
- Modify: `platform-identity-service/internal/models/models.go`
- Test: `platform-identity-service/internal/database/postgres_test.go`

**Interfaces:**
- Produces: `product_admin_requests` table (`id, product_id, subject_user_id, requested_by_user_id, status, created_at, decided_at, decided_by_user_id`); `models.ProductAdminRequest{ID, ProductID, SubjectUserID, RequestedByUserID, Status string; CreatedAt time.Time; DecidedAt *time.Time; DecidedByUserID *string}`. Consumed by Task 4's repository.

- [ ] **Step 1: Write the failing test**

Add to `platform-identity-service/internal/database/postgres_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd platform-identity-service && go test ./internal/database/... -run TestMigrate_ProductAdminRequestsTable -v`
Expected: FAIL — `relation "product_admin_requests" does not exist`.

- [ ] **Step 3: Write minimal implementation**

In `platform-identity-service/internal/database/postgres.go`, inside `Migrate`'s SQL block, add this after the `product_subprojects` table's `CREATE INDEX` line and before `user_product_subscriptions`:

```go
		CREATE TABLE IF NOT EXISTS product_admin_requests (
			id UUID PRIMARY KEY,
			product_id UUID NOT NULL REFERENCES products(id),
			subject_user_id UUID NOT NULL REFERENCES users(id),
			requested_by_user_id UUID NOT NULL REFERENCES users(id),
			status TEXT NOT NULL DEFAULT 'pending',
			created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			decided_at TIMESTAMPTZ,
			decided_by_user_id UUID REFERENCES users(id)
		);
		CREATE INDEX IF NOT EXISTS idx_product_admin_requests_product_id ON product_admin_requests (product_id);
		CREATE INDEX IF NOT EXISTS idx_product_admin_requests_status ON product_admin_requests (status);
```

In `platform-identity-service/internal/models/models.go`, add after `ProductSubproject`:

```go
const (
	ProductAdminRequestPending  = "pending"
	ProductAdminRequestApproved = "approved"
	ProductAdminRequestRejected = "rejected"
)

type ProductAdminRequest struct {
	ID                 string
	ProductID          string
	SubjectUserID      string
	RequestedByUserID  string
	Status             string
	CreatedAt          time.Time
	DecidedAt          *time.Time
	DecidedByUserID    *string
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd platform-identity-service && go test ./internal/database/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add platform-identity-service/internal/database/postgres.go platform-identity-service/internal/database/postgres_test.go platform-identity-service/internal/models/models.go
git commit -m "feat(platform-identity-service): add product_admin_requests table and model

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 4: `ProductAdminRequestRepository`

**Files:**
- Create: `platform-identity-service/internal/repository/product_admin_request_repository.go`
- Test: `platform-identity-service/internal/repository/repository_test.go`

**Interfaces:**
- Consumes: `models.ProductAdminRequest` (Task 3).
- Produces: `NewProductAdminRequestRepository(db *sql.DB) *ProductAdminRequestRepository`, `Create(ctx, r *models.ProductAdminRequest) error`, `GetByID(ctx, id string) (*models.ProductAdminRequest, error)`, `ListPendingByProduct(ctx, productID string) ([]models.ProductAdminRequest, error)`, `HasPending(ctx, productID, subjectUserID string) (bool, error)`, `SetStatus(ctx, id, status, decidedByUserID string) error`, `var ErrProductAdminRequestNotFound = errors.New(...)`. Consumed by Task 5's `ProductService`.

- [ ] **Step 1: Write the failing tests**

Append to `platform-identity-service/internal/repository/repository_test.go` (same-package `repository`, matching the file's existing convention — no import alias, `defer db.Close()`):

```go
func TestProductAdminRequestRepository_CreateListDecide(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	userRepo := NewUserRepository(db)
	productRepo := NewProductRepository(db)
	reqRepo := NewProductAdminRequestRepository(db)

	p := &models.Product{ID: uuid.NewString(), Name: "Req Product " + uuid.NewString(), CreatedAt: time.Now().UTC()}
	if err := productRepo.Create(context.Background(), p); err != nil {
		t.Fatalf("create product: %v", err)
	}
	subject := createTestUser(t, userRepo, 3)
	requester := createTestUser(t, userRepo, 2)

	hasPending, err := reqRepo.HasPending(context.Background(), p.ID, subject.ID)
	if err != nil {
		t.Fatalf("has pending (before): %v", err)
	}
	if hasPending {
		t.Fatal("expected no pending request before one is created")
	}

	req := &models.ProductAdminRequest{
		ID: uuid.NewString(), ProductID: p.ID, SubjectUserID: subject.ID, RequestedByUserID: requester.ID,
		Status: models.ProductAdminRequestPending, CreatedAt: time.Now().UTC(),
	}
	if err := reqRepo.Create(context.Background(), req); err != nil {
		t.Fatalf("create request: %v", err)
	}

	hasPending, err = reqRepo.HasPending(context.Background(), p.ID, subject.ID)
	if err != nil {
		t.Fatalf("has pending (after): %v", err)
	}
	if !hasPending {
		t.Fatal("expected a pending request after creating one")
	}

	pending, err := reqRepo.ListPendingByProduct(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("list pending: %v", err)
	}
	if len(pending) != 1 || pending[0].SubjectUserID != subject.ID {
		t.Fatalf("expected one pending request for subject, got %+v", pending)
	}

	if err := reqRepo.SetStatus(context.Background(), req.ID, models.ProductAdminRequestApproved, requester.ID); err != nil {
		t.Fatalf("set status: %v", err)
	}

	got, err := reqRepo.GetByID(context.Background(), req.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if got.Status != models.ProductAdminRequestApproved {
		t.Fatalf("expected status approved, got %q", got.Status)
	}
	if got.DecidedByUserID == nil || *got.DecidedByUserID != requester.ID {
		t.Fatalf("expected decided_by_user_id set, got %+v", got.DecidedByUserID)
	}

	hasPending, err = reqRepo.HasPending(context.Background(), p.ID, subject.ID)
	if err != nil {
		t.Fatalf("has pending (after decide): %v", err)
	}
	if hasPending {
		t.Fatal("expected no pending request after it was decided")
	}
}
```

This test uses `createTestUser(t, userRepo, roleID)` — check that this helper already exists in `repository_test.go` (it's referenced by `TestSubscriptionRepository_UpsertAndGet` at line 400 per the current file) before assuming its exact signature; if it takes different arguments, match the file's real signature instead.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd platform-identity-service && go test ./internal/repository/... -run TestProductAdminRequestRepository_CreateListDecide -v`
Expected: FAIL to compile — `NewProductAdminRequestRepository` undefined.

- [ ] **Step 3: Write minimal implementation**

Create `platform-identity-service/internal/repository/product_admin_request_repository.go`:

```go
package repository

import (
	"context"
	"database/sql"
	"errors"

	"platform-identity-service/internal/models"
)

var ErrProductAdminRequestNotFound = errors.New("product admin request not found")

type ProductAdminRequestRepository struct {
	db *sql.DB
}

func NewProductAdminRequestRepository(db *sql.DB) *ProductAdminRequestRepository {
	return &ProductAdminRequestRepository{db: db}
}

func (r *ProductAdminRequestRepository) Create(ctx context.Context, req *models.ProductAdminRequest) error {
	const q = `
		INSERT INTO product_admin_requests (id, product_id, subject_user_id, requested_by_user_id, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, q, req.ID, req.ProductID, req.SubjectUserID, req.RequestedByUserID, req.Status, req.CreatedAt)
	return err
}

const selectProductAdminRequest = `
	SELECT id, product_id, subject_user_id, requested_by_user_id, status, created_at, decided_at, decided_by_user_id
	FROM product_admin_requests
`

func scanProductAdminRequest(row *sql.Row) (*models.ProductAdminRequest, error) {
	var req models.ProductAdminRequest
	err := row.Scan(&req.ID, &req.ProductID, &req.SubjectUserID, &req.RequestedByUserID, &req.Status, &req.CreatedAt, &req.DecidedAt, &req.DecidedByUserID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrProductAdminRequestNotFound
	}
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *ProductAdminRequestRepository) GetByID(ctx context.Context, id string) (*models.ProductAdminRequest, error) {
	row := r.db.QueryRowContext(ctx, selectProductAdminRequest+" WHERE id = $1", id)
	return scanProductAdminRequest(row)
}

func (r *ProductAdminRequestRepository) ListPendingByProduct(ctx context.Context, productID string) ([]models.ProductAdminRequest, error) {
	rows, err := r.db.QueryContext(ctx, selectProductAdminRequest+" WHERE product_id = $1 AND status = 'pending' ORDER BY created_at", productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	requests := []models.ProductAdminRequest{}
	for rows.Next() {
		var req models.ProductAdminRequest
		if err := rows.Scan(&req.ID, &req.ProductID, &req.SubjectUserID, &req.RequestedByUserID, &req.Status, &req.CreatedAt, &req.DecidedAt, &req.DecidedByUserID); err != nil {
			return nil, err
		}
		requests = append(requests, req)
	}
	return requests, rows.Err()
}

func (r *ProductAdminRequestRepository) HasPending(ctx context.Context, productID, subjectUserID string) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM product_admin_requests WHERE product_id = $1 AND subject_user_id = $2 AND status = 'pending')`
	var exists bool
	err := r.db.QueryRowContext(ctx, q, productID, subjectUserID).Scan(&exists)
	return exists, err
}

func (r *ProductAdminRequestRepository) SetStatus(ctx context.Context, id, status, decidedByUserID string) error {
	const q = `UPDATE product_admin_requests SET status = $2, decided_at = now(), decided_by_user_id = $3 WHERE id = $1`
	result, err := r.db.ExecContext(ctx, q, id, status, decidedByUserID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrProductAdminRequestNotFound
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd platform-identity-service && go test ./internal/repository/... -v`
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add platform-identity-service/internal/repository/product_admin_request_repository.go platform-identity-service/internal/repository/repository_test.go
git commit -m "feat(platform-identity-service): add ProductAdminRequestRepository

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 5: `ProductService` request/decide/promote methods

**Files:**
- Modify: `platform-identity-service/internal/service/product_service.go`
- Test: `platform-identity-service/internal/service/product_service_test.go`

**Interfaces:**
- Consumes: `repository.ProductAdminRequestRepository` (Task 4), `repository.UserRepository.SetSystemRole(ctx, userID string, systemRoleID int16) error` (existing — note this is the *repository*-layer method, called directly since `ProductService` doesn't hold a `UserService`; a small map is needed to convert `models.SystemRoleAdmin` to its role ID), `subRepo.Upsert` (existing).
- Produces: `ProductService.RequestProductAdmin(ctx, caller, productID, subjectUserID string) (*models.ProductAdminRequest, error)`, `ListPendingRequests(ctx, caller, productID string) ([]models.ProductAdminRequest, error)`, `DecideRequest(ctx, caller, requestID string, approve bool) (*models.ProductAdminRequest, error)`, `PromoteDirectly(ctx, caller, productID, subjectUserID string) (*models.Subscription, error)`. New `var ErrAlreadyHasAccess = errors.New(...)`, `var ErrRequestAlreadyPending = errors.New(...)`. Consumed by Task 6's handler.

- [ ] **Step 1: Write the failing tests**

Add to `platform-identity-service/internal/service/product_service_test.go`:

```go
func TestProductService_RequestApprovePromoteFlow(t *testing.T) {
	svc := newTestProductService(t)
	p, err := svc.Create(context.Background(), "Teslahubs")
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	superadmin := shopassign.Caller{SystemRole: models.SystemRoleSuperadmin}

	// Create a plain 'user' subject via the existing product-scoped creation path.
	newUserSub, err := svc.CreateUserAndSubscribe(context.Background(), superadmin, p.ID, CreateUserInput{
		Name: "Subject", Username: "subject-" + p.ID, Email: "subject-" + p.ID + "@example.com", Password: "password123",
	}, models.SystemRoleUser)
	if err != nil {
		t.Fatalf("create subject user: %v", err)
	}
	_ = newUserSub

	// The subject already has a subscription (from CreateUserAndSubscribe), so
	// build a second, unsubscribed target product to test the request flow on.
	p2, err := svc.Create(context.Background(), "ESound")
	if err != nil {
		t.Fatalf("create second product: %v", err)
	}

	admin := shopassign.Caller{SystemRole: models.SystemRoleAdmin}
	// admin has no subscription to p2 yet, so filing a request must fail.
	if _, err := svc.RequestProductAdmin(context.Background(), admin, p2.ID, "some-user-id"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for admin unsubscribed to the product, got %v", err)
	}

	// Subscribe the admin to p2, then they can file a request for someone else.
	if _, err := svc.SetSubscription(context.Background(), superadmin, admin.UserID, p2.ID, true, false, ""); err != nil {
		t.Fatalf("subscribe admin to p2: %v", err)
	}

	req, err := svc.RequestProductAdmin(context.Background(), admin, p2.ID, newUserSub.UserID)
	if err != nil {
		t.Fatalf("request product admin: %v", err)
	}
	if req.Status != models.ProductAdminRequestPending {
		t.Fatalf("expected pending status, got %q", req.Status)
	}

	// A second request for the same pending pair must fail.
	if _, err := svc.RequestProductAdmin(context.Background(), admin, p2.ID, newUserSub.UserID); !errors.Is(err, ErrRequestAlreadyPending) {
		t.Fatalf("expected ErrRequestAlreadyPending, got %v", err)
	}

	// A plain admin cannot decide.
	if _, err := svc.DecideRequest(context.Background(), admin, req.ID, true); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for admin deciding, got %v", err)
	}

	decided, err := svc.DecideRequest(context.Background(), superadmin, req.ID, true)
	if err != nil {
		t.Fatalf("decide request: %v", err)
	}
	if decided.Status != models.ProductAdminRequestApproved {
		t.Fatalf("expected approved status, got %q", decided.Status)
	}

	access, err := svc.CheckAccess(context.Background(), newUserSub.UserID, p2.ID)
	if err != nil {
		t.Fatalf("check access after approval: %v", err)
	}
	if access != AccessFull {
		t.Fatalf("expected full access to p2 after approval, got %q", access)
	}
}

func TestProductService_PromoteDirectly_SuperadminOnly(t *testing.T) {
	svc := newTestProductService(t)
	p, err := svc.Create(context.Background(), "Vault")
	if err != nil {
		t.Fatalf("create product: %v", err)
	}
	superadmin := shopassign.Caller{SystemRole: models.SystemRoleSuperadmin}
	admin := shopassign.Caller{SystemRole: models.SystemRoleAdmin}

	sub, err := svc.CreateUserAndSubscribe(context.Background(), superadmin, p.ID, CreateUserInput{
		Name: "Direct Subject", Username: "direct-" + p.ID, Email: "direct-" + p.ID + "@example.com", Password: "password123",
	}, models.SystemRoleUser)
	if err != nil {
		t.Fatalf("create subject: %v", err)
	}

	if _, err := svc.PromoteDirectly(context.Background(), admin, p.ID, sub.UserID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden for admin calling PromoteDirectly, got %v", err)
	}

	if _, err := svc.PromoteDirectly(context.Background(), superadmin, p.ID, sub.UserID); err != nil {
		t.Fatalf("promote directly: %v", err)
	}
}
```

Note: this task changes `CreateUserAndSubscribe`'s signature to take a trailing `systemRole string` param, which Task 6 also needs — write that signature change now since these tests already assume it (`svc.CreateUserAndSubscribe(ctx, caller, productID, in, models.SystemRoleUser)`), rather than in Task 6, to avoid this task's tests failing to compile. Task 6 will only need to wire the request body field through to the already-updated service call.

- [ ] **Step 2: Run test to verify it fails**

Run: `cd platform-identity-service && go test ./internal/service/... -run 'TestProductService_RequestApprovePromoteFlow|TestProductService_PromoteDirectly_SuperadminOnly' -v`
Expected: FAIL to compile — `CreateUserAndSubscribe` takes 4 args not 5, `RequestProductAdmin`/`DecideRequest`/`PromoteDirectly` undefined, `ErrRequestAlreadyPending` undefined.

- [ ] **Step 3: Write minimal implementation**

First, update `NewProductService` and the struct to hold the new repository and a `userRepo` reference (needed for the direct `SetSystemRole` role-ID lookup):

```go
type ProductService struct {
	productRepo *repository.ProductRepository
	subRepo     *repository.SubscriptionRepository
	subprojRepo *repository.SubprojectRepository
	reqRepo     *repository.ProductAdminRequestRepository
	userRepo    *repository.UserRepository
	authSvc     *AuthService
}

func NewProductService(
	productRepo *repository.ProductRepository,
	subRepo *repository.SubscriptionRepository,
	subprojRepo *repository.SubprojectRepository,
	reqRepo *repository.ProductAdminRequestRepository,
	userRepo *repository.UserRepository,
	authSvc *AuthService,
) *ProductService {
	return &ProductService{
		productRepo: productRepo, subRepo: subRepo, subprojRepo: subprojRepo,
		reqRepo: reqRepo, userRepo: userRepo, authSvc: authSvc,
	}
}
```

Add near the top, alongside `ErrProductNotFound`:

```go
var (
	ErrAlreadyHasAccess     = errors.New("subject already has access to this product")
	ErrRequestAlreadyPending = errors.New("a pending request already exists for this user and product")
)
```

Change `CreateUserAndSubscribe`'s signature and body (this also satisfies Task 6's later needs — the role choice is exempt from request/approval per the Global Constraints):

```go
// CreateUserAndSubscribe creates a brand-new Teslahubs account and
// immediately subscribes it to productID. systemRole is the new account's
// starting role ("user" or "admin") — chosen freely by whoever creates the
// account (admin or superadmin managing this product), since a brand-new
// account has no prior state the request/approval path exists to protect;
// that path is specifically for elevating an *existing* user.
func (s *ProductService) CreateUserAndSubscribe(ctx context.Context, caller shopassign.Caller, productID string, in CreateUserInput, systemRole string) (*models.Subscription, error) {
	if err := s.canManage(ctx, caller, productID); err != nil {
		return nil, err
	}
	if _, err := s.productRepo.GetByID(ctx, productID); err != nil {
		return nil, err
	}

	in.SystemRole = systemRole
	u, err := s.authSvc.CreateUser(ctx, in)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	sub := &models.Subscription{
		ID: uuid.NewString(), UserID: u.ID, ProductID: productID,
		Subscripted: true, Renewed: false,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := s.subRepo.Upsert(ctx, sub); err != nil {
		return nil, err
	}
	return sub, nil
}
```

Add the request/decide/promote methods at the end of the file:

```go
// systemRoleNameToID mirrors the same map defined in user_service.go —
// duplicated here (rather than exported and imported across files) since
// both live in the same `service` package already; Go allows only one
// definition per package, so if user_service.go already declares a
// package-level systemRoleNameToID, reuse that one directly instead of
// redeclaring it here.
func (s *ProductService) promoteToAdminAndSubscribe(ctx context.Context, subjectUserID, productID string) (*models.Subscription, error) {
	roleID, ok := systemRoleNameToID[models.SystemRoleAdmin]
	if !ok {
		return nil, ErrInvalidSystemRole
	}
	if err := s.userRepo.SetSystemRole(ctx, subjectUserID, roleID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	existing, err := s.subRepo.GetByUserAndProduct(ctx, subjectUserID, productID)
	id := uuid.NewString()
	createdAt := now
	if err == nil {
		id = existing.ID
		createdAt = existing.CreatedAt
	} else if !errors.Is(err, repository.ErrSubscriptionNotFound) {
		return nil, err
	}

	sub := &models.Subscription{
		ID: id, UserID: subjectUserID, ProductID: productID,
		Subscripted: true, Renewed: false,
		CreatedAt: createdAt, UpdatedAt: now,
	}
	if err := s.subRepo.Upsert(ctx, sub); err != nil {
		return nil, err
	}
	return sub, nil
}

// RequestProductAdmin lets an admin (managing productID) or superadmin file
// a request to make subjectUserID an admin for productID. Only ever creates
// the request row — the actual role/subscription change happens in
// DecideRequest (or PromoteDirectly, which skips the request entirely).
func (s *ProductService) RequestProductAdmin(ctx context.Context, caller shopassign.Caller, productID, subjectUserID string) (*models.ProductAdminRequest, error) {
	if err := s.canManage(ctx, caller, productID); err != nil {
		return nil, err
	}
	if _, err := s.subRepo.GetByUserAndProduct(ctx, subjectUserID, productID); err == nil {
		return nil, ErrAlreadyHasAccess
	} else if !errors.Is(err, repository.ErrSubscriptionNotFound) {
		return nil, err
	}
	hasPending, err := s.reqRepo.HasPending(ctx, productID, subjectUserID)
	if err != nil {
		return nil, err
	}
	if hasPending {
		return nil, ErrRequestAlreadyPending
	}

	req := &models.ProductAdminRequest{
		ID: uuid.NewString(), ProductID: productID, SubjectUserID: subjectUserID,
		RequestedByUserID: caller.UserID, Status: models.ProductAdminRequestPending,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.reqRepo.Create(ctx, req); err != nil {
		return nil, err
	}
	return req, nil
}

// ListPendingRequests is superadmin-only — an admin sees the same list
// read-only through a different, unrestricted-read handler wiring (Task 6
// exposes GET .../admin-requests to admin and superadmin alike, but only
// superadmin can call DecideRequest).
func (s *ProductService) ListPendingRequests(ctx context.Context, productID string) ([]models.ProductAdminRequest, error) {
	return s.reqRepo.ListPendingByProduct(ctx, productID)
}

// DecideRequest is superadmin-only. Approving sets the subject's
// system_role to admin (if not already) and creates/updates their
// subscription to the product — the same end effect as PromoteDirectly.
func (s *ProductService) DecideRequest(ctx context.Context, caller shopassign.Caller, requestID string, approve bool) (*models.ProductAdminRequest, error) {
	if caller.SystemRole != models.SystemRoleSuperadmin {
		return nil, ErrForbidden
	}
	req, err := s.reqRepo.GetByID(ctx, requestID)
	if err != nil {
		return nil, err
	}

	status := models.ProductAdminRequestRejected
	if approve {
		status = models.ProductAdminRequestApproved
		if _, err := s.promoteToAdminAndSubscribe(ctx, req.SubjectUserID, req.ProductID); err != nil {
			return nil, err
		}
	}
	if err := s.reqRepo.SetStatus(ctx, requestID, status, caller.UserID); err != nil {
		return nil, err
	}
	return s.reqRepo.GetByID(ctx, requestID)
}

// PromoteDirectly is superadmin-only: the same end effect as an approved
// request, but with no request row created at all.
func (s *ProductService) PromoteDirectly(ctx context.Context, caller shopassign.Caller, productID, subjectUserID string) (*models.Subscription, error) {
	if caller.SystemRole != models.SystemRoleSuperadmin {
		return nil, ErrForbidden
	}
	return s.promoteToAdminAndSubscribe(ctx, subjectUserID, productID)
}
```

Before adding the `systemRoleNameToID` comment/reference above, check
`platform-identity-service/internal/service/user_service.go` — it already
declares `var systemRoleNameToID = map[string]int16{...}` at package level
(confirmed present when this plan was written), so `product_service.go`
does NOT redeclare it; it just uses the existing package-level variable
directly since both files are `package service`. Remove the comment about
"reuse that one" from the code itself — it was explanatory for this plan
only, not something to leave as an in-code comment.

Also check `ErrInvalidSystemRole` — it's already declared in
`user_service.go` (`var (... ErrInvalidSystemRole = errors.New(...))`),
same package, reuse directly.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd platform-identity-service && go test ./internal/service/... -v`
Expected: all PASS. If any pre-existing test still calls `CreateUserAndSubscribe` with the old 4-arg signature, update it to pass `models.SystemRoleUser` as the trailing argument (grep for every call site first: `grep -rn "CreateUserAndSubscribe(" internal/`).

- [ ] **Step 5: Commit**

```bash
git add platform-identity-service/internal/service/product_service.go platform-identity-service/internal/service/product_service_test.go
git commit -m "feat(platform-identity-service): add product-admin request/decide/promote flow

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 6: Handler routes for scoped listing, admin-requests, direct promote, and role choice on user creation

**Files:**
- Modify: `platform-identity-service/internal/handlers/product_handler.go`
- Test: `platform-identity-service/internal/handlers/product_handler_test.go`

**Interfaces:**
- Consumes: `ProductService.ListPendingRequests`, `RequestProductAdmin`, `DecideRequest`, `PromoteDirectly`, `CreateUserAndSubscribe` (now 5-arg) (Task 5); `repository.ErrAlreadyHasAccess`, `ErrRequestAlreadyPending`, `ErrProductAdminRequestNotFound` (Task 4/5).
- Produces: `ProductHandler.ListMine` (scoped list — superadmin sees all, admin sees only subscribed), `ListBrowse` (read-only id+name, any authenticated caller), `RequestAdmin`, `ListAdminRequests`, `DecideAdminRequest`, `PromoteAdmin` handler methods; `createProductUserRequest` gains a `SystemRole string` field; new response types `productAdminRequestResponse{id, product_id, subject_user_id, requested_by_user_id, status, created_at}`, `productBrowseResponse{id, name}`. Consumed by Task 7 (route wiring).

- [ ] **Step 1: Write the failing test**

Add to `platform-identity-service/internal/handlers/product_handler_test.go`:

```go
func TestProductHandler_ScopedListAndAdminRequestFlow(t *testing.T) {
	r, db, jwtManager := testProductRouter(t)
	superToken := createTestSuperadmin(t, db, jwtManager)

	// Create two products; only subscribe the admin to the first.
	rec := doJSON(t, r, http.MethodPost, "/api/v1/products", superToken, map[string]string{"name": "Teslahubs"})
	var p1Env struct {
		Data productResponse `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&p1Env)
	p1 := p1Env.Data

	rec = doJSON(t, r, http.MethodPost, "/api/v1/products", superToken, map[string]string{"name": "ESound"})
	var p2Env struct {
		Data productResponse `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&p2Env)
	p2 := p2Env.Data

	// Create a plain user via the product-user endpoint, requesting the
	// 'admin' role directly (new-account exemption from request/approval).
	rec = doJSON(t, r, http.MethodPost, "/api/v1/products/"+p1.ID+"/users", superToken, map[string]string{
		"name": "Admin User", "username": "adminuser-" + p1.ID, "email": "adminuser-" + p1.ID + "@example.com",
		"password": "password123", "system_role": "admin",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create product user with admin role: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var adminSubEnv struct {
		Data subscriptionResponse `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&adminSubEnv)
	adminUserID := adminSubEnv.Data.UserID

	adminToken := loginAs(t, r, "adminuser-"+p1.ID, "password123")

	// The admin's scoped list only includes p1, not p2.
	rec = doJSON(t, r, http.MethodGet, "/api/v1/products/mine", adminToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list mine: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var mineEnv struct {
		Data []productResponse `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&mineEnv)
	if len(mineEnv.Data) != 1 || mineEnv.Data[0].ID != p1.ID {
		t.Fatalf("expected admin to see only p1, got %+v", mineEnv.Data)
	}

	// Browse list shows both, read-only.
	rec = doJSON(t, r, http.MethodGet, "/api/v1/products/browse", adminToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("browse: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// The admin creates a plain user via p1, then requests admin access
	// for that user on p2 (which the admin does NOT manage) — must fail.
	rec = doJSON(t, r, http.MethodPost, "/api/v1/products/"+p1.ID+"/users", adminToken, map[string]string{
		"name": "Plain User", "username": "plainuser-" + p1.ID, "email": "plainuser-" + p1.ID + "@example.com",
		"password": "password123", "system_role": "user",
	})
	var plainSubEnv struct {
		Data subscriptionResponse `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&plainSubEnv)
	plainUserID := plainSubEnv.Data.UserID

	rec = doJSON(t, r, http.MethodPost, "/api/v1/products/"+p2.ID+"/admin-requests", adminToken, map[string]string{"subject_user_id": plainUserID})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("request admin for unmanaged product: expected 403, got %d: %s", rec.Code, rec.Body.String())
	}

	// Requesting on p1 (which the admin does manage) succeeds.
	rec = doJSON(t, r, http.MethodPost, "/api/v1/products/"+p1.ID+"/admin-requests", adminToken, map[string]string{"subject_user_id": plainUserID})
	if rec.Code != http.StatusCreated {
		t.Fatalf("request admin: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var reqEnv struct {
		Data productAdminRequestResponse `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&reqEnv)

	// The admin cannot decide their own request.
	rec = doJSON(t, r, http.MethodPost, "/api/v1/products/"+p1.ID+"/admin-requests/"+reqEnv.Data.ID+"/decide", adminToken, map[string]bool{"approve": true})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("admin deciding own request: expected 403, got %d: %s", rec.Code, rec.Body.String())
	}

	// Superadmin approves it.
	rec = doJSON(t, r, http.MethodPost, "/api/v1/products/"+p1.ID+"/admin-requests/"+reqEnv.Data.ID+"/decide", superToken, map[string]bool{"approve": true})
	if rec.Code != http.StatusOK {
		t.Fatalf("superadmin decide: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Superadmin can also promote directly, with no request at all.
	rec = doJSON(t, r, http.MethodPost, "/api/v1/products/"+p2.ID+"/admin", superToken, map[string]string{"subject_user_id": adminUserID})
	if rec.Code != http.StatusOK {
		t.Fatalf("promote directly: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}
```

This test needs a `loginAs(t, r, username, password) string` helper — check whether `product_handler_test.go` or `auth_handler_test.go` already has one; if not, add it to `product_handler_test.go`:

```go
func loginAs(t *testing.T, r chi.Router, username, password string) string {
	t.Helper()
	rec := doJSON(t, r, http.MethodPost, "/api/v1/auth/login", "", map[string]string{"identifier": username, "password": password})
	if rec.Code != http.StatusOK {
		t.Fatalf("login as %s: expected 200, got %d: %s", username, rec.Code, rec.Body.String())
	}
	var env struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&env)
	return env.Data.Token
}
```

This also requires `/api/v1/auth/login` to be wired in `testProductRouter` — check the existing `testProductRouter` function; if it doesn't already register that route, add `r.Post("/api/v1/auth/login", authHandler.Login)` alongside its other route registrations, constructing `authHandler := handlers.NewAuthHandler(authService)` (the `authService` variable already exists in `testProductRouter` per Task 5's earlier wiring).

- [ ] **Step 2: Run test to verify it fails**

Run: `cd platform-identity-service && go test ./internal/handlers/... -run TestProductHandler_ScopedListAndAdminRequestFlow -v`
Expected: FAIL to compile — `ListMine`, `ListBrowse`, `RequestAdmin`, `DecideAdminRequest`, `PromoteAdmin` undefined; `createProductUserRequest` has no `system_role` field.

- [ ] **Step 3: Write minimal implementation**

In `platform-identity-service/internal/handlers/product_handler.go`, update `createProductUserRequest` and `CreateUserAndSubscribe`:

```go
type createProductUserRequest struct {
	Name       string `json:"name"`
	Username   string `json:"username"`
	Email      string `json:"email"`
	Password   string `json:"password"`
	SystemRole string `json:"system_role"`
}
```

```go
func (h *ProductHandler) CreateUserAndSubscribe(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createProductUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" || req.Username == "" || req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "name, username, email and password are required")
		return
	}
	systemRole := req.SystemRole
	if systemRole == "" {
		systemRole = models.SystemRoleUser
	}
	if systemRole != models.SystemRoleUser && systemRole != models.SystemRoleAdmin {
		writeError(w, http.StatusBadRequest, "system_role must be 'user' or 'admin'")
		return
	}

	productID := chi.URLParam(r, "id")
	sub, err := h.svc.CreateUserAndSubscribe(r.Context(), caller, productID, service.CreateUserInput{
		Name: req.Name, Username: req.Username, Email: req.Email, Password: req.Password,
	}, systemRole)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			writeError(w, http.StatusForbidden, "not permitted to manage this product")
		case errors.Is(err, repository.ErrProductNotFound):
			writeError(w, http.StatusNotFound, "product not found")
		case errors.Is(err, service.ErrUsernameTaken):
			writeError(w, http.StatusConflict, "username already registered")
		case errors.Is(err, service.ErrEmailTaken):
			writeError(w, http.StatusConflict, "email already registered")
		default:
			writeError(w, http.StatusInternalServerError, "failed to create user")
		}
		return
	}
	writeJSON(w, http.StatusCreated, subscriptionResponse{
		UserID: sub.UserID, ProductID: sub.ProductID, Subscripted: sub.Subscripted, Renewed: sub.Renewed, Notes: sub.Notes,
	})
}
```

Note every existing `writeError(w, http.StatusForbidden, "superadmin role required")` string in this file, for `UpdateProfile`/`AddSubproject`/`RemoveSubproject`/`SetSubscription`, is now inaccurate since `canManage` (Task 1) also allows a subscribed admin — update each of those four messages to `"not permitted to manage this product"` to match the new, broader gate. Find them with `grep -n '"superadmin role required"' internal/handlers/product_handler.go`.

Append the new handler code at the end of the file:

```go
// ListMine godoc
// @Summary      List products the caller may administer
// @Description  Superadmin sees every product; admin sees only products they hold a subscription to.
// @Tags         products
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} productResponse
// @Router       /products/mine [get]
func (h *ProductHandler) ListMine(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	products, err := h.svc.ListForCaller(r.Context(), caller)
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

type productBrowseResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListBrowse godoc
// @Summary      Read-only list of every product's id and name
// @Description  Any authenticated caller — lets an admin with no subscriptions see what products exist.
// @Tags         products
// @Produce      json
// @Security     BearerAuth
// @Success      200 {array} productBrowseResponse
// @Router       /products/browse [get]
func (h *ProductHandler) ListBrowse(w http.ResponseWriter, r *http.Request) {
	products, err := h.svc.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list products")
		return
	}
	response := make([]productBrowseResponse, len(products))
	for i, p := range products {
		response[i] = productBrowseResponse{ID: p.ID, Name: p.Name}
	}
	writeJSON(w, http.StatusOK, response)
}

type requestAdminRequest struct {
	SubjectUserID string `json:"subject_user_id"`
}

type productAdminRequestResponse struct {
	ID                string `json:"id"`
	ProductID         string `json:"product_id"`
	SubjectUserID     string `json:"subject_user_id"`
	RequestedByUserID string `json:"requested_by_user_id"`
	Status            string `json:"status"`
	CreatedAt         string `json:"created_at"`
}

func toProductAdminRequestResponse(req *models.ProductAdminRequest) productAdminRequestResponse {
	return productAdminRequestResponse{
		ID: req.ID, ProductID: req.ProductID, SubjectUserID: req.SubjectUserID,
		RequestedByUserID: req.RequestedByUserID, Status: req.Status,
		CreatedAt: req.CreatedAt.Format(timeFormat),
	}
}

// RequestAdmin godoc
// @Summary      Request that a user become admin for a product
// @Description  Admin (managing this product) or superadmin. Requires superadmin approval to take effect.
// @Tags         products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Product ID"
// @Param        request body requestAdminRequest true "Subject payload"
// @Success      201 {object} productAdminRequestResponse
// @Failure      403 {object} map[string]string
// @Router       /products/{id}/admin-requests [post]
func (h *ProductHandler) RequestAdmin(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req requestAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.SubjectUserID == "" {
		writeError(w, http.StatusBadRequest, "subject_user_id is required")
		return
	}

	productID := chi.URLParam(r, "id")
	created, err := h.svc.RequestProductAdmin(r.Context(), caller, productID, req.SubjectUserID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			writeError(w, http.StatusForbidden, "not permitted to manage this product")
		case errors.Is(err, service.ErrAlreadyHasAccess):
			writeError(w, http.StatusConflict, "subject already has access to this product")
		case errors.Is(err, service.ErrRequestAlreadyPending):
			writeError(w, http.StatusConflict, "a pending request already exists for this user and product")
		default:
			writeError(w, http.StatusInternalServerError, "failed to create request")
		}
		return
	}
	writeJSON(w, http.StatusCreated, toProductAdminRequestResponse(created))
}

// ListAdminRequests godoc
// @Summary      List pending admin requests for a product
// @Tags         products
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Product ID"
// @Success      200 {array} productAdminRequestResponse
// @Router       /products/{id}/admin-requests [get]
func (h *ProductHandler) ListAdminRequests(w http.ResponseWriter, r *http.Request) {
	productID := chi.URLParam(r, "id")
	requests, err := h.svc.ListPendingRequests(r.Context(), productID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list requests")
		return
	}
	response := make([]productAdminRequestResponse, len(requests))
	for i, req := range requests {
		response[i] = toProductAdminRequestResponse(&req)
	}
	writeJSON(w, http.StatusOK, response)
}

type decideAdminRequestRequest struct {
	Approve bool `json:"approve"`
}

// DecideAdminRequest godoc
// @Summary      Approve or reject a pending admin request
// @Description  Superadmin only.
// @Tags         products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Product ID"
// @Param        requestId path string true "Request ID"
// @Param        request body decideAdminRequestRequest true "Decision payload"
// @Success      200 {object} productAdminRequestResponse
// @Failure      403 {object} map[string]string
// @Router       /products/{id}/admin-requests/{requestId}/decide [post]
func (h *ProductHandler) DecideAdminRequest(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req decideAdminRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	requestID := chi.URLParam(r, "requestId")
	decided, err := h.svc.DecideRequest(r.Context(), caller, requestID, req.Approve)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			writeError(w, http.StatusForbidden, "superadmin role required")
		case errors.Is(err, repository.ErrProductAdminRequestNotFound):
			writeError(w, http.StatusNotFound, "request not found")
		default:
			writeError(w, http.StatusInternalServerError, "failed to decide request")
		}
		return
	}
	writeJSON(w, http.StatusOK, toProductAdminRequestResponse(decided))
}

type promoteAdminRequest struct {
	SubjectUserID string `json:"subject_user_id"`
}

// PromoteAdmin godoc
// @Summary      Directly promote a user to admin for a product
// @Description  Superadmin only. No request is created — immediate effect.
// @Tags         products
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Product ID"
// @Param        request body promoteAdminRequest true "Subject payload"
// @Success      200 {object} subscriptionResponse
// @Failure      403 {object} map[string]string
// @Router       /products/{id}/admin [post]
func (h *ProductHandler) PromoteAdmin(w http.ResponseWriter, r *http.Request) {
	caller, ok := middleware.CallerFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req promoteAdminRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.SubjectUserID == "" {
		writeError(w, http.StatusBadRequest, "subject_user_id is required")
		return
	}

	productID := chi.URLParam(r, "id")
	sub, err := h.svc.PromoteDirectly(r.Context(), caller, productID, req.SubjectUserID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrForbidden):
			writeError(w, http.StatusForbidden, "superadmin role required")
		default:
			writeError(w, http.StatusInternalServerError, "failed to promote user")
		}
		return
	}
	writeJSON(w, http.StatusOK, subscriptionResponse{
		UserID: sub.UserID, ProductID: sub.ProductID, Subscripted: sub.Subscripted, Renewed: sub.Renewed, Notes: sub.Notes,
	})
}
```

This references `h.svc.ListForCaller` — add this new method to `ProductService` (in `product_service.go`, alongside `List`):

```go
// ListForCaller returns every product a superadmin may see, or only the
// products an admin holds a subscription to.
func (s *ProductService) ListForCaller(ctx context.Context, caller shopassign.Caller) ([]models.Product, error) {
	if caller.SystemRole == models.SystemRoleSuperadmin {
		return s.productRepo.List(ctx)
	}
	productIDs, err := s.subRepo.ListProductIDsByUser(ctx, caller.UserID)
	if err != nil {
		return nil, err
	}
	all, err := s.productRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	idSet := make(map[string]bool, len(productIDs))
	for _, id := range productIDs {
		idSet[id] = true
	}
	filtered := make([]models.Product, 0, len(productIDs))
	for _, p := range all {
		if idSet[p.ID] {
			filtered = append(filtered, p)
		}
	}
	return filtered, nil
}
```

This needs a new `SubscriptionRepository.ListProductIDsByUser` method — add it to
`platform-identity-service/internal/repository/subscription_repository.go`:

```go
// ListProductIDsByUser returns every product_id userID holds a subscription
// row for, regardless of Subscripted's value — an admin managing a
// product they were subscribed-then-unsubscribed from still manages it in
// this scoping sense (a separate concern from whether their own account
// has "full" product access, which CheckAccess covers).
func (r *SubscriptionRepository) ListProductIDsByUser(ctx context.Context, userID string) ([]string, error) {
	const q = `SELECT product_id FROM user_product_subscriptions WHERE user_id = $1`
	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd platform-identity-service && go test ./internal/handlers/... -run TestProductHandler_ScopedListAndAdminRequestFlow -v`
Expected: PASS. Then run the full handlers suite: `go test ./internal/handlers/... -v` — expect all PASS.

- [ ] **Step 5: Commit**

```bash
git add platform-identity-service/internal/handlers/product_handler.go platform-identity-service/internal/handlers/product_handler_test.go platform-identity-service/internal/service/product_service.go platform-identity-service/internal/repository/subscription_repository.go
git commit -m "feat(platform-identity-service): add scoped listing, browse, and admin-request HTTP routes

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 7: Wire everything in `main.go`

**Files:**
- Modify: `platform-identity-service/cmd/api/main.go`

**Interfaces:**
- Consumes: `repository.NewProductAdminRequestRepository`, `service.NewProductService` (now 6 args, Task 5), `productHandler.ListMine/ListBrowse/RequestAdmin/ListAdminRequests/DecideAdminRequest/PromoteAdmin` (Task 6), `userHandler.ListBasic` (Task 2).
- Produces: fully wired binary.

- [ ] **Step 1: Update the repository/service construction**

In `platform-identity-service/cmd/api/main.go`, find the existing lines (via `grep -n "productRepo\|productService\|subprojectRepo" cmd/api/main.go`) and add the new repository:

```go
reqRepo := repository.NewProductAdminRequestRepository(db)
```

right after the existing `subprojectRepo := repository.NewSubprojectRepository(db)` line.

Update the `NewProductService` call:

```go
productService := service.NewProductService(productRepo, subscriptionRepo, subprojectRepo, reqRepo, userRepo, authService)
```

(`userRepo` and `authService` already exist earlier in this file per Task 5's constructor change — `authService` must be constructed before this line; check with `grep -n "authService :=" cmd/api/main.go` that it already is, per the existing ordering from earlier work this session.)

- [ ] **Step 2: Add the new routes**

Add `r.Get("/products/mine", productHandler.ListMine)` and `r.Get("/products/browse", productHandler.ListBrowse)` inside the authenticated `r.Group` (alongside the existing `r.Post("/products/{id}/profile", ...)` lines) — both need an authenticated caller, so they belong in that group, not the public routes above it.

Add the remaining new routes in the same authenticated group, after the existing `r.Post("/products/{id}/users", productHandler.CreateUserAndSubscribe)` line:

```go
r.Post("/products/{id}/admin-requests", productHandler.RequestAdmin)
r.Get("/products/{id}/admin-requests", productHandler.ListAdminRequests)
r.Post("/products/{id}/admin-requests/{requestId}/decide", productHandler.DecideAdminRequest)
r.Post("/products/{id}/admin", productHandler.PromoteAdmin)
```

- [ ] **Step 3: Verify full build and full test suite**

Run: `cd platform-identity-service && go build ./...`
Expected: builds cleanly.

Run: `cd platform-identity-service && go test ./... -v 2>&1 | tail -60`
Expected: all PASS — this is the first point every package compiles and runs together against the complete set of changes from Tasks 1-6.

- [ ] **Step 4: Commit**

```bash
git add platform-identity-service/cmd/api/main.go
git commit -m "feat(platform-identity-service): wire product-scoped admin and admin-request routes into main

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 8: Frontend — split the panel gate, scope the sidebar, and update the API client

**Files:**
- Create: `platform-identity-admin/src/auth/RequireSuperadmin.tsx`
- Create: `platform-identity-admin/src/auth/RequireAdminOrAbove.tsx`
- Modify: `platform-identity-admin/src/App.tsx`
- Modify: `platform-identity-admin/src/layouts/AppLayout.tsx`
- Modify: `platform-identity-admin/src/api/types.ts`
- Modify: `platform-identity-admin/src/api/products.ts`
- Modify: `platform-identity-admin/src/api/users.ts`
- Delete: `platform-identity-admin/src/auth/RequireSystemAdmin.tsx` (replaced by the two files above)

**Interfaces:**
- Produces: `RequireSuperadmin` (gates Shops-lar, İstifadəçilər), `RequireAdminOrAbove` (gates Products) — both React Router element components using `useAuth()`'s `claims.system_role`. New API functions `listMyProducts()`, `listBrowseProducts()`, `requestProductAdmin(productId, subjectUserId)`, `listAdminRequests(productId)`, `decideAdminRequest(productId, requestId, approve)`, `promoteProductAdmin(productId, subjectUserId)`, `listBasicUsers()`. New types `ProductAdminRequest`, `BasicUser`. Consumed by Task 9 (ProductsPage rewrite).

- [ ] **Step 1: No backend test applies here — verified by `tsc --noEmit` and a manual smoke test at the end of Task 9**

- [ ] **Step 2: Create the two gate components**

Create `platform-identity-admin/src/auth/RequireSuperadmin.tsx`:

```tsx
import { Navigate, Outlet } from 'react-router-dom';
import { useAuth } from './AuthContext';

// Shops-lar and İstifadəçilər are now superadmin-only — an admin, even one
// who manages several products, has no reason to see every shop or every
// user's shop memberships. Replaces the old RequireSystemAdmin for these
// two pages specifically.
export function RequireSuperadmin() {
  const { claims, myShopId } = useAuth();
  const isSuperadmin = claims?.system_role === 'superadmin';
  if (!isSuperadmin) {
    return <Navigate to={myShopId ? `/shops/${myShopId}/members` : '/products'} replace />;
  }
  return <Outlet />;
}
```

Create `platform-identity-admin/src/auth/RequireAdminOrAbove.tsx`:

```tsx
import { Navigate, Outlet } from 'react-router-dom';
import { useAuth } from './AuthContext';

// Products-lar is open to admin and superadmin alike — an admin's view is
// scoped server-side (GET /products/mine) to only the products they hold
// a subscription to, so no further client-side restriction is needed here
// beyond "can this role reach the page at all."
export function RequireAdminOrAbove() {
  const { claims, myShopId } = useAuth();
  const isAdminOrAbove = claims?.system_role === 'admin' || claims?.system_role === 'superadmin';
  if (!isAdminOrAbove) {
    return <Navigate to={myShopId ? `/shops/${myShopId}/members` : '/login'} replace />;
  }
  return <Outlet />;
}
```

Delete `platform-identity-admin/src/auth/RequireSystemAdmin.tsx`.

- [ ] **Step 3: Update `App.tsx`**

In `platform-identity-admin/src/App.tsx`, replace the import and route wiring:

```tsx
import { RequireSuperadmin } from './auth/RequireSuperadmin';
import { RequireAdminOrAbove } from './auth/RequireAdminOrAbove';
```

(remove the old `import { RequireSystemAdmin } from './auth/RequireSystemAdmin';`)

```tsx
<Route path="/shops/:id/members" element={<ShopMembersPage />} />
<Route element={<RequireAdminOrAbove />}>
  <Route path="/products" element={<ProductsPage />} />
</Route>
<Route element={<RequireSuperadmin />}>
  <Route path="/shops" element={<ShopsPage />} />
  <Route path="/users" element={<UsersPage />} />
</Route>
```

Also update `DefaultRedirect` in the same file — it currently sends any `isSystemAdmin` to `/shops`; since a plain `admin` can no longer see Shops, redirect admin-or-above to `/products` instead:

```tsx
function DefaultRedirect() {
  const { claims, myShopId } = useAuth();
  const isAdminOrAbove = claims?.system_role === 'admin' || claims?.system_role === 'superadmin';
  if (!isAdminOrAbove && myShopId) {
    return <Navigate to={`/shops/${myShopId}/members`} replace />;
  }
  return <Navigate to="/products" replace />;
}
```

- [ ] **Step 4: Update `AppLayout.tsx`'s sidebar**

In `platform-identity-admin/src/layouts/AppLayout.tsx`, split the nav items by exact role:

```tsx
const isSuperadmin = claims?.system_role === 'superadmin';
const isAdminOrAbove = claims?.system_role === 'admin' || isSuperadmin;

const items: MenuProps['items'] = isSuperadmin
  ? [
      { key: '/shops', icon: <AppstoreOutlined />, label: 'Shop-lar' },
      { key: '/products', icon: <ShoppingOutlined />, label: 'Product-lar' },
      { key: '/users', icon: <TeamOutlined />, label: 'İstifadəçilər' },
    ]
  : isAdminOrAbove
    ? [{ key: '/products', icon: <ShoppingOutlined />, label: 'Product-lar' }]
    : [{ key: `/shops/${myShopId}/members`, icon: <TeamOutlined />, label: 'Mənim mağazam' }];
```

Replace the file's existing `const isSystemAdmin = ...` line with the two lines above (delete the old one).

- [ ] **Step 5: Update `types.ts`**

In `platform-identity-admin/src/api/types.ts`, add after `Subscription`:

```ts
export interface ProductAdminRequest {
  id: string;
  product_id: string;
  subject_user_id: string;
  requested_by_user_id: string;
  status: string;
  created_at: string;
}

export interface ProductBrowse {
  id: string;
  name: string;
}

export interface BasicUser {
  id: string;
  name: string;
  username: string;
  email: string;
  system_role: string;
}
```

- [ ] **Step 6: Update `products.ts`**

In `platform-identity-admin/src/api/products.ts`, update the import and add new functions:

```ts
import type { Product, Subproject, Subscription, ProductAdminRequest, ProductBrowse } from './types';
```

```ts
export async function listMyProducts(): Promise<Product[]> {
  const response = await platformIdentityApi.get<Product[]>('/products/mine');
  return response.data;
}

export async function listBrowseProducts(): Promise<ProductBrowse[]> {
  const response = await platformIdentityApi.get<ProductBrowse[]>('/products/browse');
  return response.data;
}

export async function requestProductAdmin(productId: string, subjectUserId: string): Promise<ProductAdminRequest> {
  const response = await platformIdentityApi.post<ProductAdminRequest>(`/products/${productId}/admin-requests`, {
    subject_user_id: subjectUserId,
  });
  return response.data;
}

export async function listAdminRequests(productId: string): Promise<ProductAdminRequest[]> {
  const response = await platformIdentityApi.get<ProductAdminRequest[]>(`/products/${productId}/admin-requests`);
  return response.data;
}

export async function decideAdminRequest(productId: string, requestId: string, approve: boolean): Promise<ProductAdminRequest> {
  const response = await platformIdentityApi.post<ProductAdminRequest>(
    `/products/${productId}/admin-requests/${requestId}/decide`,
    { approve }
  );
  return response.data;
}

export async function promoteProductAdmin(productId: string, subjectUserId: string): Promise<Subscription> {
  const response = await platformIdentityApi.post<Subscription>(`/products/${productId}/admin`, {
    subject_user_id: subjectUserId,
  });
  return response.data;
}
```

Also update `createProductUser` to accept a system role:

```ts
export async function createProductUser(
  productId: string,
  name: string,
  username: string,
  email: string,
  password: string,
  systemRole: string
): Promise<{ user_id: string; product_id: string; subscripted: boolean; renewed: boolean }> {
  const response = await platformIdentityApi.post(`/products/${productId}/users`, {
    name,
    username,
    email,
    password,
    system_role: systemRole,
  });
  return response.data;
}
```

- [ ] **Step 7: Add `listBasicUsers` to `users.ts`**

In `platform-identity-admin/src/api/users.ts`, add:

```ts
import type { BasicUser, Member, User } from './types';
```

(update the existing `import type { Member, User } from './types';` line to include `BasicUser`)

```ts
export async function listBasicUsers(): Promise<BasicUser[]> {
  const response = await platformIdentityApi.get<BasicUser[]>('/users/basic');
  return response.data;
}
```

- [ ] **Step 8: Type-check**

Run: `cd platform-identity-admin && npx tsc --noEmit`
Expected: no errors. `ProductsPage.tsx` will still reference the old `listProducts`/`createProductUser` (4-arg) signatures at this point — Task 9 fixes those; if `tsc` reports errors in `ProductsPage.tsx` specifically (and only there), that's expected and resolved in the next task, not a regression to fix here. Any error in `App.tsx`, `AppLayout.tsx`, `types.ts`, `products.ts`, or `users.ts` must be fixed now.

- [ ] **Step 9: Commit**

```bash
git add platform-identity-admin/src/auth/RequireSuperadmin.tsx platform-identity-admin/src/auth/RequireAdminOrAbove.tsx platform-identity-admin/src/App.tsx platform-identity-admin/src/layouts/AppLayout.tsx platform-identity-admin/src/api/types.ts platform-identity-admin/src/api/products.ts platform-identity-admin/src/api/users.ts
git rm platform-identity-admin/src/auth/RequireSystemAdmin.tsx
git commit -m "feat(platform-identity-admin): split panel gate into superadmin-only and admin-or-above

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

### Task 9: Frontend — `ProductsPage.tsx` scoped data, browse list, and admin-request UI

**Files:**
- Modify: `platform-identity-admin/src/pages/ProductsPage.tsx`

**Interfaces:**
- Consumes: `listMyProducts`, `listBrowseProducts`, `requestProductAdmin`, `listAdminRequests`, `decideAdminRequest`, `promoteProductAdmin`, `createProductUser` (now 6-arg) (Task 8); `listBasicUsers` (Task 8); `useAuth()` for `claims.system_role` (existing `AuthContext`).

- [ ] **Step 1: Read the current file's exact state before editing**

Run: `cat platform-identity-admin/src/pages/ProductsPage.tsx` — this plan was written against the version that has Profil, Subscription, and Yeni-istifadəçi modals already in place (from earlier work this session). Confirm the file still matches that shape before applying the edits below; if the create-user mutation already calls `createProductUser` with 5 args (no system role), that confirms the expected starting point.

- [ ] **Step 2: Update imports and add the system-role-aware data source**

Replace the top of `platform-identity-admin/src/pages/ProductsPage.tsx`:

```tsx
import { useState } from 'react';
import { Button, Form, Input, List, Modal, Select, Space, Switch, Table, Tag, message } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import type { BasicUser, Product, ProductAdminRequest, ProductBrowse, Subproject } from '../api/types';
import {
  addSubproject,
  createProduct,
  createProductUser,
  decideAdminRequest,
  listAdminRequests,
  listBrowseProducts,
  listMyProducts,
  listSubprojects,
  promoteProductAdmin,
  removeSubproject,
  requestProductAdmin,
  setSubscription,
  updateProductProfile,
} from '../api/products';
import { listBasicUsers } from '../api/users';
import { useAuth } from '../auth/AuthContext';
import { useQueryErrorToast } from '../hooks/useQueryErrorToast';
```

(remove the old `import { listAllUsers } from '../api/users';` — replaced by `listBasicUsers`, which any admin-or-above may call, unlike `listAllUsers`)

Inside `ProductsPage`, replace the products query and add the request-related state:

```tsx
export function ProductsPage() {
  const queryClient = useQueryClient();
  const { claims } = useAuth();
  const isSuperadmin = claims?.system_role === 'superadmin';
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [subModalProduct, setSubModalProduct] = useState<Product | null>(null);
  const [subUserId, setSubUserId] = useState<string | null>(null);
  const [subscripted, setSubscripted] = useState(false);
  const [renewed, setRenewed] = useState(false);
  const [subNotes, setSubNotes] = useState('');
  const [profileModalProduct, setProfileModalProduct] = useState<Product | null>(null);
  const [userModalProduct, setUserModalProduct] = useState<Product | null>(null);
  const [requestsModalProduct, setRequestsModalProduct] = useState<Product | null>(null);
  const [promoteModalProduct, setPromoteModalProduct] = useState<Product | null>(null);
  const [promoteUserId, setPromoteUserId] = useState<string | null>(null);
  const [form] = Form.useForm<ProductFormValues>();
  const [profileForm] = Form.useForm<{ description: string; techStack: string }>();
  const [subprojectForm] = Form.useForm<SubprojectFormValues>();
  const [productUserForm] = Form.useForm<ProductUserFormValues>();

  const { data: products, isLoading, isError, error } = useQuery({ queryKey: ['my-products'], queryFn: () => listMyProducts() });
  useQueryErrorToast(isError, error);

  const { data: browseProducts } = useQuery({ queryKey: ['browse-products'], queryFn: () => listBrowseProducts() });

  const { data: basicUsers } = useQuery({ queryKey: ['basic-users'], queryFn: () => listBasicUsers(), enabled: !!subModalProduct || !!promoteModalProduct });

  const { data: subprojects } = useQuery({
    queryKey: ['product-subprojects', profileModalProduct?.id],
    queryFn: () => listSubprojects(profileModalProduct!.id),
    enabled: !!profileModalProduct,
  });

  const { data: adminRequests } = useQuery({
    queryKey: ['admin-requests', requestsModalProduct?.id],
    queryFn: () => listAdminRequests(requestsModalProduct!.id),
    enabled: !!requestsModalProduct,
  });
```

`ProductFormValues`/`SubprojectFormValues`/`ProductUserFormValues` interfaces stay as they are in the current file — only add a new one:

```tsx
interface ProductUserFormValues {
  name: string;
  username: string;
  email: string;
  password: string;
  systemRole: string;
}
```

(this replaces the existing `ProductUserFormValues` interface, which lacked `systemRole` — find and replace the whole interface block, not just add to it)

- [ ] **Step 3: Update the create-user mutation and its `Select`**

Replace the existing `createProductUserMutation`:

```tsx
  const createProductUserMutation = useMutation({
    mutationFn: (values: ProductUserFormValues) =>
      createProductUser(userModalProduct!.id, values.name, values.username, values.email, values.password, values.systemRole),
    onSuccess: () => {
      message.success('İstifadəçi yaradıldı və məhsula abunə edildi');
      setUserModalProduct(null);
      productUserForm.resetFields();
    },
    onError: (err) => message.error(err instanceof Error ? err.message : 'Xəta baş verdi'),
  });
```

In the "Yeni istifadəçi" modal's `<Form>`, add a role select before the closing `</Form>`:

```tsx
          <Form.Item name="systemRole" label="Sistem rolu" initialValue="user" rules={[{ required: true }]}>
            <Select
              options={[
                { label: 'user', value: 'user' },
                { label: 'admin', value: 'admin' },
              ]}
            />
          </Form.Item>
```

- [ ] **Step 4: Add the request/promote/decide mutations**

Add alongside the existing mutations:

```tsx
  const requestAdminMutation = useMutation({
    mutationFn: (subjectUserId: string) => requestProductAdmin(promoteModalProduct!.id, subjectUserId),
    onSuccess: () => {
      message.success('Sorğu göndərildi, superadmin təsdiqini gözləyir');
      setPromoteModalProduct(null);
      setPromoteUserId(null);
    },
    onError: (err) => message.error(err instanceof Error ? err.message : 'Xəta baş verdi'),
  });

  const promoteDirectlyMutation = useMutation({
    mutationFn: (subjectUserId: string) => promoteProductAdmin(promoteModalProduct!.id, subjectUserId),
    onSuccess: () => {
      message.success('İstifadəçi admin təyin olundu');
      setPromoteModalProduct(null);
      setPromoteUserId(null);
    },
    onError: (err) => message.error(err instanceof Error ? err.message : 'Xəta baş verdi'),
  });

  const decideRequestMutation = useMutation({
    mutationFn: ({ requestId, approve }: { requestId: string; approve: boolean }) =>
      decideAdminRequest(requestsModalProduct!.id, requestId, approve),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin-requests', requestsModalProduct?.id] });
    },
    onError: (err) => message.error(err instanceof Error ? err.message : 'Xəta baş verdi'),
  });
```

- [ ] **Step 5: Add the two new buttons to the row actions**

Replace the `columns` definition's `render` for `Əməliyyat`:

```tsx
      render: (_: unknown, record: Product) => (
        <Space wrap>
          <Button size="small" onClick={() => openProfileModal(record)}>
            Profil
          </Button>
          <Button size="small" onClick={() => setSubModalProduct(record)}>
            Subscription idarə et
          </Button>
          <Button size="small" onClick={() => setUserModalProduct(record)}>
            Yeni istifadəçi
          </Button>
          <Button size="small" onClick={() => setRequestsModalProduct(record)}>
            Admin sorğuları
          </Button>
          <Button size="small" onClick={() => setPromoteModalProduct(record)}>
            Admin təyin et
          </Button>
        </Space>
      ),
```

- [ ] **Step 6: Add the two new modals**

Add before the closing `</>` of the component's returned JSX:

```tsx
      <Modal
        title={requestsModalProduct ? `${requestsModalProduct.name} — Admin sorğuları` : ''}
        open={!!requestsModalProduct}
        onCancel={() => setRequestsModalProduct(null)}
        footer={null}
      >
        <List
          size="small"
          dataSource={adminRequests ?? []}
          locale={{ emptyText: 'Gözləyən sorğu yoxdur' }}
          renderItem={(req: ProductAdminRequest) => (
            <List.Item
              actions={
                isSuperadmin
                  ? [
                      <Button key="approve" size="small" type="primary" onClick={() => decideRequestMutation.mutate({ requestId: req.id, approve: true })}>
                        Approve
                      </Button>,
                      <Button key="reject" size="small" danger onClick={() => decideRequestMutation.mutate({ requestId: req.id, approve: false })}>
                        Reject
                      </Button>,
                    ]
                  : []
              }
            >
              <List.Item.Meta title={req.subject_user_id} description={`Status: ${req.status}, göndərən: ${req.requested_by_user_id}`} />
            </List.Item>
          )}
        />
      </Modal>
      <Modal
        title={promoteModalProduct ? `${promoteModalProduct.name} — Admin təyin et` : ''}
        open={!!promoteModalProduct}
        onCancel={() => {
          setPromoteModalProduct(null);
          setPromoteUserId(null);
        }}
        onOk={() => (isSuperadmin ? promoteDirectlyMutation.mutate(promoteUserId!) : requestAdminMutation.mutate(promoteUserId!))}
        okText={isSuperadmin ? 'Birbaşa təyin et' : 'Sorğu göndər'}
        confirmLoading={isSuperadmin ? promoteDirectlyMutation.isPending : requestAdminMutation.isPending}
        okButtonProps={{ disabled: !promoteUserId }}
      >
        <Select
          style={{ width: '100%' }}
          placeholder="İstifadəçi seç"
          value={promoteUserId ?? undefined}
          onChange={setPromoteUserId}
          options={basicUsers?.map((u: BasicUser) => ({ label: `${u.name} (${u.username}) — ${u.system_role}`, value: u.id }))}
        />
      </Modal>
```

- [ ] **Step 7: Add the read-only browse section for an admin with zero products**

Add right after the `<Table ... />` line, before the first `<Modal>`:

```tsx
      {!isSuperadmin && (products?.length ?? 0) === 0 && (
        <div style={{ marginTop: 24 }}>
          <h4>Bütün productlar</h4>
          <List
            size="small"
            dataSource={browseProducts ?? []}
            renderItem={(p: ProductBrowse) => (
              <List.Item
                actions={[
                  <Button
                    key="request"
                    size="small"
                    onClick={() => {
                      setPromoteModalProduct(p as Product);
                      setPromoteUserId(claims?.user_id ?? null);
                    }}
                  >
                    Sorğu göndər
                  </Button>,
                ]}
              >
                <List.Item.Meta title={p.name} />
              </List.Item>
            )}
          />
        </div>
      )}
```

Note: this casts `p as Product` to reuse `promoteModalProduct`'s existing `Product`-typed state, since `ProductBrowse` only has `id`/`name` but `promoteModalProduct` is only ever read for `.id` and `.name` in the modal above it — this is a deliberate, narrow cast, not a type-safety gap, since nothing else on `promoteModalProduct` is read in this flow.

- [ ] **Step 8: Type-check and build**

Run: `cd platform-identity-admin && npx tsc --noEmit && npm run build`
Expected: no errors, build succeeds.

- [ ] **Step 9: Manual smoke test**

Restart the backend (`kill` the process on port 8095 found via `lsof -iTCP:8095 -sTCP:LISTEN -n -P`, then `cd platform-identity-service && nohup go run ./cmd/api > /tmp/pis-backend.log 2>&1 &`) so it picks up every backend change from Tasks 1-7. Then in the browser (frontend dev server, already running on 5173 per this session's earlier work):

1. Log in as `superadmin`. Confirm Shops-lar, Product-lar, İstifadəçilər all still visible (unchanged for superadmin).
2. On Products, click "Yeni istifadəçi" on Teslahubs, create a new account with role `admin`. Confirm success.
3. Log out, log in as that new admin account. Confirm the sidebar shows only "Product-lar" (no Shop-lar/İstifadəçilər).
4. Confirm the Products table shows only Teslahubs (the product this admin was created under), not the other products.
5. Click "Admin təyin et" on Teslahubs — confirm the button/OK-text reads "Sorğu göndər" (not "Birbaşa təyin et").
6. Create a second, plain `user` account via "Yeni istifadəçi" (role `user`) on Teslahubs. Then use "Admin təyin et" to select that user and send a request.
7. Log back in as `superadmin`, open Teslahubs's "Admin sorğuları" — confirm the pending request appears with Approve/Reject buttons, approve it.
8. Confirm (via a fresh `GET /users` or the İstifadəçilər page) that the approved user's system_role is now `admin` and they have a Teslahubs subscription.
9. Log in as that just-approved user — confirm they can now see and manage Teslahubs.

- [ ] **Step 10: Commit**

```bash
git add platform-identity-admin/src/pages/ProductsPage.tsx
git commit -m "feat(platform-identity-admin): add product-scoped visibility and admin-request UI to Products page

Co-Authored-By: Claude Sonnet 5 <noreply@anthropic.com>"
```

---

## Self-Review Notes

- **Spec coverage:** system_role gate unchanged (Task 8's `RequireAdminOrAbove`/`RequireSuperadmin`); admin scope narrows to Products only (Task 8/9); admin sees only subscribed products (Task 6/7 `GET /products/mine`, Task 9); admin with zero subscriptions still logs in and sees a browse list (Task 9 Step 7); only superadmin promotes directly or decides a request (Task 5's `canManage`/`DecideRequest`/`PromoteDirectly` role checks); admin can only file a request, never promote directly (Task 5/6); request targets only non-subscribed subjects, no duplicate pending requests (Task 5's `ErrAlreadyHasAccess`/`ErrRequestAlreadyPending`); approval queue lives on the Products page per-product (Task 9's "Admin sorğuları" modal); approval sets system_role=admin + creates subscription (Task 5's `promoteToAdminAndSubscribe`); rejection leaves state untouched and doesn't block retry (Task 5's `DecideRequest` reject path plus `HasPending` only checking `status='pending'`); existing Subscription/Yeni-istifadəçi actions unaffected in shape, only their gate loosened (Task 1, Task 6); new-user system-role choice exempt from request/approval (Task 5/6/9). Every spec bullet has a corresponding task.
- **Placeholder scan:** no TBD/TODO; every code block is complete, runnable Go/TypeScript matching the surrounding file's real imports and conventions as read from the actual current files in this worktree.
- **Type consistency:** `CreateUserAndSubscribe`'s signature change (4 args → 5, adding `systemRole string`) is introduced once in Task 5 and consistently used in Task 5's own tests, Task 6's handler, and Task 9's frontend mutation — no mismatched call sites. `NewProductService`'s signature change (4 args → 6, adding `reqRepo`, `userRepo`) is introduced in Task 5 and wired in Task 7; Task 5's own test helper `newTestProductService` (pre-existing in the file from earlier work this session) must be updated in Task 5 Step 3 to match — this update is implied by "update `NewProductService`" but call out explicitly: after Task 5's struct/constructor change, `newTestProductService` in `product_service_test.go` also needs its `NewProductService(...)` call updated to pass `reqRepo` and `userRepo` (both constructed the same way `product_handler_test.go`'s `testProductRouter` already does) — add this as an explicit sub-step when executing Task 5 Step 3 if the existing helper isn't already caught by the "Run test to verify it passes" full-package run failing to compile.
