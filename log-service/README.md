# log-service

Mərkəzi log anbarı — bütün servislər hər HTTP sorğusunu (method, path, status, müddət) buraya göndərir; servisləri bir yerdən izləmək üçün. `service_logs` cədvəlinin sxem sahibidir.

Hər servisdəki `internal/logclient` paketi iki şey edir:

- **`logclient.New(baseURL, serviceName)`** — fire-and-forget client: log göndərmə ayrıca goroutine-də, 2 saniyəlik timeout ilə gedir; log-service işləmirsə belə əsas sorğu **heç vaxt** bloklanmır və ya uğursuz olmur.
- **`logclient.RequestLogger(client)`** — chi middleware: hər bitmiş sorğunu göndərir. `/health` və `/swagger` səs-küy yaratmasın deyə ötürülür. Səviyyə avtomatik: `5xx → error`, `4xx → warn`, qalanı `info`.

Konfiqurasiya hər servisdə `LOG_SERVICE_URL` env dəyişəni ilə (default `http://localhost:8091`).

## Stack
- Go 1.26 + chi router
- PostgreSQL (`pgx`) — `service_logs` cədvəlinin sahibi

## Setup

```bash
cp .env.example .env
go mod tidy
go run ./cmd/api
```

Default port: **8091**. Swagger UI: http://localhost:8091/swagger/index.html

## Endpoints

| Method | Path                  | Auth | Body / Query |
|--------|-----------------------|------|--------------|
| GET    | /health               | -    | - |
| POST   | /api/v1/logs          | -    | `{service, level?, method?, path?, status?, duration_ms?, message?}` — `level`: `info`(default)/`warn`/`error` |
| GET    | /api/v1/logs          | -    | `?service=&level=&path=&from=&to=&limit=` — `from`/`to` RFC3339, `limit` default 100 / max 500, ən yenilər əvvəldə |
| GET    | /api/v1/logs/services | -    | - (log göndərmiş servislərin siyahısı) |

## Nümunələr

```bash
# Son 100 log (bütün servislər):
curl "http://localhost:8091/api/v1/logs"

# Yalnız bir servisin errorları:
curl "http://localhost:8091/api/v1/logs?service=shop-order-service&level=error"

# Müəyyən path üzrə axtarış (ILIKE substring):
curl "http://localhost:8091/api/v1/logs?path=/orders"

# Tarix aralığı:
curl "http://localhost:8091/api/v1/logs?from=2026-08-05T00:00:00Z&to=2026-08-06T00:00:00Z"

# Hansı servislər log göndərib:
curl "http://localhost:8091/api/v1/logs/services"
```

Bax [../ARCHITECTURE.md](../ARCHITECTURE.md) tam servislərarası axın üçün.
