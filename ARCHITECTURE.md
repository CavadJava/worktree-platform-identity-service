# Arxitektura

8 ayrı Go mikroservisi, ortaq bir PostgreSQL instansiyasına (`localhost:5433`) qoşulur.

```
 registration-service :8081 ──POST /notifications──▶ notification-service :8083
  (register, login,                                   (mock, log-only)
   owns `users` table,
   JWT yaradır: role +
   shop_id + shop_role_level)
        │
        │ writes
        ▼
  ┌───────────────┐        reads/writes        ┌─────────────────────────┐
  │   PostgreSQL   │◀────────────────────────────│  user-service    :8082   │
  │  `users` table │                             │  (profile GET/PUT)       │
  └───────┬────────┘                             └────────────┬─────────────┘
          │ reads/writes                                      │
          │ (shop_id, shop_role_level)                        │ POST /authorize
          ▼                                                    ▼
  ┌─────────────────────┐                          ┌─────────────────────────┐
  │ shop-role-service     │──── POST /authorize ───▶│  authorization-service    │
  │        :8085          │   (admin-only)          │         :8084             │
  │ (assign/revoke a       │                          │  stateless, JWT_SECRET    │
  │  shop level 1-4)        │                          │  ilə token doğrulayır     │
  └─────────────────────┘                          └─────────────────────────┘
                                                                 ▲
  ┌─────────────────────┐   POST /authorize                     │
  │  shop-service          │───────────────────────────────────┘
  │        :8086          │
  │ (shops CRUD +           │──── writes users.shop_id/level ──▶ (on application approval)
  │  shop-applications)      │
  └─────────────────────┘
                        ▲
  ┌─────────────────────┐   POST /authorize
  │  shop-product-service │───────────────────────────────────▶ (same authorization-service)
  │        :8087          │
  │ (products CRUD,        │   permission decided from the caller's own
  │  owns `products` table) │   JWT shop_id/shop_role_level — no cross-service call needed
  └─────────────────────┘
```

## Servislər

| Servis                  | Port | Vəzifə                                             | DB                          | Asılılıqlar |
|--------------------------|------|-----------------------------------------------------|------------------------------|-------------|
| registration-service     | 8081 | qeydiyyat, login, JWT **yaratma** (authentication)   | var, `users` sxeminin sahibi | notification-service |
| user-service              | 8082 | profil GET/PUT                                        | var, `users`-ı paylaşır       | authorization-service |
| notification-service      | 8083 | bildiriş (mock, log-only)                             | yox                            | - |
| authorization-service     | 8084 | JWT **doğrulama** (authorization), stateless          | yox                            | - |
| shop-role-service         | 8085 | mağaza səviyyəsi (1-4) assign/revoke (yalnız admin)   | var, `users.shop_id`/`shop_role_level`-i paylaşır | authorization-service |
| shop-service               | 8086 | mağaza CRUD + müraciət/təsdiq axını + abunəlik        | var, `shops`/`shop_applications`/`shop_subscriptions` sxemlərinin sahibi | authorization-service, notification-service |
| shop-product-service       | 8087 | məhsul CRUD + favoritlər                               | var, `products`/`product_favorites` sxemlərinin sahibi | authorization-service |
| shop-chat-service           | 8088 | istifadəçi ↔ mağaza yazışması                          | var, `conversations`/`messages` sxemlərinin sahibi | authorization-service |

## Niyə belə bölündü

- **Authentication vs authorization ayrıdır**: registration-service token yaradır (kimsən), authorization-service token doğrulayır (icazən varmı). `JWT_SECRET` yalnız bu ikisində var, digərlərində yoxdur.
- **Auth tələb edən servislər token doğrulamır özləri** — hər qorunan sorğuda authorization-service-ə HTTP ilə müraciət edir (`internal/client/authorization_client.go` hər servisdə eyni pattern). Bu, servisləri kriptoqrafik sirdən tam təcrid edir.
- **notification-service mock-dur** — real email/SMS provideri yoxdur, sadəcə konsola loglayır. registration-service bu çağırışın uğursuz olmasını qeydiyyatı bloklamaq üçün istifadə etmir.
- **Ortaq DB, ayrı sxem sahibliyi**: registration-service, user-service, shop-role-service eyni `users` cədvəlinə baxır, amma yalnız registration-service migrasiya aparır. shop-service tətbiq təsdiqi zamanı da bu cədvəlin iki sütununa (`shop_id`, `shop_role_level`) yazır — mağaza yaradılanda müraciət sahibini avtomatik o mağazanın admin(4) səviyyəsinə təyin etmək üçün. shop-service və shop-product-service isə öz `shops`/`shop_applications`/`products` cədvəllərinin tam sahibidir.
- **Mağaza icazəsi tamamilə JWT-dən qərarlaşdırılır**: istifadəçi bir mağazaya aiddirsə, bunu `shop_id`+`shop_role_level` claim-ləri daşıyır. shop-product-service artıq shop-service-ə cross-service sorğu göndərmir (əvvəlki versiyada belə idi) — sadəcə JWT-dəki `shop_id`-ni əməliyyatın hədəf mağazası ilə müqayisə edir. Bu, sistemi həm sadələşdirir, həm sürətləndirir.

