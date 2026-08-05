# localization-service

Sistemdəki mətnlərin/xəta mesajlarının çoxdilli (az/en/ru) idarəetməsi. `translations` cədvəlinin sxem sahibidir.

Bir tərcümə `(namespace, key, locale)` üçlüyü ilə unikaldır (məs. `namespace="errors", key="not_found", locale="az"`).

## Seed data

Servis ilk açılışda avtomatik olaraq bütün digər servislərin response-envelope-unda artıq istifadə olunan **error code-larını** (`bad_request`, `unauthorized`, `forbidden`, `not_found`, `conflict`, `internal_error`, `service_unavailable`, `error`) `errors` namespace-i altında az/en/ru dillərində yükləyir (`ON CONFLICT DO NOTHING` — admin redaktələrini əzmir). Bu, ən praktik başlanğıc nöqtəsidir: istənilən client `error.code`-u alıb bu servisdən lokallaşdırılmış mesaj sorğulaya bilər, hər Go servisin özündəki ingiliscə mesaja güvənmədən.

## Kim nə edə bilər

- **Oxumaq** (`list`/`map`/`lookup`/`locales`) — public, auth tələb olunmur (login/xəta səhifələri autentifikasiyadan əvvəl tərcümələrə ehtiyac duya bilər).
- **Yazmaq** (`upsert`/`delete`) — yalnız `role=administrator`.

`lookup` sorğulanan `locale`-da tapılmasa, servisin `DEFAULT_LOCALE`-una (default `az`) fallback edir.

## Stack
- Go 1.26 + chi router
- PostgreSQL (`pgx`) — `translations` cədvəlinin sahibi

## Setup

```bash
cp .env.example .env
go mod tidy
go run ./cmd/api
```

Default port: **8090**. Swagger UI: http://localhost:8090/swagger/index.html

## Endpoints

| Method | Path                          | Auth                     | Body / Query |
|--------|----------------------------------|------------------------------|------|
| GET    | /health                        | -                              | - |
| GET    | /api/v1/translations            | -                                | opt. `?namespace=&locale=&key=` |
| GET    | /api/v1/translations/map        | -                                | tələb olunur: `?namespace=&locale=` → `{key: value}` |
| GET    | /api/v1/translations/lookup     | -                                | tələb olunur: `?namespace=&key=&locale=` (fallback ilə) |
| GET    | /api/v1/locales                | -                                | - (mövcud olan unikal locale-lar) |
| POST   | /api/v1/translations            | Bearer (administrator)          | `{namespace, key, locale, value}` (upsert) |
| DELETE | /api/v1/translations/{id}       | Bearer (administrator)          | - |

Bax [../ARCHITECTURE.md](../ARCHITECTURE.md) tam servislərarası axın üçün.
