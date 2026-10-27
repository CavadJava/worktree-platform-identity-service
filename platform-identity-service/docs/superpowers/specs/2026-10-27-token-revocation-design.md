# Single-Session Token Revocation — Design Spec

**Date:** 2026-10-27
**Status:** Approved by user, pending implementation plan

## Context and motivation

This is step 1 of a 2-step project. AutoParts (a separate repo, `me-github/auto-parts`)
wants to stop minting its own local opaque session tokens for its 3 login flows
(admin/user/shop) and instead store the platform-identity-service JWT directly
in its cookies — removing 3 near-identical Postgres-backed session stores and
their Go code. That AutoParts-side work is step 2, specced separately, and is
blocked on this one: a JWT-in-cookie design needs a real way to log a single
session out, and platform-identity-service has no such capability today.

**The gap:** `internal/auth/jwt.go`'s `JWTManager.Verify()` only checks the
HS256 signature and the standard `exp` claim — it is fully stateless, with no
database lookup at all. `middleware.RequireAuth` separately calls
`userRepo.GetByID` and rejects any non-`ACTIVE` user, so deactivating an
*account* already takes effect on that account's very next request. But there
is no way to invalidate one already-issued token without touching the
account — e.g. "log this one browser out" while every other active session for
that same user keeps working. No endpoint exists for this today; only
`POST /auth/register` and `POST /auth/login` exist under `/auth`.

A per-user "bump a token_version column, every token is invalidated" approach
was considered and explicitly rejected: it cannot invalidate one session
without invalidating every session for that user, and multi-device use
(the same shop/user logged in on two browsers) is expected to keep working
independently of each other.

## Approach

Add a per-token identifier (`jti`, a random UUID minted at `Generate()` time)
and a small Postgres blocklist table keyed on it. Logging out writes that
token's `jti` into the blocklist; every authenticated request checks the
blocklist by `jti` in addition to the signature/expiry check it already does.

This is the standard "JWT blocklist" pattern — it keeps the token itself
stateless and self-contained for everything except revocation, and adds
exactly one new table and one new query to the already-existing per-request
DB round trip (`RequireAuth` already does one `userRepo.GetByID` per request;
this adds a second, independent lookup alongside it — see Open Questions
Resolved below for why this stays two separate queries rather than one
combined one).

No cleanup/sweep job for expired blocklist rows is being built. Real traffic
through this service's 3 login flows is low-volume (a handful of admins, a
few hundred shops/users) — the table will stay small for a long time. If it
ever becomes a real problem, that's a separate, later piece of work.

## Data model

New table, added to the existing `database.Migrate()` function (this project
has no numbered migration files — every table is created via one idempotent,
additive SQL block that runs on every service startup):

```sql
CREATE TABLE IF NOT EXISTS token_revocations (
    jti        TEXT PRIMARY KEY,
    revoked_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL
);
```

`expires_at` is copied from the token's own `exp` claim at logout time. It is
stored so that a future cleanup job (if one is ever built) knows which rows
are safe to delete — rows past their own `expires_at` can never match a still-
valid token's signature check anyway, so correctness does not depend on this
column; it exists only to support cleanup later, if ever.

## `Claims` and `JWTManager` changes

`internal/auth/jwt.go`:

- `Claims` gains a `JTI string` field, read from the standard JWT `jti`
  registered claim (`jwt.RegisteredClaims.ID`, exposed as `jti` in the JSON).
- `Generate()` mints a random UUID (`github.com/google/uuid`, already a
  dependency of this project) and sets it as `RegisteredClaims.ID` before
  signing.
- `Verify()` is unchanged — it still only checks signature and expiry. The
  blocklist check is a separate concern, handled by the middleware (see
  below), not folded into `Verify()` itself, so `Verify()` stays a pure,
  dependency-free function exactly as it is today (no DB handle needed to
  call it, which several existing callers — including `identityclient`-side
  decoding in AutoParts, and this project's own tests — rely on).

## New repository: `TokenRevocationRepository`

New file `internal/repository/token_revocation_repository.go`, following this
project's existing small-repository convention (see
`settings_repository.go`, the closest existing analog — single table, two
methods, `database/sql`, no transaction needed):

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

// Revoke marks jti as revoked. expiresAt should be the token's own exp
// claim — stored for a possible future cleanup job, not required for
// correctness today.
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

`ON CONFLICT (jti) DO NOTHING` on `Revoke` makes a double logout call (e.g. a
retried request) a harmless no-op rather than an error.

## New endpoint: `POST /auth/logout`

Added to `internal/handlers/auth_handler.go` as `AuthHandler.Logout`, and
wired in `cmd/api/main.go` **inside** the existing `RequireAuth`-protected
route group (it needs a valid, not-yet-revoked bearer token to know which
`jti` to revoke — an already-logged-out token correctly fails this with 401,
same as any other protected route):