## Rol modeli

Rollar DB-dən deyil, **JWT claim-ləri** kimi daşınır — authorization-service DB-yə getmədən, sadəcə tokeni decode edərək rolları qaytarır. Rol dəyişsə, istifadəçi yenidən login olmalıdır (token yenilənməlidir).

- `role`: `user` | `administrator` — hesabın əsas rolu. Hər qeydiyyat default `user` alır. `administrator` sistemin tam idarəçisidir: mağaza müraciətlərini təsdiq/rədd edir, istənilən mağazaya səviyyə təyin edir, bütün mağaza/məhsul əməliyyatlarında sahiblik yoxlaması bypass olunur.
- `shop_id` + `shop_role_level`: bir istifadəçi eyni anda **yalnız bir mağazaya** aid ola bilər, o mağazada **hierarxik səviyyə** daşıyır:

  ```
  admin(4) > review(3) > add-product(2) > chat(1)
  ```

  Yüksək səviyyə aşağı səviyyələrin bütün icazələrini əhatə edir (məs. `review`(3) `add-product`(2) və `chat`(1) tələb edən əməliyyatları da edə bilər). `shop_id=null, shop_role_level=0` — mağazaya aid deyil.

**Mağaza necə açılır (öz-özünə qeydiyyat YOXDUR, 2 mərhələli axın):**

1. İstənilən login olmuş istifadəçi `POST /shop-applications` ilə müraciət göndərir (ad + təsvir) — status `pending`. Heç bir mağaza hələ yaranmır.
2. Administrator `GET /shop-applications?status=pending` ilə müraciətlərə baxır və ya birbaşa **rədd** edə bilər (`.../reject`, mağaza heç yaranmadığı üçün heç nə silinmir), ya da **formu göndərir** (`POST .../send-form`).
3. **Form göndəriləndə**: müvəqqəti (`temporary=true`) mağaza yaranır, müraciət sahibi avtomatik onun **admin(4)** səviyyəli sahibi olur (`users.shop_id`/`shop_role_level` yenilənir), status `form_sent`-ə keçir, applicant-a mock bildiriş göndərilir. Müvəqqəti mağaza `GET /shops` (public siyahı) içində **görünmür** — yalnız birbaşa ID ilə (`GET /shops/{id}`) əlçatandır.
4. İstifadəçi yenidən login olur (yeni token `shop_id`+`admin(4)` daşıyır) və **formu özü doldurur** — sadəcə adi `PUT /shops/{id}` çağırıb mağaza məlumatlarını (ad, təsvir) yeniləyir. Ayrıca "form" API-si yoxdur — müvəqqəti mağazanın özü formdur.
5. Administrator yekun qərar verir:
   - **Approve**: mağaza `temporary=false` olur, public siyahıda görünməyə başlayır, status `approved`.
   - **Reject**: müvəqqəti mağaza silinir, müraciət sahibinin `shop_id`/`shop_role_level` sıfırlanır (`0`), status `rejected`, səbəb (`reason`) `review_note`-da saxlanılır **və** mock bildirişlə göndərilir.
6. Mağazanın admin(4) sahibi sonra **özü** (sistem administratoru olmadan) shop-role-service vasitəsilə öz mağazasına əməkdaş (chat/add-product/review, hətta başqa admin) təyin/geri ala bilər — bax aşağıda.

Hər müraciət `template_version` sahəsi daşıyır (default `"v1"`) — administrator gələcəkdə fərqli müraciət forması/qaydalar versiyaları üçün genişləndirə bilər.

**Birbaşa `POST /shops`** yalnız administrator üçündür (skip-the-line bootstrap) — adi istifadəçilər üçün bağlıdır, mütləq müraciət axınından keçməlidirlər.

