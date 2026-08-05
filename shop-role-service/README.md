# shop-role-service

İstifadəçini bir mağazaya hierarxik səviyyə ilə təyin edir/geri alır: `admin(4) > review(3) > add-product(2) > chat(1)`. `users` cədvəlinin `shop_id`/`shop_role_level` sütunlarını yeniləyir — [registration-service](../registration-service) ilə eyni Postgres cədvəlini paylaşır, öz sxemi yoxdur.

**Kim çağıra bilər:**
- **Sistem administratoru** (`role=administrator`) — istənilən istifadəçini istənilən mağazaya, istənilən səviyyə ilə.
- **Mağazanın öz admin(4)-ü** — yalnız **öz mağazası** üçün, yalnız hazırda başqa mağazaya aid olmayan (və ya artıq elə bu mağazada olan) istifadəçiləri idarə edə bilər. Başqa mağazaya müdaxilə `403` ilə rədd olunur.

Bir istifadəçi eyni anda yalnız **bir mağazaya** aid ola bilər. Səviyyələr JWT-yə login anında yazıldığı üçün, dəyişdikdən sonra həmin istifadəçi **yenidən login olmalıdır** ki, yeni token bunu daşısın.

## Stack
- Go 1.26 + chi router
- PostgreSQL (`pgx`) — `users.shop_id`/`users.shop_role_level` sütunlarını oxuyur/yazır, migration aparmır

## Setup

```bash
cp .env.example .env
go mod tidy
go run ./cmd/api
```

Default port: **8085**. Swagger UI: http://localhost:8085/swagger/index.html

## Endpoints

| Method | Path                     | Auth                                         | Body |
|--------|--------------------------|------------------------------------------------|------|
| GET    | /health                  | -                                                | - |
| POST   | /api/v1/roles/assign     | Bearer (administrator / öz mağazasının admin(4)-ü) | `{user_id, shop_id, shop_role_level}` |
| POST   | /api/v1/roles/revoke     | Bearer (administrator / öz mağazasının admin(4)-ü) | `{user_id}` |
| GET    | /api/v1/roles/{user_id}  | Bearer (administrator / öz mağazasının admin(4)-ü) | - |

`shop_role_level` dəyərləri: `1`=chat, `2`=add-product, `3`=review, `4`=admin.

Mağaza yeni açılanda (bax [shop-service](../shop-service)-in müraciət/təsdiq axını) müraciət sahibinə avtomatik `4` (admin) verilir — bu endpoint mövcud mağazaya **əlavə əməkdaş** təyin etmək üçündür, adətən mağazanın öz admin(4)-ü tərəfindən çağırılır.

## İlk administratoru necə təyin etmək olar

Heç kim özünü administrator edə bilməz (təhlükəsizlik üçün). İlk admini birbaşa DB-dən təyin etmək lazımdır:

```bash
psql -h localhost -p 5433 -U postgres -d postgres -c \
  "UPDATE users SET role='administrator' WHERE email='sizin@email.com';"
```

Bax [../ARCHITECTURE.md](../ARCHITECTURE.md) tam servislərarası axın üçün.
