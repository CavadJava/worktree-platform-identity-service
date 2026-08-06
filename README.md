# go-project-practices

14 ayrı Go mikroservisi: qeydiyyat/login, profil, bildiriş, autorizasiya, mağaza rolları, mağaza CRUD + müraciət axını + abunəlik, məhsul CRUD + favoritlər, istifadəçi↔mağaza yazışması, sifarişlər, çoxdilli mətnlər, mərkəzi loglama, kataloq (kateqoriyalar), rəylər (reyting + şəkil/video), ödənişlər + mağaza bakiyəsi. Arxitektura və servislərin bir-biri ilə əlaqəsi üçün bax [ARCHITECTURE.md](ARCHITECTURE.md).

## Tələblər

- Go 1.26+
- PostgreSQL, `localhost:5433`, `postgres` bazası, şifrə `.env` fayllarında təyin olunub (default `1`)
- `psql` (opsional, ilk administratoru təyin etmək üçün) — `brew install libpq && export PATH="/opt/homebrew/opt/libpq/bin:$PATH"`

## Servisləri işə salmaq

Hər servisin öz `.env`-i var (repo-da hazır, lokal inkişaf üçün). 14 ayrı terminalda:

```bash
cd notification-service   && go run ./cmd/api   # :8083
cd authorization-service  && go run ./cmd/api   # :8084
cd registration-service   && go run ./cmd/api   # :8081
cd user-service           && go run ./cmd/api   # :8082
cd shop-role-service      && go run ./cmd/api   # :8085
cd shop-service           && go run ./cmd/api   # :8086
cd shop-product-service   && go run ./cmd/api   # :8087
cd shop-chat-service      && go run ./cmd/api   # :8088
cd shop-order-service     && go run ./cmd/api   # :8089
cd localization-service   && go run ./cmd/api   # :8090
cd log-service            && go run ./cmd/api   # :8091
cd shop-category-service  && go run ./cmd/api   # :8092
cd review-service         && go run ./cmd/api   # :8093
cd payment-service        && go run ./cmd/api   # :8094
```

Sıra fərq etmir, amma tam axın üçün hamısı ayaqda olmalıdır. Yoxlamaq:

```bash
for p in 8081 8082 8083 8084 8085 8086 8087 8088 8089 8090 8091 8092 8093 8094; do curl -s http://localhost:$p/health; echo " :$p"; done
```

Hər servisin Swagger UI-ı: `http://localhost:<port>/swagger/index.html`

## Seed skriptləri

`scripts/seed-taobao-marketplace.sh` — `taobao-v1` mobil tətbiqinin checkout/chat axını üçün lazım olan `"Taobao Marketplace"` mağazasını və onun 10 mock məhsulunu (`taobao-v1/src/data/products.ts`-i güzgüləyir) yaradır. İdempotentdir — nə çatışmırsa yalnız onu əlavə edir, təkrar işə salmaq təhlükəsizdir:

```bash
./scripts/seed-taobao-marketplace.sh
```

Tələb edir: `registration-service`, `shop-service`, `shop-product-service` ayaqda olsun + bir administrator hesabı (default `admin@example.com`/`password123`, `ADMIN_EMAIL`/`ADMIN_PASSWORD` env ilə override edilə bilər).

## CORS

Bütün servislər default olaraq `http://localhost:5173`-dən (Vite dev server) gələn brauzer sorğularına icazə verir — `CORS_ALLOWED_ORIGINS` env dəyişəni ilə (vergüllə ayrılmış siyahı) hər servisdə fərdi tənzimlənə bilər. Frontend inkişafı zamanı əlavə CORS konfiqurasiyası tələb olunmur.

## Mərkəzi loglama

Bütün servislər hər HTTP sorğusunu (method, path, status, müddət) avtomatik [log-service](log-service)-ə (:8091) göndərir — fire-and-forget, əsas sorğunu heç vaxt bloklamır. Servisləri bir yerdən izləmək:

```bash
# Son loglar (bütün servislər):
curl "http://localhost:8091/api/v1/logs"

# Bir servisin errorları:
curl "http://localhost:8091/api/v1/logs?service=shop-order-service&level=error"
```

`5xx → error`, `4xx → warn`, qalanı `info`. `LOG_SERVICE_URL` env dəyişəni ilə tənzimlənir.

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

### 8a. Məhsula variantlar (itemlər) əlavə etmək

Hər variantın öz qiyməti/stoku/endirimi var — məs. "Bayraq" məhsuluna fərqli ölçü/qat variantları:

```bash
curl -X POST "http://localhost:8087/api/v1/products/$PRODUCT_ID/items" \
  -H "Authorization: Bearer $APPLICANT_TOKEN" -H "Content-Type: application/json" \
  -d '{"name":"30x60 1 qat","price":15,"stock":200,"is_discounted":false}'

curl -X POST "http://localhost:8087/api/v1/products/$PRODUCT_ID/items" \
  -H "Authorization: Bearer $APPLICANT_TOKEN" -H "Content-Type: application/json" \
  -d '{"name":"100x100 2 qat","price":45,"stock":50,"is_discounted":true,"discount_price":35}'
# is_discounted=true olduqda discount_price məcburidir və price-dan kiçik olmalıdır — əks halda 400

curl "http://localhost:8087/api/v1/products/$PRODUCT_ID/items"
```