```go
r.Group(func(r chi.Router) {
    r.Use(appmiddleware.RequireAuth(jwtManager, userRepo, tokenRevocationRepo))
    ...
    r.Post("/auth/logout", authHandler.Logout)
    ...
})
```

`AuthHandler` gains a `tokenRevocationRepo *repository.TokenRevocationRepository`
field (constructor-injected, same pattern as its existing `svc` field). The
handler reads the already-authenticated claims from context (see
`middleware.CallerFromContext`-style access — `RequireAuth` will be extended
to also stash the raw `*auth.Claims`, not just the smaller `shopassign.Caller`
projection, so the handler can reach `claims.JTI` and `claims.ExpiresAt`
without re-parsing the Authorization header itself), then calls
`tokenRevocationRepo.Revoke(ctx, claims.JTI, claims.ExpiresAt.Time)` and
returns `200 {"success": true}`.

No request body. No response payload beyond a success flag — there is
nothing else to return.

## `RequireAuth` middleware changes

`internal/middleware/auth.go`:

`RequireAuth` gains a third parameter, `tokenRevocationRepo *repository.TokenRevocationRepository`,
and performs the blocklist check as a second, independent query right after
the existing `jwt.Verify()` call and before the existing `userRepo.GetByID`
call (cheapest check first — a revoked token never needs to touch the users
table at all):

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

func ClaimsFromContext(ctx context.Context) (*auth.Claims, bool) {
    claims, ok := ctx.Value(claimsKey).(*auth.Claims)
    return claims, ok
}
```

(`claimsKey` is a new `contextKey` constant alongside the existing `callerKey`.)

This is a breaking change to `RequireAuth`'s signature — every call site must
be updated. There is exactly one call site today (`cmd/api/main.go`'s route
group above), so this is mechanical, not risky.

## Open questions resolved during brainstorming

- **Why not fold the blocklist check into `userRepo.GetByID` as one combined
  query (e.g. a JOIN)?** Considered and explicitly rejected in favor of
  keeping it as two independent, simple queries. The combined-query version
  would require changing `UserRepository.GetByID`'s own query/signature,
  which is used by other callers beyond `RequireAuth` — simplicity and not
  touching an already-widely-used method won over the marginal performance
  gain of one fewer round trip.
- **Why `jti` + blocklist instead of a per-user `token_version` column?**
  `token_version` was proposed first as the simpler option, but it can only
  revoke *all* of a user's tokens at once — it cannot log out one specific
  browser session while leaving the user's other active sessions alone. The
  user explicitly requires independent multi-device sessions (e.g. a shop
  logged in on both a desktop and a phone, logging out of one must not log
  out the other), so `token_version` was rejected.
- **Why no blocklist cleanup/sweep job?** YAGNI — real traffic through this
  service's login flows is low volume; the table will not grow fast enough
  to matter. Revisit only if it actually becomes a problem.

## Testing

Following this project's existing test conventions (`TEST_DSN` env var,
`internal/repository/repository_test.go` and `internal/handlers/auth_handler_test.go`
as the closest existing analogs):

- `internal/repository/token_revocation_repository_test.go`: `Revoke` then
  `IsRevoked` returns true; `IsRevoked` on a never-revoked `jti` returns
  false; calling `Revoke` twice on the same `jti` does not error.
- `internal/auth/jwt_test.go` (extend existing, if present, else new):
  `Generate` produces a non-empty, unique `jti` on every call; `Verify`
  round-trips it back out in `Claims.JTI`.
- `internal/middleware/auth_test.go` (extend existing, if present, else new):
  a request with a revoked token's `jti` in the blocklist gets 401 with the
  "token has been revoked" message, even though the signature and `exp` are
  still both valid.
- `internal/handlers/auth_handler_test.go` (extend existing): `POST
  /auth/logout` with a valid bearer token returns 200 and the same token can
  no longer pass `RequireAuth` on a subsequent request; a second logout call
  with the same (now-revoked) token correctly gets 401 (it fails
  `RequireAuth` itself before `Logout`'s own handler body ever runs, which is
  the correct behavior — there is nothing left to revoke a second time, and
  the blocklist row already exists from the first call).

## Out of scope (explicitly deferred)

- Blocklist cleanup/sweep job (YAGNI, see above).
- "Log out all my other sessions" (would need per-user enumeration of active
  `jti`s, which this design does not track — only revoked ones are stored).
- Any change to `JWTManager.Verify()`'s own signature or behavior.
- The AutoParts-side consumption of this endpoint (separate spec, step 2 of
  this 2-step project — depends on this one being implemented and deployed
  first).
