# payment-service

Sifariş ödənişlərinin qeydiyyatı və mağazaların **müvəqqəti bakiyəsi**. `payments`, `payment_items` və `shop_balances` cədvəllərinin sxem sahibidir.

[shop-order-service](../shop-order-service) hər sifariş yaradılandan **dərhal sonra** bura `POST /payments` göndərir — bu, "müştəri ödəniş edir" anının mock qarşılığıdır (real ödəniş gateway-i yoxdur, elə buna görə `status` həmişə birbaşa `completed`-dir, real gateway-in `pending`/uğursuz vəziyyətləri simulyasiya olunmur). Hər ödəniş öz mağazasının bakiyəsini avtomatik artırır — bir tranzaksiya daxilində (ödəniş + sətirlər + bakiyə yenilənməsi), bakiyə heç vaxt ödənişlərin cəmindən "yayına" bilməz.

## Müvəqqəti mağaza hesabı

`shop_balances (shop_id, balance, updated_at)` — hər mağazanın yığılan bakiyəsi. Bu, **hələ ki, yalnız payment-service daxilində yaşayan bir dəftərdir** — real bank hesabına köçürmə (payout) gələcək bir funksiyadır və bu sxem onu bloklamır: bir payout sadəcə `balance`-dan çıxıla, öz ledger sətrini yaza bilərdi (hələ tikilməyib).

## Ödəniş = sifariş (1:1)

Bir `Payment` bir `Order`-ə uyğundur (`order_id` üzərində unikal indeks) — shop-order-service-in sifarişləri həmişə tək-mağaza olduğu üçün ödəniş də təbii olaraq tək-mağazadır. Hər `PaymentItem` [shop-order-service](../shop-order-service)-in `OrderItem`-i kimi məhsul adını/sayını/qiymətini **sifariş anında saxlayır** (snapshot) — mağaza tərəfinin ödəniş tarixçəsi baxışı üçün əlavə cross-service çağırışa ehtiyac yoxdur.

## İcazə

- `POST /payments` — **auth yoxdur**, daxili servis-servis çağırışıdır (notification-service-in `POST /notifications`-ı ilə eyni konvensiya — bu, real istifadəçi tokenindən imzalanan bir şey deyil).
- `GET /shops/{id}/payments`, `GET /shops/{id}/balance` — mağazanın **chat(1)+** səviyyəli əməkdaşı (bütün komanda, təkcə sahib yox — review-service-in rəy müəllifləri və shop-service-in abunəçi siyahısı ilə eyni əsaslandırma) və ya administrator.

## Sifariş yaradılması uğursuz ödənişdən asılı deyil

shop-order-service ödənişi **sinxron** çağırır (goroutine-də deyil) — mağaza/müştəri sifariş yaradılan andaca ödənişin qeydə alındığını görsün deyə. Amma uğursuzluq **fatal deyil**: payment-service əlçatmaz olsa, sifariş yenə uğurla yaranır, xəta sadəcə loglanır — "ikinci dərəcəli narahatlıq heç vaxt əsas yazını pozmamalıdır" prinsipi, bu kodbazada bildiriş fan-out-unda da izlənilir (bax [shop-product-service](../shop-product-service)).

## Stack
- Go 1.26 + chi router
- PostgreSQL (`pgx`) — `payments`, `payment_items`, `shop_balances` cədvəllərinin sahibi

## Setup

```bash
cp .env.example .env
go mod tidy
go run ./cmd/api
```

Default port: **8094**. Swagger UI: http://localhost:8094/swagger/index.html

## Endpoints

| Method | Path                        | Auth                       | Body / Qeyd |
|--------|-----------------------------|-----------------------------|------|
| GET    | /health                     | -                            | - |
| POST   | /api/v1/payments            | -  (daxili çağırış)          | `{order_id, user_id, shop_id, amount, items:[{product_id, product_name, quantity, unit_price}]}` → 201, təkrar `order_id` üçün 409 |
| GET    | /api/v1/shops/{id}/payments | Bearer (chat(1)+ / admin)    | - (ən yenidən köhnəyə, hər ödənişin sətirləri ilə) |
| GET    | /api/v1/shops/{id}/balance  | Bearer (chat(1)+ / admin)    | - (heç ödəniş yoxdursa `{balance: 0}`) |

Bax [../ARCHITECTURE.md](../ARCHITECTURE.md) tam servislərarası axın üçün.
