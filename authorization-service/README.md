# authorization-service

JWT-lərin doğrulanması. [registration-service](../registration-service) tokeni yaradır (imzalayır), bu servis isə doğrulayır — authentication (kimsən) və authorization (icazən varmı) ayrı servislərdə saxlanılır.

Digər servislər (məs. [user-service](../user-service)) qorunan endpoint-lərə gələn sorğularda `Authorization` başlığını olduğu kimi bura göndərir və cavabda `user_id` alır.

`JWT_SECRET` registration-service ilə **eyni** olmalıdır (paylaşılan simmetrik açar).

## Stack
- Go 1.26 + chi router
- Stateless — DB yoxdur

## Setup

```bash
cp .env.example .env   # JWT_SECRET-i registration-service ilə eyni et
go mod tidy
go run ./cmd/api
```

Default port: **8084**. Swagger UI: http://localhost:8084/swagger/index.html

## Endpoints

| Method | Path                | Auth   | Cavab |
|--------|---------------------|--------|-------|
| GET    | /health             | -      | - |
| POST   | /api/v1/authorize   | Bearer | 200 `{success:true,data:{user_id,role,shop_id,shop_role_level}}` / 401 `{success:false,error:{...}}` |

`role`, `shop_id`, `shop_role_level` DB-dən deyil, birbaşa token-in claim-lərindən çıxarılır (stateless) — token login anındakı vəziyyəti əks etdirir.

## Structure

```
cmd/api            entrypoint
internal/config    env config (JWT_SECRET, PORT)
internal/auth      JWT doğrulama (Verify)
internal/handlers  HTTP handlers
```

Bax [../ARCHITECTURE.md](../ARCHITECTURE.md) tam servislərarası axın üçün.
