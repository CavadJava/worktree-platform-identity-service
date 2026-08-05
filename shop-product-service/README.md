# shop-product-service

Mağaza məhsullarının CRUD idarəetməsi. `products` cədvəlinin sxem sahibidir.

İcazə yoxlaması **tamamilə JWT-dən** qərarlaşdırılır — cross-service çağırış yoxdur. Cari istifadəçinin tokenindəki `shop_id`/`shop_role_level` hədəf məhsulun `shop_id`-si ilə müqayisə olunur:

- **Yaratmaq/yeniləmək**: `add-product(2)`+ səviyyəli mağaza əməkdaşı və ya administrator.
- **Silmək**: `review(3)`+ səviyyəli mağaza əməkdaşı və ya administrator.
- **Oxumaq/siyahı**: public.

Hierarxiya: `admin(4) > review(3) > add-product(2) > chat(1)` — yüksək səviyyə aşağı səviyyələrin bütün icazələrini əhatə edir.

## Favoritlər

İstənilən login olmuş istifadəçi bir məhsulu favoritə əlavə edə bilər (sahiblik/səviyyə tələb olunmur) — `product_favorites (user_id, product_id)`, idempotent (`ON CONFLICT DO NOTHING`).

## Stack
- Go 1.26 + chi router
- PostgreSQL (`pgx`) — `products`, `product_favorites` cədvəllərinin sahibi

## Setup

```bash
cp .env.example .env
go mod tidy
go run ./cmd/api
```

Default port: **8087**. Swagger UI: http://localhost:8087/swagger/index.html

## Endpoints

| Method | Path                 | Auth                                     | Body |
|--------|----------------------|--------------------------------------------|------|
| GET    | /health              | -                                            | - |
| POST   | /api/v1/products     | Bearer (add-product(2)+ / administrator)    | `{shop_id, name, description, price, stock}` |
| GET    | /api/v1/products     | -                                             | - (opt. `?shop_id=`) |
| GET    | /api/v1/products/{id}| -                                             | - |
| PUT    | /api/v1/products/{id}| Bearer (add-product(2)+ / administrator)    | `{name, description, price, stock}` |
| DELETE | /api/v1/products/{id}| Bearer (review(3)+ / administrator)         | - |
| POST   | /api/v1/products/{id}/favorite | Bearer (istənilən login istifadəçi) | - |
| DELETE | /api/v1/products/{id}/favorite | Bearer (istənilən login istifadəçi) | - |
| GET    | /api/v1/favorites     | Bearer (istənilən login istifadəçi)          | - (öz favoritləri) |

Bax [../ARCHITECTURE.md](../ARCHITECTURE.md) tam servislərarası axın üçün.
