# Single-Session Token Revocation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let platform-identity-service log out one specific issued JWT (one browser session) without affecting any of that same user's other active sessions, and without changing anything about the token's existing signature/expiry checks.

**Architecture:** Every JWT gets a random `jti` (JWT ID) claim at mint time. A new Postgres table (`token_revocations`) records which `jti`s have been logged out. `RequireAuth` gains a second DB check — after verifying the signature/expiry, it looks up the token's `jti` in that table and rejects the request if found. A new `POST /auth/logout` endpoint (itself behind `RequireAuth`, so it only accepts a currently-valid token) writes that token's own `jti` into the table.

**Tech Stack:** Go, `database/sql` + `github.com/jackc/pgx/v5/stdlib` driver, `github.com/golang-jwt/jwt/v5`, `github.com/google/uuid`, `github.com/go-chi/chi/v5`.

## Global Constraints

- This project has no numbered migration files — every table is created inside the single idempotent `database.Migrate()` function in `internal/database/postgres.go`, using `CREATE TABLE IF NOT EXISTS` / `ADD COLUMN IF NOT EXISTS`. The new table goes there, not in a new file.
- Integration tests read the `TEST_DSN` env var, falling back to `host=localhost port=5433 user=postgres password=1 dbname=platform_identity sslmode=disable` (see `internal/repository/repository_test.go`'s `testDB(t)` and `internal/handlers/auth_handler_test.go`'s `testRouter(t)` — both already exist and will be reused, not duplicated).
- Every handler response goes through `internal/handlers/response.go`'s `writeJSON`/`writeError` helpers (wraps in `{"success": bool, "data": ...}` / `{"success": false, "error": {"code", "message"}}`). Do not invent a different response shape.
- `RequireAuth`'s signature is changing (gains a third parameter). It has exactly one production call site (`cmd/api/main.go`) and is constructed fresh inside `testRouter(t)` in `internal/handlers/auth_handler_test.go` for every handler test — both must be updated in the same task that changes the signature, or the package will not compile.
- `JWTManager.Verify()` itself stays signature-and-expiry-only, exactly as it is today. The blocklist check is a separate, independent lookup inside `RequireAuth`, never folded into `Verify()`.
- No cleanup/sweep job for old `token_revocations` rows — explicitly out of scope (see spec's "Out of scope").

---

### Task 1: `jti` claim on `Claims`, minted by `Generate`

**Files:**
- Modify: `internal/auth/jwt.go`
- Test: `internal/auth/jwt_test.go`

**Interfaces:**
- Consumes: nothing new — `github.com/google/uuid` is already a dependency (`go.mod` already has `github.com/google/uuid v1.6.0`; `cmd/api/main.go` already imports and calls `uuid.NewString()`).
- Produces: `Claims.JTI string` (populated from the standard `jti` registered claim, `jwt.RegisteredClaims.ID`). Every later task that needs the current request's `jti` reads `claims.JTI` where `claims` is a `*auth.Claims`.

- [ ] **Step 1: Write the failing test**

Add to `internal/auth/jwt_test.go`:

```go
func TestJWTManager_Generate_SetsUniqueJTI(t *testing.T) {
	m := NewJWTManager("test-secret", 60)

	token1, _, err := m.Generate("user-123", "admin")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	claims1, err := m.Verify(token1)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if claims1.JTI == "" {
		t.Fatal("expected a non-empty JTI claim")
	}

	token2, _, err := m.Generate("user-123", "admin")
	if err != nil {
		t.Fatalf("second Generate failed: %v", err)
	}
	claims2, err := m.Verify(token2)
	if err != nil {
		t.Fatalf("second Verify failed: %v", err)
	}
	if claims2.JTI == "" {
		t.Fatal("expected a non-empty JTI claim on the second token")
	}
	if claims1.JTI == claims2.JTI {
		t.Fatal("expected two separate Generate calls to mint two different JTIs")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd internal/auth && go test ./... -run TestJWTManager_Generate_SetsUniqueJTI -v`
Expected: FAIL — `claims1.JTI` does not exist yet (compile error: `Claims` has no field `JTI`).

- [ ] **Step 3: Write minimal implementation**

In `internal/auth/jwt.go`:

1. Add the import:

```go
import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)
```

2. Add `JTI` to `Claims`:

```go
type Claims struct {
	UserID     string `json:"user_id"`
	SystemRole string `json:"system_role"`
	JTI        string `json:"jti"`
	jwt.RegisteredClaims
}
```

3. In `Generate`, set both the registered claim and the mirrored field (the mirrored `JTI` field exists purely so callers read `claims.JTI` directly instead of reaching into the embedded `RegisteredClaims.ID` — same reasoning as `UserID`/`SystemRole` already being pulled out as their own top-level fields instead of relying on `RegisteredClaims.Subject`):

```go
func (m *JWTManager) Generate(userID, systemRole string) (string, time.Time, error) {
	expiresAt := time.Now().Add(m.ttl)
	jti := uuid.NewString()
	claims := Claims{
		UserID:     userID,
		SystemRole: systemRole,
		JTI:        jti,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiresAt, nil
}
```

`Verify` needs no change — `jwt.ParseWithClaims` already populates every field of `Claims` (including the new `JTI` and the embedded `RegisteredClaims.ID`) from the token's payload automatically; it is only `Generate` that needed to actively set the new field.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd internal/auth && go test ./... -v`
Expected: PASS — both the new test and the two pre-existing tests (`TestJWTManager_GenerateAndVerify`, `TestJWTManager_Verify_RejectsWrongSecret`) still pass.

- [ ] **Step 5: Commit**

```bash
git add internal/auth/jwt.go internal/auth/jwt_test.go
git commit -m "feat: mint a unique jti claim on every generated JWT"
```

---

### Task 2: `token_revocations` table + `TokenRevocationRepository`

**Files:**
- Modify: `internal/database/postgres.go`
- Create: `internal/repository/token_revocation_repository.go`
- Test: `internal/repository/repository_test.go` (add to the existing file — it already has a shared `testDB(t)` helper this task reuses, not a new one)

**Interfaces:**
- Consumes: `database/sql`'s `*sql.DB` (same as every other repository in this package, e.g. `NewSettingsRepository(db *sql.DB)`).
- Produces: `NewTokenRevocationRepository(db *sql.DB) *TokenRevocationRepository`, `(*TokenRevocationRepository).Revoke(ctx context.Context, jti string, expiresAt time.Time) error`, `(*TokenRevocationRepository).IsRevoked(ctx context.Context, jti string) (bool, error)`. Task 4 (middleware) calls `IsRevoked`; Task 5 (logout handler) calls `Revoke`.

- [ ] **Step 1: Write the failing test**

Add to `internal/repository/repository_test.go` (reuses the file's existing `testDB(t)` helper — do not write a new DB-connection helper):

```go
func TestTokenRevocationRepository_RevokeAndIsRevoked(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	repo := NewTokenRevocationRepository(db)

	jti := uuid.NewString()

	revoked, err := repo.IsRevoked(context.Background(), jti)
	if err != nil {
		t.Fatalf("IsRevoked (before revoke) failed: %v", err)
	}
	if revoked {
		t.Fatal("expected a never-revoked jti to report false")
	}

	if err := repo.Revoke(context.Background(), jti, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("Revoke failed: %v", err)
	}

	revoked, err = repo.IsRevoked(context.Background(), jti)
	if err != nil {
		t.Fatalf("IsRevoked (after revoke) failed: %v", err)
	}
	if !revoked {
		t.Fatal("expected a revoked jti to report true")
	}

	// Revoking the same jti twice must not error (e.g. a retried logout call).
	if err := repo.Revoke(context.Background(), jti, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("second Revoke on the same jti failed: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd internal/repository && go test ./... -run TestTokenRevocationRepository_RevokeAndIsRevoked -v`
Expected: FAIL — compile error, `NewTokenRevocationRepository` does not exist yet.

- [ ] **Step 3: Write minimal implementation**

First, add the table to `internal/database/postgres.go`'s `Migrate` function. Insert this block right before the final `settings` block (after the `user_product_subscriptions` block, before the `-- Generic key/value store...` comment):

```go
		CREATE TABLE IF NOT EXISTS token_revocations (
			jti        TEXT PRIMARY KEY,
			revoked_at TIMESTAMPTZ NOT NULL DEFAULT now(),
			expires_at TIMESTAMPTZ NOT NULL
		);

```

Then create `internal/repository/token_revocation_repository.go`:

```go
package repository

import (
	"context"
	"database/sql"
	"time"
)

type TokenRevocationRepository struct {
	db *sql.DB
}

func NewTokenRevocationRepository(db *sql.DB) *TokenRevocationRepository {
	return &TokenRevocationRepository{db: db}
}

// Revoke marks jti as logged out. expiresAt should be the token's own exp
// claim — stored for a possible future cleanup job, not required for
// correctness today. ON CONFLICT DO NOTHING makes a repeated call for the
// same jti (e.g. a retried logout request) a harmless no-op.
func (r *TokenRevocationRepository) Revoke(ctx context.Context, jti string, expiresAt time.Time) error {
	const q = `
		INSERT INTO token_revocations (jti, expires_at) VALUES ($1, $2)
		ON CONFLICT (jti) DO NOTHING
	`
	_, err := r.db.ExecContext(ctx, q, jti, expiresAt)
	return err
}

// IsRevoked reports whether jti has been logged out.
func (r *TokenRevocationRepository) IsRevoked(ctx context.Context, jti string) (bool, error) {
	const q = `SELECT 1 FROM token_revocations WHERE jti = $1`
	var one int
	err := r.db.QueryRowContext(ctx, q, jti).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd internal/repository && go test ./... -v`
Expected: PASS — the new test passes, and every pre-existing test in this package still passes (the new `CREATE TABLE IF NOT EXISTS` is additive and does not touch any existing table).

- [ ] **Step 5: Commit**

```bash
git add internal/database/postgres.go internal/repository/token_revocation_repository.go internal/repository/repository_test.go
git commit -m "feat: add token_revocations table and TokenRevocationRepository"
```

---

### Task 3: Stash `*auth.Claims` in request context, add `ClaimsFromContext`

**Files:**
- Modify: `internal/middleware/auth.go`
- Test: none yet standalone — this task's behavior is exercised indirectly by Task 5's handler test, since there is no existing direct unit test of context stashing alone in this codebase (`CallerFromContext`, the existing analog, has none either). Verified here only via `go build`/`go vet`, with real test coverage arriving in Task 4 and Task 5.

**Interfaces:**
- Consumes: `auth.Claims` (from Task 1, already has `.JTI` and the embedded `RegisteredClaims.ExpiresAt`).
- Produces: `middleware.ClaimsFromContext(ctx context.Context) (*auth.Claims, bool)`. Task 5's `Logout` handler calls this to read `claims.JTI` and `claims.ExpiresAt.Time`.

- [ ] **Step 1: Add the new context key and stash the claims**

This is a small, mechanical addition with no independently-testable behavior of its own (storing a value in a `context.Context` and reading it back has no failure mode worth a dedicated test — Go's `context` package already guarantees this works). Make the change directly, then verify via the build step below. In `internal/middleware/auth.go`, add a second context key next to the existing one:

```go
const callerKey contextKey = "caller"
const claimsKey contextKey = "claims"
```

Add the new accessor function next to the existing `CallerFromContext`:

```go
func ClaimsFromContext(ctx context.Context) (*auth.Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(*auth.Claims)
	return claims, ok
}
```

(The actual `ctx = context.WithValue(ctx, claimsKey, claims)` call happens inside `RequireAuth` itself, written in Task 4 alongside the blocklist check — both land in the same function body in the same task, since splitting "stash claims" from "add the blocklist check that produces the need to stash them" would leave this task's own `RequireAuth` body half-rewritten and not compiling cleanly on its own.)

- [ ] **Step 2: Build to confirm it compiles**

Run: `go build ./...`
Expected: succeeds (this task only adds new, unused-so-far code — `claimsKey`'s "declared and not used" concern does not apply to package-level `const`s, and `ClaimsFromContext` is an exported function so Go does not flag it as unused either).

- [ ] **Step 3: Commit**

```bash
git add internal/middleware/auth.go
git commit -m "feat: add ClaimsFromContext accessor to middleware package"
```

---

### Task 4: Blocklist check in `RequireAuth`

**Files:**
- Modify: `internal/middleware/auth.go`
- Modify: `cmd/api/main.go` (update the one production call site)
- Modify: `internal/handlers/auth_handler_test.go` (update `testRouter`'s call site so the package keeps compiling)
- Test: `internal/middleware/auth_test.go` (new file)

**Interfaces:**
- Consumes: `repository.TokenRevocationRepository` (Task 2) via its `IsRevoked` method; `auth.Claims.JTI` (Task 1); `middleware.claimsKey`/`ClaimsFromContext` (Task 3).
- Produces: `RequireAuth(jwt *auth.JWTManager, userRepo *repository.UserRepository, revocationRepo *repository.TokenRevocationRepository) func(http.Handler) http.Handler` — the new 3-argument signature every caller must use from now on. Task 5's logout route is mounted inside this same middleware.

- [ ] **Step 1: Write the failing test**

Create `internal/middleware/auth_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd internal/middleware && go test ./... -run TestRequireAuth_RejectsRevokedToken -v`
Expected: FAIL — compile error, `RequireAuth` only takes 2 arguments today, not 3.

- [ ] **Step 3: Write minimal implementation**

In `internal/middleware/auth.go`, replace the `RequireAuth` function body:

```go
func RequireAuth(jwt *auth.JWTManager, userRepo *repository.UserRepository, revocationRepo *repository.TokenRevocationRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			token := strings.TrimPrefix(header, "Bearer ")
			if header == "" || token == header {
				writeAuthError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid authorization header")
				return
			}

			claims, err := jwt.Verify(token)
			if err != nil {
				writeAuthError(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
				return
			}

			revoked, err := revocationRepo.IsRevoked(r.Context(), claims.JTI)
			if err != nil {
				writeAuthError(w, http.StatusInternalServerError, "internal_error", "failed to verify token")
				return
			}
			if revoked {
				writeAuthError(w, http.StatusUnauthorized, "unauthorized", "token has been revoked")
				return
			}

			user, err := userRepo.GetByID(r.Context(), claims.UserID)
			if err != nil {
				writeAuthError(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
				return
			}
			if user.Status != models.UserStatusActive {
				writeAuthError(w, http.StatusUnauthorized, "unauthorized", "account is deactivated")
				return
			}

			caller := shopassign.Caller{UserID: claims.UserID, SystemRole: claims.SystemRole}
			ctx := context.WithValue(r.Context(), callerKey, caller)
			ctx = context.WithValue(ctx, claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
```

Now fix the two call sites so the package set still compiles:

In `cmd/api/main.go`, change:

```go
r.Use(appmiddleware.RequireAuth(jwtManager, userRepo))
```

to:

```go
r.Use(appmiddleware.RequireAuth(jwtManager, userRepo, tokenRevocationRepo))
```

— this requires `tokenRevocationRepo` to exist as a local variable in `main()`. Add it right next to the other repository constructions (near `settingsRepo := repository.NewSettingsRepository(db)`):

```go
tokenRevocationRepo := repository.NewTokenRevocationRepository(db)
```

In `internal/handlers/auth_handler_test.go`'s `testRouter(t)` helper, change:

```go
r.Use(appmiddleware.RequireAuth(jwtManager, userRepo))
```

to:

```go
tokenRevocationRepo := repository.NewTokenRevocationRepository(db)
r.Use(appmiddleware.RequireAuth(jwtManager, userRepo, tokenRevocationRepo))
```

(placed right before the `r.Group(...)` block that uses it, next to the existing `userRepo`/`shopRepo`/`membershipRepo` declarations at the top of `testRouter`).

- [ ] **Step 4: Run test to verify it passes**

Run: `go build ./... && go test ./... -v`
Expected: PASS across every package — `internal/middleware`'s new test, and every pre-existing test in `internal/handlers` and elsewhere that depends on `RequireAuth` compiling and behaving as before for a non-revoked token (in particular `TestRequireAuth_RejectsTokenAfterUserDeactivated` in `internal/handlers/auth_handler_test.go`, which must keep passing unchanged).

- [ ] **Step 5: Commit**

```bash
git add internal/middleware/auth.go internal/middleware/auth_test.go cmd/api/main.go internal/handlers/auth_handler_test.go
git commit -m "feat: reject requests whose token has been revoked"
```

---

### Task 5: `POST /auth/logout` endpoint

**Files:**
- Modify: `internal/handlers/auth_handler.go`
- Modify: `cmd/api/main.go` (wire the new handler + route)
- Test: `internal/handlers/auth_handler_test.go`

**Interfaces:**
- Consumes: `middleware.ClaimsFromContext` (Task 3), `repository.TokenRevocationRepository.Revoke` (Task 2).
- Produces: `AuthHandler.Logout(w http.ResponseWriter, r *http.Request)`, mounted at `POST /api/v1/auth/logout` inside the `RequireAuth`-protected route group. This is the last piece of this plan — nothing downstream in this repo consumes it (AutoParts's own consumption of this endpoint is a separate, later plan).

- [ ] **Step 1: Write the failing test**

Add to `internal/handlers/auth_handler_test.go`:

```go
// TestLogout_RevokesTokenForThisSessionOnly proves the core point of this
// endpoint: logging out one token must reject that SAME token on its next
// use, while leaving a second, independently-issued token for the same
// user completely unaffected (e.g. the same account logged in on a second
// browser).
func TestLogout_RevokesTokenForThisSessionOnly(t *testing.T) {
	r, _ := testRouter(t)

	username := "logout-" + uuid.NewString()
	rec := doJSON(t, r, http.MethodPost, "/api/v1/auth/register", "", map[string]string{
		"name": "Logout Me", "username": username, "email": uuid.NewString() + "@example.com", "password": "password123",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var userEnv struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&userEnv)
	userID := userEnv.Data.ID

	login := func() string {
		rec := doJSON(t, r, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
			"identifier": username, "password": "password123",
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("login: expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var loginEnv struct {
			Data struct {
				Token string `json:"token"`
			} `json:"data"`
		}
		json.NewDecoder(rec.Body).Decode(&loginEnv)
		if loginEnv.Data.Token == "" {
			t.Fatal("expected non-empty token")
		}
		return loginEnv.Data.Token
	}

	// Two independent logins, standing in for the same account signed in on
	// two separate browsers/devices.
	tokenA := login()
	tokenB := login()

	// Both work before either is logged out.
	rec = doJSON(t, r, http.MethodGet, "/api/v1/users/"+userID, tokenA, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("tokenA before logout: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, r, http.MethodGet, "/api/v1/users/"+userID, tokenB, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("tokenB before logout: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Log out tokenA only.
	rec = doJSON(t, r, http.MethodPost, "/api/v1/auth/logout", tokenA, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("logout: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// tokenA is now rejected...
	rec = doJSON(t, r, http.MethodGet, "/api/v1/users/"+userID, tokenA, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("tokenA after logout: expected 401, got %d: %s", rec.Code, rec.Body.String())
	}

	// ...but tokenB, the OTHER session for the same user, is untouched.
	rec = doJSON(t, r, http.MethodGet, "/api/v1/users/"+userID, tokenB, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("tokenB after tokenA's logout: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// A second logout call with the now-already-revoked tokenA fails at
	// RequireAuth itself (401) before Logout's own handler body ever runs —
	// there is nothing left to revoke, and the row already exists from the
	// first call.
	rec = doJSON(t, r, http.MethodPost, "/api/v1/auth/logout", tokenA, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("second logout with revoked token: expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd internal/handlers && go test ./... -run TestLogout_RevokesTokenForThisSessionOnly -v`
Expected: FAIL — `POST /api/v1/auth/logout` is not yet registered in `testRouter`, so the first logout call gets a 404, not 200.

- [ ] **Step 3: Write minimal implementation**

In `internal/handlers/auth_handler.go`, add the `Logout` method (place it after `Login`):

```go
// Logout godoc
// @Summary      Revoke the current session's token
// @Description  Revokes only the bearer token used on THIS request — any other active sessions for the same user (e.g. logged in on a second device) are unaffected.
// @Tags         auth
// @Security     BearerAuth
// @Success      200 {object} map[string]bool
// @Failure      401 {object} map[string]string
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing session")
		return
	}
	if err := h.revocationRepo.Revoke(r.Context(), claims.JTI, claims.ExpiresAt.Time); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to revoke token")
		return
	}
	writeJSON(w, http.StatusOK, nil)
}
```

Add the new dependency to `AuthHandler` and its constructor:

```go
type AuthHandler struct {
	svc            *service.AuthService
	revocationRepo *repository.TokenRevocationRepository
}

func NewAuthHandler(svc *service.AuthService, revocationRepo *repository.TokenRevocationRepository) *AuthHandler {
	return &AuthHandler{svc: svc, revocationRepo: revocationRepo}
}
```

Add `"platform-identity-service/internal/repository"` to this file's import block.

This changes `NewAuthHandler`'s signature, so update both of its call sites:

In `cmd/api/main.go`, change:

```go
authHandler := handlers.NewAuthHandler(authService)
```

to:

```go
authHandler := handlers.NewAuthHandler(authService, tokenRevocationRepo)
```

(`tokenRevocationRepo` already exists from Task 4's step 3.) Then add the route inside the existing `RequireAuth`-protected group (next to the other `/users/...` routes, anywhere in that block — e.g. right after `r.Get("/users/{id}/password", userHandler.GetPlainPassword)`):

```go
r.Post("/auth/logout", authHandler.Logout)
```

In `internal/handlers/auth_handler_test.go`'s `testRouter(t)` helper, change:

```go
authHandler := NewAuthHandler(authSvc)
```

to:

```go
tokenRevocationRepo := repository.NewTokenRevocationRepository(db)
authHandler := NewAuthHandler(authSvc, tokenRevocationRepo)
```

(this `tokenRevocationRepo` local variable is the same one already added for `RequireAuth`'s third argument in Task 4 — do not declare it twice; if Task 4's edit already added it to this function, Task 5 just reuses the existing local variable instead of re-declaring it), and inside the `r.Group(...)` block, add:

```go
r.Post("/api/v1/auth/logout", authHandler.Logout)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go build ./... && go test ./... -v`
Expected: PASS across every package, including the new `TestLogout_RevokesTokenForThisSessionOnly` and every pre-existing test (in particular `TestFullFlow_RegisterLoginGetUser`, `TestFullFlow_RegistrationCreatesNoShopMembership`, and `TestRequireAuth_RejectsTokenAfterUserDeactivated`, all of which must still pass unchanged).

- [ ] **Step 5: Commit**

```bash
git add internal/handlers/auth_handler.go internal/handlers/auth_handler_test.go cmd/api/main.go
git commit -m "feat: add POST /auth/logout to revoke the current session's token"
```

---

### Task 6: Full-suite verification and swagger docs regen

**Files:**
- None new — this task only runs commands and, if the project's swagger docs are checked in, regenerates them.

**Interfaces:**
- Consumes: everything from Tasks 1–5.
- Produces: nothing new; this is the plan's final confirmation gate.

- [ ] **Step 1: Run the full test suite**

Run: `go build ./... && go vet ./... && go test ./... -v`
Expected: every package passes, with zero `go vet` warnings. If `TEST_DSN` is unset, these tests fall back to `host=localhost port=5433 user=postgres password=1 dbname=platform_identity sslmode=disable` (see each test file's own fallback) — confirm a Postgres instance is reachable at that DSN (or export `TEST_DSN` pointing at one) before running.

- [ ] **Step 2: Regenerate swagger docs if this project checks them in**

Check whether `internal/docs/docs.go` is checked into git (`git ls-files internal/docs/docs.go`). If it is, and this project has a documented regen command (check `Makefile` or `README.md` for something like `swag init`), run it now so the new `Logout` endpoint's `@Router`/`@Summary` annotations (added in Task 5) actually appear in the generated swagger spec. If no such file/command exists, skip this step — there is nothing to regenerate.

- [ ] **Step 3: Commit (only if Step 2 produced changes)**

```bash
git add internal/docs/
git commit -m "docs: regenerate swagger spec for POST /auth/logout"
```

If Step 2 produced no file changes (either because the file isn't checked in, or `swag init` found nothing new to emit for an already-up-to-date spec), skip this commit — there's nothing to commit.