### 8b. Məhsul növü/alt növü təyin etmək

Mağaza sahibi öz kataloqu üçün taksonomiya qurur — məs. "Ölçü" növü, "En"/"Uzunluq" alt növləri:

```bash
TYPE_ID=$(curl -s -X POST http://localhost:8087/api/v1/product-types \
  -H "Authorization: Bearer $APPLICANT_TOKEN" -H "Content-Type: application/json" \
  -d '{"shop_id":"'"$SHOP_ID"'","name":"Ölçü"}' | python3 -c "import sys,json;print(json.load(sys.stdin)['data']['id'])")

curl -X POST "http://localhost:8087/api/v1/product-types/$TYPE_ID/subtypes" \
  -H "Authorization: Bearer $APPLICANT_TOKEN" -H "Content-Type: application/json" \
  -d '{"name":"En"}'

curl -X POST "http://localhost:8087/api/v1/product-types/$TYPE_ID/subtypes" \
  -H "Authorization: Bearer $APPLICANT_TOKEN" -H "Content-Type: application/json" \
  -d '{"name":"Uzunluq"}'

# Məhsulu bu növə etiketləmək (product_type_id başqa mağazaya aiddirsə 400 qaytarır):
curl -X POST http://localhost:8087/api/v1/products \
  -H "Authorization: Bearer $APPLICANT_TOKEN" -H "Content-Type: application/json" \
  -d '{"shop_id":"'"$SHOP_ID"'","name":"Bayraq","price":10,"stock":5,"product_type_id":"'"$TYPE_ID"'"}'
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

### 15. Favoritlər, abunəlik, yazışma (istənilən login istifadəçi)

```bash
# Məhsulu favoritə əlavə et / sil / siyahıla:
curl -X POST "http://localhost:8087/api/v1/products/$PRODUCT_ID/favorite" -H "Authorization: Bearer $CUSTOMER_TOKEN"
curl -X DELETE "http://localhost:8087/api/v1/products/$PRODUCT_ID/favorite" -H "Authorization: Bearer $CUSTOMER_TOKEN"
curl "http://localhost:8087/api/v1/favorites" -H "Authorization: Bearer $CUSTOMER_TOKEN"

# Mağazaya abunə ol / abunəlikdən çıx / siyahıla:
curl -X POST "http://localhost:8086/api/v1/shops/$SHOP_ID/subscribe" -H "Authorization: Bearer $CUSTOMER_TOKEN"
curl -X DELETE "http://localhost:8086/api/v1/shops/$SHOP_ID/subscribe" -H "Authorization: Bearer $CUSTOMER_TOKEN"
curl "http://localhost:8086/api/v1/subscriptions" -H "Authorization: Bearer $CUSTOMER_TOKEN"

# Mağaza ilə yazışma (eyni müştəri+mağaza cütü üçün bir söhbət):
CONV_ID=$(curl -s -X POST http://localhost:8088/api/v1/conversations \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" -H "Content-Type: application/json" \
  -d '{"shop_id":"'"$SHOP_ID"'"}' | python3 -c "import sys,json;print(json.load(sys.stdin)['data']['id'])")

curl -X POST "http://localhost:8088/api/v1/conversations/$CONV_ID/messages" \
  -H "Authorization: Bearer $CUSTOMER_TOKEN" -H "Content-Type: application/json" \
  -d '{"body":"Salam, bu məhsul mövcuddurmu?"}'

# Mağazanın chat(1)+ əməkdaşı (və ya sahibi) öz mağazasının söhbətlərinə baxır və cavab yazır:
curl "http://localhost:8088/api/v1/shops/$SHOP_ID/conversations" -H "Authorization: Bearer $STAFF_TOKEN"
curl -X POST "http://localhost:8088/api/v1/conversations/$CONV_ID/messages" \
  -H "Authorization: Bearer $STAFF_TOKEN" -H "Content-Type: application/json" \
  -d '{"body":"Bəli, mövcuddur!"}'

