# go-project-practices

7 ayrı Go mikroservisi: qeydiyyat/login, profil, bildiriş, autorizasiya, mağaza rolları, mağaza CRUD + müraciət axını, məhsul CRUD. Arxitektura və servislərin bir-biri ilə əlaqəsi üçün bax [ARCHITECTURE.md](ARCHITECTURE.md).

## Tələblər

- Go 1.26+
- PostgreSQL, `localhost:5433`, `postgres` bazası, şifrə `.env` fayllarında təyin olunub (default `1`)
- `psql` (opsional, ilk administratoru təyin etmək üçün) — `brew install libpq && export PATH="/opt/homebrew/opt/libpq/bin:$PATH"`

## Servisləri işə salmaq

Hər servisin öz `.env`-i var (repo-da hazır, lokal inkişaf üçün). 7 ayrı terminalda:

```bash
cd notification-service   && go run ./cmd/api   # :8083
cd authorization-service  && go run ./cmd/api   # :8084
cd registration-service   && go run ./cmd/api   # :8081
cd user-service           && go run ./cmd/api   # :8082
cd shop-role-service      && go run ./cmd/api   # :8085
cd shop-service           && go run ./cmd/api   # :8086
cd shop-product-service   && go run ./cmd/api   # :8087
```

Sıra fərq etmir, amma tam axın üçün hamısı ayaqda olmalıdır. Yoxlamaq:

```bash
for p in 8081 8082 8083 8084 8085 8086 8087; do curl -s http://localhost:$p/health; echo " :$p"; done
```

Hər servisin Swagger UI-ı: `http://localhost:<port>/swagger/index.html`

## Cavab formatı

Bütün endpoint-lər eyni zərfi (envelope) qaytarır:

```json
{"success": true, "data": {...}}
{"success": false, "error": {"code": "forbidden", "message": "..."}}
```

`error.code` sabit dəyərlərdən biridir: `bad_request`, `unauthorized`, `forbidden`, `not_found`, `conflict`, `internal_error`, `service_unavailable` — frontend mesaj mətni yox, bu koda görə budaqlana bilər.

## Rol modeli — qısa

- `role`: `user` | `administrator` — hesabın əsas rolu.
- `shop_id` + `shop_role_level`: istifadəçi **bir mağazaya** aiddir, orada hierarxik səviyyə daşıyır:

  ```
  admin(4) > review(3) > add-product(2) > chat(1)
  ```

- Mağaza **özü-özünə açılmır**, 2 mərhələli müraciət axını var: istifadəçi müraciət göndərir → administrator ya rədd edir, ya da **formu göndərir** (bu zaman müvəqqəti mağaza yaranır, müraciətçi onun admin(4)-ü olur və məlumatları özü doldurur) → administrator yekun **təsdiq/rədd** verir.
- Mağazanın admin(4) sahibi sonra **özü** (sistem admini olmadan) öz mağazasına əməkdaş təyin edə bilər.
- Rədd olunanda səbəb mock bildirişlə (notification-service) göndərilir.

