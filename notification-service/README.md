# notification-service

Bildirişləri qəbul edir və **mock** göndərir (konsola loglayır) — real email/SMS provideri qoşulmayıb. [registration-service](../registration-service) uğurlu qeydiyyatdan sonra bura sync HTTP çağırışı edir.

Real provider (SMTP, Twilio və s.) qoşmaq üçün `internal/handlers/notification_handler.go`-da `Send` funksiyasındakı log sətrini əvəz etmək kifayətdir.

## Stack
- Go 1.26 + chi router
- DB yoxdur, stateless

## Setup

```bash
cp .env.example .env
go mod tidy
go run ./cmd/api
```

Default port: **8083**. Swagger UI: http://localhost:8083/swagger/index.html

## Endpoints

| Method | Path                  | Auth | Body |
|--------|-----------------------|------|------|
| GET    | /health               | -    | - |
| POST   | /api/v1/notifications | -    | `{type, to, full_name}` → 202 `{status:"sent"}` |

## Structure

```
cmd/api            entrypoint
internal/config    env config (PORT)
internal/handlers  HTTP handlers (mock send)
```

Bax [../ARCHITECTURE.md](../ARCHITECTURE.md) tam servislərarası axın üçün.
