# Shop Category — Design

**Date:** 2026-08-19
**Repo:** go-project-practices (platform-identity-service + platform-identity-admin)
**Status:** Approved for planning

## Purpose

Every Shop registered in Teslahubs belongs to one business sector. This adds
a required `category` field to Shop, seeded with three fixed values:
Avtomobil satışı (car sales, e.g. turbo.az-style), Gözəllik xidmətləri
(beauty services), Avtomobil detal xidmətləri (auto parts/detailing
services).

## Scope decisions (from brainstorming)

- **This step is category selection only** — no category-specific fields
  (e.g. make/model for car sales, service lists for beauty) are added yet.
  Those are a deliberately deferred follow-on, not part of this spec.
- **Categories are a fixed, hardcoded 3-row seed**, not a superadmin-managed
  CRUD resource — matches this codebase's existing pattern for
  `system_roles`/`shop_roles` (seeded via migration, not user-editable).
  Adding category management is out of scope; if a real need for a 4th
  category appears later, that becomes its own small follow-on task.
- **Category is required** on every Shop, enforced at the DB level
  (`NOT NULL` after backfill).
- **This is the first real backward-compatible migration on this branch.**
  Every prior schema change on this branch used a full `DROP TABLE` rebuild,
  justified at the time because there was no data worth preserving. That
  justification no longer holds — this branch now has real Shops, Users,
  and memberships created through actual use (verified live, tested,
  demoed). This migration must be additive and preserve existing rows:
  add the column nullable, backfill every existing Shop to category id 1
  (Avtomobil satışı — the first-listed, most general category), then set
  `NOT NULL`. No `DROP TABLE` anywhere in this change.

## Data model

```sql
CREATE TABLE shop_categories (
	id SMALLSERIAL PRIMARY KEY,
	name TEXT UNIQUE NOT NULL
);
-- seed, in this exact order (id assignment depends on it):
-- (1, 'avtomobil-satisi')
-- (2, 'gozellik-xidmetleri')
-- (3, 'avtomobil-detal-xidmetleri')

ALTER TABLE shops ADD COLUMN IF NOT EXISTS category_id SMALLINT REFERENCES shop_categories(id);
UPDATE shops SET category_id = 1 WHERE category_id IS NULL;
ALTER TABLE shops ALTER COLUMN category_id SET NOT NULL;
```

All three statements must be idempotent and safe to run on every service
startup (matching this service's existing `Migrate()` convention of
running unconditionally at boot) — `ADD COLUMN IF NOT EXISTS`, an `UPDATE`
that only touches NULL rows (a no-op on repeat runs), and `SET NOT NULL`
(a no-op if already set).

Category names use the same lowercase-hyphenated convention as
`shop-admin`/`shop-user` in `shop_roles`, for consistency with the rest of
this service's seeded-enum naming.

## Backend

- `models.ShopCategory{ID int16, Name string}`.
- `models.Shop` gains `CategoryID int16` and `CategoryName string` (the
  latter populated by a join, following the existing
  `User.SystemRoleName`/`ShopMembership.ShopRoleName` precedent for
  join-only display fields).
- New `ShopCategoryRepository.List(ctx) ([]models.ShopCategory, error)` —
  trivial, mirrors `SystemRoleRepository`/`ShopRoleRepository`.
- `ShopRepository.Create` and the `shops` SELECT queries updated to
  include `category_id`/joined category name.
- `ShopService.Create(ctx, name string, categoryID int16) (*models.Shop, error)`
  — category is now a required parameter, validated against the seeded
  set (invalid id → a new `ErrInvalidShopCategory`).
- New `GET /shop-categories` endpoint (public, unauthenticated — same
  tier as `GET /shop-roles`/`GET /system-roles`) to populate the frontend
  dropdown.
- `POST /shops`'s request body gains a required `category_id` field.
- `shopResponse` (and anywhere a Shop is serialized) gains `category_id`
  and `category_name`.

## Frontend

- `api/types.ts`: `ShopCategory{id, name}`; `Shop` gains `category_id`,
  `category_name`.
- New `api/shopCategories.ts`: `listShopCategories()`.
- `ShopsPage.tsx`:
  - "Yeni shop" modal gains a required category `Select`, populated from
    `listShopCategories()`.
  - The shops table gains a **Kateqoriya** column showing `category_name`.

## Out of scope (explicitly deferred, not silently dropped)

- Category-specific fields (car make/model, beauty service lists, etc.).
- Superadmin category management UI/endpoints (create/rename/delete a
  category).
- Filtering/searching shops by category.
- Any change to Product or User — this spec touches Shop only.