curl "http://localhost:8088/api/v1/conversations/$CONV_ID/messages" -H "Authorization: Bearer $CUSTOMER_TOKEN"
```

Kənar şəxs (nə söhbətin müştərisi, nə mağazanın chat(1)+ əməkdaşı) `403` alır.

### 16. Sifariş yaratma (unikal nömrələr, `order_number`)

Hər istifadəçinin (`user_seq`) və mağazanın (`shop_seq`) qeydiyyat/yaranma anında avtomatik ardıcıl nömrəsi var — `GET /users`, `GET /shops/{id}` cavablarında görünür. Sifariş nömrəsi bunlardan qurulur: `"U<user_seq>-S<shop_seq>-<n>"`.

```bash
# Mağaza sahibi "Bayraq" məhsulunu və onun variantlarını (item) yaradır:
PRODUCT_ID=$(curl -s -X POST http://localhost:8087/api/v1/products -H "Authorization: Bearer $OWNER_TOKEN" -H "Content-Type: application/json" -d '{"shop_id":"'"$SHOP_ID"'","name":"Bayraq"}' | python3 -c "import sys,json;print(json.load(sys.stdin)['data']['id'])")
ITEM1=$(curl -s -X POST "http://localhost:8087/api/v1/products/$PRODUCT_ID/items" -H "Authorization: Bearer $OWNER_TOKEN" -H "Content-Type: application/json" -d '{"name":"30x60 1 qat","price":15,"stock":200,"is_discounted":false}' | python3 -c "import sys,json;print(json.load(sys.stdin)['data']['id'])")
ITEM2=$(curl -s -X POST "http://localhost:8087/api/v1/products/$PRODUCT_ID/items" -H "Authorization: Bearer $OWNER_TOKEN" -H "Content-Type: application/json" -d '{"name":"100x100 2 qat","price":45,"stock":50,"is_discounted":true,"discount_price":35}' | python3 -c "import sys,json;print(json.load(sys.stdin)['data']['id'])")

# Customer bir mağazadan bir neçə VARİANT seçib sifariş yaradır (product_id yox, product_item_id):
curl -X POST http://localhost:8089/api/v1/orders -H "Authorization: Bearer $CUSTOMER_TOKEN" -H "Content-Type: application/json" \
  -d '{"shop_id":"'"$SHOP_ID"'","items":[{"product_item_id":"'"$ITEM1"'","quantity":3},{"product_item_id":"'"$ITEM2"'","quantity":2}]}'
# → {"success":true,"data":{"order_number":"U27-S6-1","total_amount":115,"items":[
#     {"product_name":"Bayraq","item_name":"30x60 1 qat","unit_price":15,"quantity":3},
#     {"product_name":"Bayraq","item_name":"100x100 2 qat","unit_price":35,"quantity":2}  ← discount_price avtomatik tətbiq olundu (45 yox, 35)
#   ],"status":"pending",...}}
# Hər sətirdə "item_name"/"unit_price" sifariş anındakı "şəkildir" — mağaza sonra qiyməti/endirimi dəyişsə belə bu sifariş dəyişmir.

# Customer öz sifarişlərinə baxır:
curl "http://localhost:8089/api/v1/orders" -H "Authorization: Bearer $CUSTOMER_TOKEN"

# Mağazanın add-product(2)+ əməkdaşı (və ya sahibi) öz mağazasının gələn sifarişlərinə baxır:
curl "http://localhost:8089/api/v1/shops/$SHOP_ID/orders" -H "Authorization: Bearer $OWNER_TOKEN"

# Kənar şəxs bu sifarişə baxa bilmir:
curl -w "\nHTTP:%{http_code}\n" "http://localhost:8089/api/v1/orders/$ORDER_ID" -H "Authorization: Bearer $STRANGER_TOKEN"
# → 403
```

### 17. Çoxdilli mətnlər (localization-service)

```bash
# errors namespace-i, az dilində, bütün açarlar (key→value map):
curl "http://localhost:8090/api/v1/translations/map?namespace=errors&locale=az"

# Tək açar, fallback ilə (sorğulanan locale yoxdursa DEFAULT_LOCALE-a keçir):
curl "http://localhost:8090/api/v1/translations/lookup?namespace=errors&key=forbidden&locale=ru"

# Admin yeni tərcümə əlavə edir/yeniləyir (upsert):
curl -X POST http://localhost:8090/api/v1/translations -H "Authorization: Bearer $ADMIN_TOKEN" -H "Content-Type: application/json" \
  -d '{"namespace":"common","key":"welcome","locale":"az","value":"Xoş gəldiniz"}'

# Mövcud locale-ların siyahısı:
curl "http://localhost:8090/api/v1/locales"
# → ["az","en","ru"]
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
| Məhsul CRUD + favoritlər | [shop-product-service](shop-product-service) | 8087 | [README](shop-product-service/README.md) |
| İstifadəçi↔mağaza yazışması | [shop-chat-service](shop-chat-service) | 8088 | [README](shop-chat-service/README.md) |
| Sifarişlər | [shop-order-service](shop-order-service) | 8089 | [README](shop-order-service/README.md) |
| Çoxdilli mətnlər | [localization-service](localization-service) | 8090 | [README](localization-service/README.md) |
| Mərkəzi loglama | [log-service](log-service) | 8091 | [README](log-service/README.md) |
| Kataloq (kateqoriyalar) | [shop-category-service](shop-category-service) | 8092 | [README](shop-category-service/README.md) |
| Rəylər (reyting + şəkil/video) | [review-service](review-service) | 8093 | [README](review-service/README.md) |
| Ödənişlər + mağaza bakiyəsi | [payment-service](payment-service) | 8094 | [README](payment-service/README.md) |