Tam izah üçün bax [ARCHITECTURE.md](ARCHITECTURE.md#rol-modeli).

## Nümunə API axını

Aşağıdakı ssenari tam test edilib: istifadəçi qeydiyyatı → admin təyini → mağaza açma müraciəti → form göndərmə (müvəqqəti mağaza) → müraciətçi özü doldurur → yekun təsdiq → mağazanın admin(4) sahibi özü əməkdaşa rol verir → əməkdaş səviyyə iyerarxiyası → rədd olunan müraciət (səbəblə).

### 1. Müraciətçi qeydiyyatdan keçir və login olur

```bash
curl -X POST http://localhost:8081/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{"email":"applicant@example.com","password":"password123","full_name":"Shop Applicant"}'
```

```json
{"success":true,"data":{"id":"65fc032c-...","email":"applicant@example.com","full_name":"Shop Applicant","role":"user","shop_role_level":0,"created_at":"...","updated_at":"..."}}
```

```bash
curl -X POST http://localhost:8081/api/v1/login -H "Content-Type: application/json" \
  -d '{"email":"applicant@example.com","password":"password123"}'
# APPLICANT_TOKEN=...
```

### 2. İlk administratoru təyin et (bir dəfəlik bootstrap, birbaşa DB)

Sistemdə özünü admin edən heç bir endpoint yoxdur (təhlükəsizlik üçün qəsdən belədir):

```bash
curl -X POST http://localhost:8081/api/v1/register -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"password123","full_name":"Admin"}'

psql -h localhost -p 5433 -U postgres -d postgres -c \
  "UPDATE users SET role='administrator' WHERE email='admin@example.com';"

curl -X POST http://localhost:8081/api/v1/login -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"password123"}'
# ADMIN_TOKEN=...
```

### 3. Müraciətçi mağaza açmaq üçün müraciət göndərir

```bash
curl -X POST http://localhost:8086/api/v1/shop-applications \
  -H "Authorization: Bearer $APPLICANT_TOKEN" -H "Content-Type: application/json" \
  -d '{"name":"Applicant Shop","description":"Ev əşyaları satışı"}'
```

```json
{"success":true,"data":{"id":"d292d691-...","applicant_id":"65fc032c-...","name":"Applicant Shop","template_version":"v1","status":"pending","created_at":"...","updated_at":"..."}}
```

`APP_ID`-ni saxla. Bu mərhələdə heç bir mağaza yoxdur.

### 4. Administrator gözləyən müraciətlərə baxır

```bash
curl "http://localhost:8086/api/v1/shop-applications?status=pending" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

### 5. Administrator formu göndərir — müvəqqəti mağaza yaranır

```bash
curl -X POST "http://localhost:8086/api/v1/shop-applications/$APP_ID/send-form" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

```json
{"success":true,"data":{"id":"d292d691-...","status":"form_sent","reviewer_id":"...","shop_id":"e57e0a78-...","...":"..."}}
```

`SHOP_ID`-ni saxla. Müraciətçi avtomatik bu mağazanın **admin(4)** sahibi olur, mock bildiriş göndərilir. Mağaza hələ `GET /shops` public siyahısında **görünmür** (`temporary=true`).

### 6. Müraciətçi yenidən login olur və formu özü doldurur

Rollar/səviyyələr JWT-yə login anında yazılır — DB-də dəyişsə də köhnə token yenilənmir. Ayrıca "form" API-si yoxdur, sadəcə adi `PUT /shops/{id}` — müvəqqəti mağazanın özü formdur:

```bash
curl -X POST http://localhost:8081/api/v1/login -H "Content-Type: application/json" \
  -d '{"email":"applicant@example.com","password":"password123"}'
# APPLICANT_TOKEN=... (indi shop_id=SHOP_ID, shop_role_level=4)

curl -X PUT "http://localhost:8086/api/v1/shops/$SHOP_ID" \
  -H "Authorization: Bearer $APPLICANT_TOKEN" -H "Content-Type: application/json" \
  -d '{"name":"Applicant Shop","description":"Ev əşyaları satışı, ünvan: Bakı, Nizami 12"}'
```

### 7. Administrator yekun təsdiq verir — mağaza public olur

```bash
curl -X POST "http://localhost:8086/api/v1/shop-applications/$APP_ID/approve" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
# → status: "approved", mağaza artıq GET /shops siyahısında görünür
```

(Rədd üçün: `POST .../reject` gövdəsi `{"reason":"..."}` — `pending` mərhələsində çağırılsa heç nə silinmir, `form_sent` mərhələsində isə müvəqqəti mağaza silinir və müraciətçinin `shop_id`/`shop_role_level`-i sıfırlanır; hər iki halda səbəb mock bildirişlə göndərilir.)

### 8. Mağazanın admini məhsul əlavə edir

```bash
curl -X POST http://localhost:8087/api/v1/products \
  -H "Authorization: Bearer $APPLICANT_TOKEN" -H "Content-Type: application/json" \
  -d '{"shop_id":"'"$SHOP_ID"'","name":"Divan","description":"3 nəfərlik","price":899.99,"stock":5}'
```

### 9. Mağazanın öz sahibi (admin(4)) — sistem administratoru OLMADAN — əməkdaşa rol verir

```bash
# Yeni istifadəçi qeydiyyatdan keçir. Mağaza SAHİBİ (APPLICANT_TOKEN, admin(4)) ona öz mağazasında chat(1) verir:
curl -X POST http://localhost:8085/api/v1/roles/assign \
  -H "Authorization: Bearer $APPLICANT_TOKEN" -H "Content-Type: application/json" \
  -d '{"user_id":"'"$STAFF_ID"'","shop_id":"'"$SHOP_ID"'","shop_role_level":1}'

# Əməkdaş yenidən login olur, məhsul əlavə etməyə cəhd edir:
curl -X POST http://localhost:8087/api/v1/products \
  -H "Authorization: Bearer $STAFF_TOKEN" -H "Content-Type: application/json" \
  -d '{"shop_id":"'"$SHOP_ID"'","name":"Stul","price":50,"stock":20}'
# → 403 {"success":false,"error":{"code":"forbidden","message":"insufficient permission: ..."}} (chat(1) kifayət etmir)

# Sahib əməkdaşı özü add-product(2)-yə yüksəldir:
curl -X POST http://localhost:8085/api/v1/roles/assign \
  -H "Authorization: Bearer $APPLICANT_TOKEN" -H "Content-Type: application/json" \
  -d '{"user_id":"'"$STAFF_ID"'","shop_id":"'"$SHOP_ID"'","shop_role_level":2}'

# Yenidən login, indi uğurlu olur:
curl -X POST http://localhost:8087/api/v1/products \
  -H "Authorization: Bearer $STAFF_TOKEN" -H "Content-Type: application/json" \
  -d '{"shop_id":"'"$SHOP_ID"'","name":"Stul","price":50,"stock":20}'
# → 201 Created
```

Sahib yalnız **öz** mağazasına əməkdaş təyin edə bilər — başqa mağazaya cəhd `403` alır (sistem administratoru istisnadır, o istənilən mağazaya təyin edə bilər).

### 10. Yalnız admin(4) mağazanı silə bilər — add-product(2) səviyyəsi kifayət etmir

```bash
curl -X DELETE "http://localhost:8086/api/v1/shops/$SHOP_ID" \
  -H "Authorization: Bearer $STAFF_TOKEN"
# → 403 {"success":false,"error":{"code":"forbidden","message":"admin(4) level in this shop, or administrator, required"}}
```

### 11. Public siyahılar (auth tələb olunmur, istənilən vaxt)

```bash
curl http://localhost:8086/api/v1/shops
curl http://localhost:8087/api/v1/products
curl "http://localhost:8087/api/v1/products?shop_id=$SHOP_ID"
```

### 12. Birbaşa mağaza yaratma yalnız administrator üçündür

```bash
curl -X POST http://localhost:8086/api/v1/shops \
  -H "Authorization: Bearer $APPLICANT_TOKEN" -H "Content-Type: application/json" \
  -d '{"name":"Qanunsuz mağaza"}'
# → 403 {"success":false,"error":{"code":"forbidden","message":"administrator role required"}}
```

### 13. Rədd olunan müraciət (form_sent mərhələsində, səbəblə)

```bash
# Başqa müraciətçi, form göndərilir, sonra rədd edilir:
curl -X POST "http://localhost:8086/api/v1/shop-applications/$OTHER_APP_ID/reject" \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "Content-Type: application/json" \
  -d '{"reason":"Vergi qeydiyyat sənədi çatışmır"}'
# → müvəqqəti mağaza silinir, müraciətçinin shop_id/shop_role_level sıfırlanır,
#   notification-service konsoluna mock bildiriş yazılır: type=shop_application_rejected, message=səbəb
```

### 14. Administrator hər şeyi idarə edə bilər (səviyyədən asılı olmayaraq)

```bash
curl -X PUT "http://localhost:8086/api/v1/shops/$SHOP_ID" \
  -H "Authorization: Bearer $ADMIN_TOKEN" -H "Content-Type: application/json" \
  -d '{"name":"Owner Shop (moderated)","description":"..."}'
# → 200 OK
```

## Servislərin siyahısı

| Servis | Qovluq | Port | README |
|---|---|---|---|
| Qeydiyyat/login | [registration-service](registration-service) | 8081 | [README](registration-service/README.md) |
| Profil | [user-service](user-service) | 8082 | [README](user-service/README.md) |
| Bildiriş (mock) | [notification-service](notification-service) | 8083 | [README](notification-service/README.md) |
| Token doğrulama | [authorization-service](authorization-service) | 8084 | [README](authorization-service/README.md) |
| Mağaza səviyyələri | [shop-role-service](shop-role-service) | 8085 | [README](shop-role-service/README.md) |
| Mağaza CRUD + müraciət | [shop-service](shop-service) | 8086 | [README](shop-service/README.md) |
| Məhsul CRUD | [shop-product-service](shop-product-service) | 8087 | [README](shop-product-service/README.md) |
