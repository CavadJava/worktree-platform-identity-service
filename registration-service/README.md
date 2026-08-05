# registration-service

İstifadəçi qeydiyyatı və login. Email unikaldır (DB səviyyəsində unique constraint) və bu servis `users` cədvəlinin sxem sahibidir (avtomatik migrate edir).

Uğurlu qeydiyyatdan sonra [notification-service](../notification-service)-ə sync HTTP çağırışı ilə "welcome" bildirişi göndərilir (uğursuz olsa belə qeydiyyat uğurlu sayılır, sadəcə loglanır).

Login zamanı verilən JWT-ni [authorization-service](../authorization-service) doğrulayır — bu servis token *yaradır* (authentication), amma doğrulamır (authorization ayrı servisdədir).

## Stack
- Go 1.26 + chi router
- PostgreSQL (`pgx` driver)
- bcrypt password hashing
- JWT generasiyası (`internal/auth` — kiçik, təkcə hash+sign edir)

## Setup

```bash
cp .env.example .env
go mod tidy
go run ./cmd/api
```

Default port: **8081**. Swagger UI: http://localhost:8081/swagger/index.html

## Endpoints

| Method | Path              | Auth | Body |
|--------|-------------------|------|------|
| GET    | /health           | -    | -    |
| POST   | /api/v1/register  | -    | `{email, password, full_name, phone}` |
| POST   | /api/v1/login     | -    | `{email, password}` → `{token, expires_at}` |

## Rollar

Hər qeydiyyat `role="user"`, `shop_id=null`, `shop_role_level=0` alır (bax `internal/roles`). Bunlar login zamanı JWT-nin içinə yazılır:

- `role`: `user` | `administrator` — hesabın əsas rolu; `administrator` sistemin tam idarəçisidir.
- `shop_id` + `shop_role_level`: istifadəçi bir mağazaya aiddirsə, hansı mağaza və hansı hierarxik səviyyə (`1`=chat … `4`=admin) — bax [shop-role-service](../shop-role-service) və [shop-service](../shop-service)-in mağaza açma müraciəti axını.

Rol/səviyyə dəyişikliyi bu servisdən edilə bilməz (özünü admin edən endpoint yoxdur) — [shop-role-service](../shop-role-service) və DB-dən birbaşa müdaxilə lazımdır. Rollar JWT-yə login anında yazılır, ona görə dəyişsə istifadəçi yenidən login olmalıdır.

## `user_seq`

Hər istifadəçi qeydiyyat anında avtomatik, ardıcıl `user_seq` (`BIGSERIAL`) alır — API cavablarında görünür. [shop-order-service](../shop-order-service) sifariş nömrəsi (`order_number`) qurmaq üçün bunu birbaşa DB-dən oxuyur.

## Structure

```
cmd/api              entrypoint
internal/config      env config
internal/database    postgres connection + schema (owns migration)
internal/models      User struct
internal/repository  DB queries (create, find by email)
internal/service     register/login business logic + notification call
internal/auth        bcrypt + JWT signing (authentication module)
internal/client      HTTP client for notification-service
internal/handlers    HTTP handlers
migrations/          raw SQL (reference; app also auto-migrates on boot)
```

Bax [../ARCHITECTURE.md](../ARCHITECTURE.md) tam servislərarası axın üçün.
