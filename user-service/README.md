# user-service

İstifadəçi profilinin oxunması/yenilənməsi. `users` cədvəlinin sxemini idarə etmir — [registration-service](../registration-service) yaratdığı eyni Postgres cədvələ oxuyur/yazır.

Qorunan endpoint-lərə gələn hər sorğuda JWT-ni özü doğrulamır — [authorization-service](../authorization-service)-ə HTTP çağırışı ilə token doğrulatdırır (`internal/middleware/auth.go` → `internal/client/authorization_client.go`). Bu səbəbdən bu servisdə `JWT_SECRET` yoxdur.

## Stack
- Go 1.26 + chi router
- PostgreSQL (`pgx` driver) — read/write, migration yoxdur

## Setup

```bash
cp .env.example .env
go mod tidy
go run ./cmd/api
```

Default port: **8082**. Swagger UI: http://localhost:8082/swagger/index.html

`AUTHORIZATION_SERVICE_URL` authorization-service-in ünvanını göstərir (default `http://localhost:8084`).

## Endpoints

| Method | Path             | Auth   | Body |
|--------|------------------|--------|------|
| GET    | /health          | -      | -    |
| GET    | /api/v1/profile  | Bearer | -    |
| PUT    | /api/v1/profile  | Bearer | `{full_name, phone}` |

## Structure

```
cmd/api              entrypoint
internal/config      env config
internal/database    postgres connection (read/write only, no migration)
internal/models      User struct (password_hash yoxdur — bu servis şifrəyə toxunmur)
internal/repository  DB queries (find by id, update profile)
internal/service     profile business logic
internal/client      HTTP client for authorization-service
internal/middleware  auth middleware (authorization-service-ə delegasiya edir)
internal/handlers    HTTP handlers
```

Bax [../ARCHITECTURE.md](../ARCHITECTURE.md) tam servislərarası axın üçün.
