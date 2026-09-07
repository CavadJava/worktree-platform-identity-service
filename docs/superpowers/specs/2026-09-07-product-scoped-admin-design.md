# Product-Scoped Admin + Admin-Request Workflow — Design

**Date:** 2026-09-07
**Repo:** go-project-practices (platform-identity-service + platform-identity-admin)
**Status:** Approved for planning

## Purpose

Today `system_role` (`user` / `admin` / `superadmin`) is the only gate on admin-panel
access: `user` cannot log in to the panel at all, `admin`/`superadmin` can — and once
in, an `admin` currently sees the exact same Shops/Products/Users pages as a
`superadmin`, with no restriction.

This introduces **product-scoped admin access**: an `admin` (not `superadmin`) is
narrowed to only the Products page, and only to the products they have a
subscription to. Making someone an `admin` for a product is no longer something an
`admin` can do unilaterally — it requires either a `superadmin`'s direct action, or a
request an `admin` files that a `superadmin` approves.

## Scope decisions (from conversation)

- **No new role table.** "Product access" already exists as
  `user_product_subscriptions` — a subscription *is* product access. No separate
  `product_roles` or `is_product_admin` flag is introduced. Whether someone can
  administer a product they're subscribed to is entirely determined by their
  existing, global `system_role` (`admin`/`superadmin`) combined with having a
  subscription row for that product.
- **`admin` scope narrows to Products only.** Shops-lar and İstifadəçilər (Users)
  pages, currently open to any `admin` via `RequireSystemAdmin`, become
  `superadmin`-only. An `admin`'s only page is Products, and there they only see
  products they hold a subscription to — `superadmin` continues to see everything,
  unrestricted, exactly as today.
- **An `admin` with zero subscriptions still logs in successfully** and sees an empty
  Products table — not a blocked login. From there they can browse the *full* product
  list (read-only: names only, no management actions) to know what exists, and can
  file a request for any product they don't yet have a subscription to.
- **Only a `superadmin` can directly promote a `user` to `admin` for a product.** An
  `admin` (regardless of which products they already manage) can never unilaterally
  promote another `user` — they can only file a request. This is a hard rule with no
  exception: even for a product the requesting `admin` already manages, granting a
  *new* person `admin` status still requires `superadmin` approval.
- **The request is always filed by an admin, on behalf of a `user`; the `user` never
  files it themselves.** A `user` has no path to request anything — they are purely
  the *subject* of a request, not the requester. Two things can trigger a request:
  1. An `admin` browsing the system's users picks an existing `user` and asks to make
     them `admin` for one of the `admin`'s own subscribed products.
  2. An `admin` with zero (or fewer) subscriptions requests admin access to a product
     for **themselves** — same request shape, subject == requester's own account, but
     still requires `superadmin` approval like any other request.
- **A request can only target a product the subject does not already have a
  subscription to.** No duplicate/no-op requests for something already granted.
- **Approval queue lives on the Products page itself.** Each product row shows a
  pending-request count/badge; opening it lists the pending requests for that product
  with Approve/Reject actions — no separate "Requests" page.
- **What approval actually does:** the subject user's `system_role` is set to
  `admin` (if it was `user`) and a subscription row for that product is created (or
  left as-is if it already exists, though the request wouldn't have been created for
  a product they already hold). This is the *same* effect as `superadmin` directly
  assigning someone — approval is just the gated path to that same outcome.
- **Rejecting a request** simply marks it rejected and leaves the subject's role and
  subscriptions untouched. A rejected request does not block a future new request for
  the same (user, product) pair.
- **The existing "Subscription idarə et" and "Yeni istifadəçi" actions on the Products
  page are unaffected in shape** — an `admin` can still use them freely for products
  they already manage; those actions do not touch `system_role` and were never gated
  by superadmin approval. Only the *new* "make this user an admin for this product"
  action goes through the request/approval path.

## Data model

```sql
CREATE TABLE IF NOT EXISTS product_admin_requests (
    id UUID PRIMARY KEY,
    product_id UUID NOT NULL REFERENCES products(id),
    subject_user_id UUID NOT NULL REFERENCES users(id),
    requested_by_user_id UUID NOT NULL REFERENCES users(id),
    status TEXT NOT NULL DEFAULT 'pending', -- 'pending' | 'approved' | 'rejected'
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    decided_at TIMESTAMPTZ,
    decided_by_user_id UUID REFERENCES users(id)
);
CREATE INDEX IF NOT EXISTS idx_product_admin_requests_product_id ON product_admin_requests (product_id);
CREATE INDEX IF NOT EXISTS idx_product_admin_requests_status ON product_admin_requests (status);
```

No uniqueness constraint on `(product_id, subject_user_id)` beyond the status check
done in the service layer (only one row may be `pending` at a time for a given pair;
old resolved rows stay as history) — enforced in application code, not the schema,
to keep the migration simple and additive.

## Backend

- `models.ProductAdminRequest{ID, ProductID, SubjectUserID, RequestedByUserID,
  Status, CreatedAt, DecidedAt *time.Time, DecidedByUserID *string}`.
