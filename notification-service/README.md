# notification-service

Bildirişləri qəbul edir, **mock** göndərir (konsola loglayır — real email/SMS provideri qoşulmayıb) və `user_id` verildikdə istifadəçinin **inbox**-unda saxlayır. `notifications` cədvəlinin sxem sahibidir.

İki istifadə pattern-i:

- **Köhnə (mock) çağırışlar**: [registration-service](../registration-service) və [shop-service](../shop-service) `{type, to, full_name, message}` göndərir — yalnız konsola loglanır (`to` email-dir, inbox-a düşmür).
- **İnbox çağırışları**: `user_id` verildikdə bildiriş həm loglanır, həm DB-də saxlanılır — istifadəçi `GET /notifications` ilə öz inbox-unu oxuyur, oxunmamışları sayır, oxunmuş işarələyir. İlk real istifadəçisi: [shop-product-service](../shop-product-service) yeni məhsul yaradılanda mağazanın **abunəçilərinə** (`shop_subscriptions`) bildiriş göndərir.

Real provider (SMTP, Twilio və s.) qoşmaq üçün `Send` funksiyasındakı log sətrini əvəz etmək kifayətdir.

## Stack
- Go 1.26 + chi router
- PostgreSQL (`pgx`) — `notifications` cədvəlinin sahibi

## Setup

```bash
cp .env.example .env
go mod tidy
go run ./cmd/api
```

Default port: **8083**. Swagger UI: http://localhost:8083/swagger/index.html

## Endpoints

| Method | Path                                   | Auth   | Body / Qeyd |
|--------|----------------------------------------|--------|-------------|
| GET    | /health                                | -      | - |
| POST   | /api/v1/notifications                  | -      | `{type, to?, user_id?, full_name?, message?}` → 202 `{status:"sent", stored:bool}` — `user_id` verilsə inbox-da saxlanılır |
| GET    | /api/v1/notifications                  | Bearer | öz inbox-u, ən yenilər əvvəldə (`?limit=`, default 50 / max 200) |
| GET    | /api/v1/notifications/unread-count     | Bearer | `{unread: n}` |
| PUT    | /api/v1/notifications/{id}/read        | Bearer | öz bildirişini oxunmuş işarələyir |
| PUT    | /api/v1/notifications/read-all         | Bearer | hamısını oxunmuş işarələyir |

## Nümunə

```bash
# Abunəçi öz inbox-una baxır:
curl http://localhost:8083/api/v1/notifications -H "Authorization: Bearer $TOKEN"
# → [{"type":"new_product","message":"Applicant Shop yeni məhsul əlavə etdi: Yeni Gələn Xalça","is_read":false,...}]
```

Bax [../ARCHITECTURE.md](../ARCHITECTURE.md) tam servislərarası axın üçün.
