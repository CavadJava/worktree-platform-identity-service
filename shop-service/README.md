# shop-service

Mağazaların CRUD idarəetməsi + mağaza açma **2 mərhələli müraciət/təsdiq** axını. `shops` və `shop_applications` cədvəllərinin sxem sahibidir (avtomatik migrate edir).

Mağaza özü-özünə açılmır — istifadəçi müraciət göndərir, administrator ya rədd edir, ya da **formu göndərir** (bu zaman müvəqqəti mağaza yaranır və müraciətçi onu özü doldurur), sonda administrator yekun qərar (təsdiq/rədd) verir.

## Şop CRUD

- **Birbaşa yaratma (`POST /shops`)**: yalnız administrator (skip-the-line bootstrap üçün) — adi istifadəçilər mütləq müraciət axınından keçməlidir.
- **Yeniləmək**: `review(3)`+ səviyyəli mağaza əməkdaşı və ya administrator. (Müvəqqəti mağazanı doldurmaq da elə bu endpoint-dir — mağazanın admin(4) sahibi öz `PUT /shops/{id}`-ni çağırır.)
- **Silmək**: yalnız `admin(4)` səviyyəli mağaza əməkdaşı və ya administrator.
- **Oxumaq (`GET /shops/{id}`)**: public, auth tələb olunmur, müvəqqəti mağazalar da daxil (sahib/admin baxa bilsin deyə).
- **Siyahı (`GET /shops`)**: public, amma **müvəqqəti (təsdiqlənməmiş) mağazalar göstərilmir** — yalnız vetted/approved mağazalar görünür.

## Mağaza açma müraciəti (2 mərhələ)

1. İstənilən login olmuş istifadəçi `POST /shop-applications` ilə müraciət göndərir — status `pending`, hələ heç bir mağaza yoxdur.
2. Administrator ya birbaşa **rədd edir** (`.../reject`, mağaza olmadığı üçün heç nə silinmir), ya da **`POST .../send-form`** çağırır:
   - Müvəqqəti (`temporary=true`) mağaza yaranır.
   - Müraciətçi avtomatik onun **admin(4)** səviyyəli sahibi olur.
   - Status `form_sent`-ə keçir, müraciətçiyə mock bildiriş göndərilir.
3. Müraciətçi **yenidən login olur** (token indi `shop_id`+`admin(4)` daşıyır) və formu **özü doldurur** — sadəcə adi `PUT /shops/{id}` ilə. Ayrıca form API-si yoxdur, müvəqqəti mağazanın özü formdur.
4. Administrator yekun qərar verir:
   - **`.../approve`**: mağaza `temporary=false` olur → indi `GET /shops` siyahısında görünür.
   - **`.../reject`**: müvəqqəti mağaza silinir, müraciətçinin `shop_id`/`shop_role_level` sıfırlanır, səbəb (`reason`) saxlanılır və bildirişlə göndərilir.
5. Mağazanın admin(4) sahibi sonra [shop-role-service](../shop-role-service) vasitəsilə (sistem administratoru olmadan, özü) öz mağazasına əməkdaş təyin edə bilər.

Hər dəyişiklikdə (`send-form`/`approve`/`reject`) [notification-service](../notification-service)-ə mock bildiriş göndərilir (müraciətçinin email-i `users` cədvəlindən oxunur).

## Abunəlik

İstənilən login olmuş istifadəçi bir mağazaya abunə ola bilər (sahiblik/səviyyə tələb olunmur) — `shop_subscriptions (user_id, shop_id)`, idempotent (`ON CONFLICT DO NOTHING`).

## Stack
- Go 1.26 + chi router
- PostgreSQL (`pgx`) — `shops`, `shop_applications`, `shop_subscriptions` cədvəllərinin sahibi + `users.shop_id`/`shop_role_level`-ə yazır (send-form/approve/reject zamanı)
- notification-service-ə HTTP client (mock bildiriş)

## Setup

```bash
cp .env.example .env
go mod tidy
go run ./cmd/api
```

Default port: **8086**. Swagger UI: http://localhost:8086/swagger/index.html

## Endpoints

| Method | Path                                     | Auth                                  | Body |
|--------|--------------------------------------------|-----------------------------------------|------|
| GET    | /health                                   | -                                        | - |
| POST   | /api/v1/shops                             | Bearer (administrator)                  | `{name, description}` |
| GET    | /api/v1/shops                             | -                                        | - (opt. `?owner_id=`, müvəqqəti mağazalar xaric) |
| GET    | /api/v1/shops/{id}                        | -                                        | - |
| PUT    | /api/v1/shops/{id}                        | Bearer (review(3)+ / administrator)     | `{name, description}` |
| DELETE | /api/v1/shops/{id}                        | Bearer (admin(4) / administrator)       | - |
| POST   | /api/v1/shop-applications                 | Bearer (istənilən login istifadəçi)     | `{name, description}` |
| GET    | /api/v1/shop-applications                 | Bearer (administrator)                  | - (opt. `?status=pending\|form_sent\|approved\|rejected`) |
| GET    | /api/v1/shop-applications/{id}            | Bearer (müraciətçi özü / administrator) | - |
| POST   | /api/v1/shop-applications/{id}/send-form  | Bearer (administrator)                  | - |
| POST   | /api/v1/shop-applications/{id}/approve    | Bearer (administrator)                  | - |
| POST   | /api/v1/shop-applications/{id}/reject     | Bearer (administrator)                  | `{reason}` (opt.) |
| POST   | /api/v1/shops/{id}/subscribe              | Bearer (istənilən login istifadəçi)     | - |
| DELETE | /api/v1/shops/{id}/subscribe              | Bearer (istənilən login istifadəçi)     | - |
| GET    | /api/v1/subscriptions                     | Bearer (istənilən login istifadəçi)     | - (öz abunəlikləri) |

Bax [../ARCHITECTURE.md](../ARCHITECTURE.md) tam servislərarası axın üçün.