- New `ProductAdminRequestRepository`: `Create`, `ListPendingByProduct`, `GetByID`,
  `SetStatus(ctx, id, status, decidedByUserID string) error`, `HasPending(ctx,
  productID, subjectUserID string) (bool, error)`.
- `ProductService` gains:
  - `ListAllForAdmin(ctx, caller) ([]models.Product, error)` — `superadmin` gets
    every product (same as today's `List`); `admin` gets only products they hold a
    subscription for, via a new `SubscriptionRepository.ListProductIDsByUser`
    lookup joined against `ProductRepository.List`.
  - `ListAllNames(ctx) ([]models.Product, error)` — the unrestricted, read-only
    browse list (id + name only, reuses existing `List`) shown to any `admin` so
    they know what products exist even without a subscription.
  - `RequestProductAdmin(ctx, caller, productID, subjectUserID string)
    (*models.ProductAdminRequest, error)` — caller must be `admin` or
    `superadmin`; rejects if `subjectUserID` already has a subscription to
    `productID`, or already has a pending request for it.
  - `ListPendingRequests(ctx, caller, productID string)
    ([]models.ProductAdminRequest, error)` — `superadmin`-only.
  - `DecideRequest(ctx, caller, requestID string, approve bool)
    (*models.ProductAdminRequest, error)` — `superadmin`-only; on approve, calls
    `UserService.SetSystemRole` (existing method) to set the subject to `admin` if
    not already, then upserts a subscription row (reusing the existing
    `SubscriptionRepository.Upsert`, `Subscripted: true`).
  - `PromoteDirectly(ctx, caller, productID, subjectUserID string)
    (*models.Subscription, error)` — `superadmin`-only direct path (no request
    row created at all): sets `system_role=admin` + upserts subscription, same
    end effect as an approved request.
- New handler routes, all under the existing authenticated group:
  - `GET /products/mine` — list scoped to caller (`admin` sees only their own,
    `superadmin` sees all). Replaces the plain `GET /products` as what the panel's
    Products page calls when authenticated; the existing public `GET /products` is
    untouched for any other unauthenticated caller.
  - `GET /products/browse` — read-only id+name list, any authenticated caller.
  - `POST /products/{id}/admin-requests` — body `{subject_user_id}`, `admin` or
    `superadmin`.
  - `GET /products/{id}/admin-requests` — pending list, `superadmin`-only.
  - `POST /products/{id}/admin-requests/{requestId}/decide` — body
    `{approve: bool}`, `superadmin`-only.
  - `POST /products/{id}/admin` — body `{subject_user_id}`, `superadmin`-only
    direct-promote path (no request created).

## Frontend

- `RequireSystemAdmin` (`platform-identity-admin/src/auth/RequireSystemAdmin.tsx`)
  splits into two gates: `RequireSuperadmin` (Shops-lar, İstifadəçilər — unchanged
  pages, now superadmin-only) and a looser `RequireAdminOrAbove` (Products —
  `admin` or `superadmin`). `App.tsx`'s routes are updated accordingly; the sidebar
  in `AppLayout.tsx` hides the Shop-lar/İstifadəçilər nav items for a plain `admin`.
- `ProductsPage.tsx` calls the new `GET /products/mine` instead of the current
  `GET /products` when rendering the authenticated admin's own table — for a
  `superadmin` this returns everything (identical to today), for an `admin` it's
  pre-filtered server-side.
- Each product row gains a **"Admin sorğuları"** button showing a pending-count
  badge (0 hides the badge, not the button) — opens a modal listing pending
  requests for that product (subject user + who requested + when). A `superadmin`
  sees Approve/Reject buttons on each row; an `admin` viewing their own product's
  requests sees the same list read-only (no action buttons) — useful confirmation
  that their filed request is still pending, with the decision itself always
  requiring `superadmin` (backend enforces this regardless of what the UI renders).
- Each product row also gains an **"Admin təyin et"** button:
  - For `superadmin`: a user picker + "Birbaşa təyin et" (calls the direct-promote
    endpoint) — no request created, immediate effect.
  - For `admin`: the same user picker, but the button reads "Sorğu göndər" and
    calls the request-creation endpoint instead — the UI text itself communicates
    which path is available to the current caller.
- An `admin` with zero subscriptions sees the Products table with a note (or a
  second, clearly-separated "Bütün productlar" read-only list below it) listing
  every product by name only, each with a "Sorğu göndər" action targeting
  themselves as the subject.

## Out of scope (explicitly deferred, not silently dropped)

- A separate, product-scoped role distinct from the global `system_role` (the
  shop-admin/shop-user precedent) — explicitly rejected in favor of reusing
  `system_role` + subscription, per the "Yalnız 'admin'" decision.
- Any notification (email, in-app toast-on-login) when a request is approved or
  rejected — the requester/subject must notice via the panel itself for now.
- Demoting a product-admin back to `user`, or revoking a subscription as part of
  the admin-request flow — the existing "Subscription idarə et" toggle already
  covers turning `subscripted` off; role demotion is a separate, pre-existing
  concern (`UserService.SetSystemRole`) untouched by this design.
