# shop-category-service

Sistem-səviyyəli kataloq: **kateqoriyalar** və **alt-kateqoriyalar**. `categories`/`subcategories` cədvəllərinin sxem sahibidir.

shop-product-service-dəki `product_types`/`product_subtypes` ilə qarışdırmamalı: onlar hər mağazanın **öz məhsulları üçün** təyin etdiyi lokal taksonomiyadır; buradakı kataloq isə bütün marketplace-in gəzinti strukturudur (məs. "Women's Fashion → Women's Dresses") və yalnız **administrator** tərəfindən idarə olunur.

- ID-lər UUID deyil, oxunaqlı **slug**-lardır (`women-fashion`, `wf-dresses`) — seed idempotent qalır, client-lər sabit istinad daşıya bilir. Yaratmada `id` verilməsə addan avtomatik slug düzəldilir (`Pet Supplies` → `pet-supplies`).
- İlk açılışda default kataloq (10 kateqoriya + 57 alt-kateqoriya — taobao-v1 mobil app-ın istifadə etdiyi dəst) `ON CONFLICT DO NOTHING` ilə seed edilir — admin redaktələri heç vaxt əzilmir.
- Kateqoriya silinəndə alt-kateqoriyaları da silinir (`ON DELETE CASCADE`).
- Oxumaq public-dir; yaratma/yeniləmə/silmə administrator tələb edir (authorization-service ilə yoxlanılır).

## Stack
- Go 1.26 + chi router
- PostgreSQL (`pgx`) — `categories`, `subcategories` cədvəllərinin sahibi

## Setup

```bash
cp .env.example .env
go mod tidy
go run ./cmd/api
```

Default port: **8092**. Swagger UI: http://localhost:8092/swagger/index.html

## Endpoints

| Method | Path                                  | Auth                    | Body |
|--------|---------------------------------------|-------------------------|------|
| GET    | /health                               | -                       | - |
| GET    | /api/v1/categories                    | -                       | - (sort_order üzrə) |
| GET    | /api/v1/categories/{id}               | -                       | - |
| GET    | /api/v1/categories/{id}/subcategories | -                       | - |
| GET    | /api/v1/subcategories                 | -                       | - (hamısı) |
| POST   | /api/v1/categories                    | Bearer (administrator)  | `{id?, name, icon?, sort_order}` |
| PUT    | /api/v1/categories/{id}               | Bearer (administrator)  | `{name, icon?, sort_order}` |
| DELETE | /api/v1/categories/{id}               | Bearer (administrator)  | - |
| POST   | /api/v1/categories/{id}/subcategories | Bearer (administrator)  | `{id?, name, image?, sort_order}` |
| PUT    | /api/v1/subcategories/{id}            | Bearer (administrator)  | `{name, image?, sort_order}` |
| DELETE | /api/v1/subcategories/{id}            | Bearer (administrator)  | - |

Bax [../ARCHITECTURE.md](../ARCHITECTURE.md) tam servislərarası axın üçün.