**Mağaza daxilində rol idarəetməsi — kim kimə icazə verə bilər:**

- **Sistem administratoru** (`role=administrator`) istənilən istifadəçini istənilən mağazaya, istənilən səviyyə ilə təyin/geri ala bilər.
- **Mağazanın öz admin(4)-ü** — shop-role-service-in `POST /roles/assign` və `/revoke` endpoint-lərini eyni şəkildə çağıra bilər, AMMA yalnız **öz mağazası** üçün, və yalnız hazırda heç bir mağazaya aid olmayan (və ya artıq elə öz mağazasında olan) istifadəçiləri idarə edə bilər. Başqa mağazaya müdaxilə cəhdi `403 forbidden` ilə rədd olunur.

Bu, "sistem 4 səviyyəni müəyyən edir, mağaza sahibi isə kimin hansı səviyyədə işləyəcəyini özü qərarlaşdırır" prinsipini əks etdirir.

**İlk administratoru necə təyin etmək olar**: sistemdə özünü admin edən heç bir endpoint yoxdur (təhlükəsizlik üçün qəsdən belədir) — DB-dən birbaşa təyin etmək lazımdır (bax [README.md](README.md)).

## İstifadəçi-tərəfli funksiyalar: favoritlər, abunəlik, yazışma

Bunlar istənilən login olmuş istifadəçi üçün açıqdır (sahiblik/səviyyə tələb olunmur — sadəcə auth):

- **Məhsul favoritləri** ([shop-product-service](shop-product-service)): `POST/DELETE /products/{id}/favorite`, `GET /favorites` — istifadəçinin özü üçün. `product_favorites (user_id, product_id)` composite PK, `ON CONFLICT DO NOTHING` ilə idempotent.
- **Mağaza abunəliyi** ([shop-service](shop-service)): `POST/DELETE /shops/{id}/subscribe`, `GET /subscriptions`. `shop_subscriptions (user_id, shop_id)` eyni pattern.
- **İstifadəçi ↔ mağaza yazışması** ([shop-chat-service](shop-chat-service), yeni servis, :8088): hər `(shop_id, user_id)` cütü üçün **bir** söhbət mövcuddur (`UNIQUE (shop_id, user_id)`) — `POST /conversations {shop_id}` mövcud olanı tapıb qaytarır, yoxdursa yaradır. Mesaj göndərən tərəf avtomatik müəyyən olunur: `caller.UserID == conversation.UserID` olsa `sender_role=user`, əks halda `sender_role=shop`. Kim yaza/oxuya bilər: söhbətin sahibi olan müştəri, mağazanın **chat(1)+** səviyyəli əməkdaşı, və ya sistem administratoru — bu, `chat` (1) səviyyəsinin ilk konkret istifadəsidir (əvvəllər yalnız hierarxiyada nəzərdə tutulmuşdu, funksional qarşılığı yox idi).

## Cavab formatı (uğur/xəta)

Bütün 8 servisin bütün endpoint-ləri eyni JSON zərfini (envelope) qaytarır:

```json
// uğurlu:
{"success": true, "data": { ... }}

// xəta:
{"success": false, "error": {"code": "not_found", "message": "shop not found"}}
```

`error.code` HTTP statusundan avtomatik törəyir (`bad_request`, `unauthorized`, `forbidden`, `not_found`, `conflict`, `internal_error`, `service_unavailable`) — client tərəf mesajı parse etmədən, sabit koda görə budaqlana bilər. `/health` endpoint-ləri bu zərfə daxil deyil (sadə liveness probe).

## İşə salma ardıcıllığı

Hər servisin öz `.env`-i var (bax hər qovluqdakı `.env.example`). Sırası fərq etmir, amma tam funksionallıq üçün hamısı ayaqda olmalıdır. Tam addım-addım təlimat və nümunə API çağırışları üçün bax [README.md](README.md).

```bash
# 8 ayrı terminalda:
cd notification-service   && go run ./cmd/api   # :8083
cd authorization-service  && go run ./cmd/api   # :8084
cd registration-service   && go run ./cmd/api   # :8081
cd user-service           && go run ./cmd/api   # :8082
cd shop-role-service      && go run ./cmd/api   # :8085
cd shop-service           && go run ./cmd/api   # :8086
cd shop-product-service   && go run ./cmd/api   # :8087
cd shop-chat-service      && go run ./cmd/api   # :8088
```

Hər servisin öz Swagger UI-ı var: `http://localhost:<port>/swagger/index.html`
