# Product Profile (Description + Tech Stack + Subprojects) — Design

**Date:** 2026-09-07
**Repo:** go-project-practices (platform-identity-service + platform-identity-admin)
**Status:** Approved for planning

## Purpose

Each Product (Teslahubs, ESound, Xerite, Vault, ...) currently has only a
name and a subscription-management action. This adds a profile to every
Product: a free-text description, a comma-separated tech stack, and a
list of named subprojects/modules it contains internally (e.g. Teslahubs
→ auth-service, admin-panel, mobile-app). Superadmin fills this in
directly through the admin panel — no external integration (GitHub API
etc.) is involved.

## Scope decisions (from brainstorming)

- **Manually entered by superadmin in the panel** — not pulled from any
  external source. This keeps the feature small and matches how every
  other piece of data in this admin panel already works (superadmin
  types it in, nothing is auto-discovered).
- **Three fields**: description (free text), tech stack (comma-separated
  tags, e.g. "React, Go, Postgres" — a single TEXT column, not a
  constrained enum/dropdown, so a new technology never requires a code
  change), and subprojects (a real list, not a single blob).
- **Subprojects are a proper list UI**, not a flattened textarea — each
  subproject is its own name+description pair, addable/removable
  individually. This is the one place in this feature that isn't "just a
  text field," because a list of named items with their own descriptions
  genuinely benefits from real list semantics (add one, remove one,
  without re-typing the rest) — the same reasoning that made
  `ShopMembersPage`'s member list a real table rather than a text blob.
- **Backward-compatible migration** — this branch's `Migrate()` is now
  fully idempotent/additive (see the September 2026 fix removing all
  `DROP TABLE`s). This profile addition continues that discipline:
  `ALTER TABLE products ADD COLUMN IF NOT EXISTS ... DEFAULT ''`, plus a
  brand-new `product_subprojects` table via `CREATE TABLE IF NOT EXISTS`.
  Every existing Product (Teslahubs, ESound, Xerite, Vault) gets an empty
  profile by default — no data loss, no required backfill beyond the
  column defaults themselves.

## Data model

```sql
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

`tech_stack` stores the raw comma-separated string as typed (e.g.
"React, Go, Postgres") — splitting/trimming into a tag list happens at
render time in the frontend, not persisted as a separate structure. This
mirrors how this codebase already treats other free-text-with-display-
formatting fields rather than introducing a new join table for something
that's still fundamentally a single edited string.

## Backend

- `models.Product` gains `Description string`, `TechStack string`.
- New `models.ProductSubproject{ID, ProductID, Name, Description string; CreatedAt time.Time}`.
- `ProductRepository`:
  - `GetByID` query extended to include `description`, `tech_stack`.
  - New `Update(ctx, productID string, description, techStack string) error`
    — always both fields together (no partial-update complexity needed
    here, unlike `UserUpdate`'s nil-field pattern, since this form always
    submits both).
- New `SubprojectRepository`: `Create`, `ListByProduct`, `Delete` — same
  shape as `ShopMembershipRepository`'s CRUD methods.
- `ProductService` gains `UpdateProfile(ctx, caller, productID, description, techStack) (*models.Product, error)`
  (superadmin-only, reusing the existing `SystemRoleSuperadmin` check
  already in `ProductService.Create`), `AddSubproject`, `ListSubprojects`,
  `RemoveSubproject` (all superadmin-only).
- New handler routes, all under the existing superadmin-gated route
  group:
  - `POST /products/{id}/profile` — body `{description, tech_stack}`,
    returns the updated `productResponse` (extended with the two new
    fields).
  - `POST /products/{id}/subprojects` — body `{name, description}`,
    returns the created subproject.
  - `GET /products/{id}/subprojects` — list.
  - `DELETE /products/{id}/subprojects/{subId}` — remove.
- `productResponse` gains `description`, `tech_stack`. A separate
  `subprojectResponse{id, product_id, name, description, created_at}`.

## Frontend

- `api/types.ts`: `Product` gains `description?: string`,
  `tech_stack?: string`; new `Subproject{id, product_id, name, description, created_at}`.
- New `api/products.ts` functions: `updateProductProfile`,
  `listSubprojects`, `addSubproject`, `removeSubproject`.
- `ProductsPage.tsx`: each row gains a **"Profil"** button next to the
  existing "Subscription idarə et" button. Opens a modal with:
  - Description: `Input.TextArea`.
  - Tech stack: `Input`, comma-separated, rendered as AntD `Tag`s on
    display (split/trim client-side, matching the "not persisted as a
    structure" decision above).
  - Subprojects: a small table (name, description, remove button) plus
    an "əlavə et" mini-form (name + description inputs) below it —
    mirrors `ShopMembersPage`'s existing member-list-plus-add-form
    pattern for visual/interaction consistency.

## Out of scope (explicitly deferred, not silently dropped)

- Any external/automatic source for tech stack or subprojects (GitHub API
  or similar) — purely manual entry for now.
- Editing a subproject's name/description in place — only add/remove; if
  a name needs correcting, remove and re-add (small enough list that this
  is an acceptable interim tradeoff, matching this codebase's general
  bias toward the simplest thing that works until a real need for more
  appears).
- Any constraint or validation on tech stack values (no dropdown, no
  fixed vocabulary) — freeform text is intentional here.
