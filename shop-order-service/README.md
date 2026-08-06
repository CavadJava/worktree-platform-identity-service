# shop-order-service

İstifadəçilərin mağazalardan məhsul **variantı** (product item) seçib sifariş yaratması. `orders` və `order_items` cədvəllərinin sxem sahibidir.

## Sifariş nömrəsi

Format: `"U<user_seq>-S<shop_seq>-<n>"` — məsələn `"U27-S6-2"` = istifadəçi #27-nin mağaza #6-dan 2-ci sifarişi.

- `user_seq` — [registration-service](../registration-service)-in `users.user_seq` sütunu (`BIGSERIAL`, qeydiyyat anında avtomatik).
- `shop_seq` — [shop-service](../shop-service)-in `shops.shop_seq` sütunu (`BIGSERIAL`, mağaza yaranma anında avtomatik).
- `<n>` — bu istifadəçinin (bütün mağazalar üzrə) əvvəlki sifarişlərinin sayı + 1.

shop-order-service bu iki sütunu, həmçinin [shop-product-service](../shop-product-service)-in `products`/`product_items` cədvəllərini **birbaşa, read-only** eyni Postgres instansiyasından oxuyur — əlavə HTTP round-trip yoxdur, digər servislərin artıq istifadə etdiyi "ortaq DB, konkret sütun üçün nəzarətli cross-read" konvensiyasını izləyir.

## Qaydalar

- Sifariş **variant (`product_item_id`) səviyyəsindədir** — məs. "Bayraq" məhsulunun "30x60 1 qat" variantı sifariş edilir, təkcə "Bayraq" yox. Bir sifarişin bütün sətirləri **eyni mağazaya** aid olmalıdır (variantın öz məhsulu vasitəsilə) — qarışıq-mağaza sifarişi yoxdur.
- Qiymət server tərəfdə həll olunur, client-dən **etibar edilmir**: variant `is_discounted=true` isə `discount_price`, əks halda `price` istifadə olunur.
- Hər sətirdə məhsulun/variantın cari adı və həll olunmuş qiyməti sifariş yaradılan anda **"şəkil" kimi saxlanılır** (`order_items.product_name`/`item_name`/`unit_price`) — mağaza sonradan qiyməti/endirimi dəyişsə belə, artıq yaranmış sifariş dəyişmir.
- Kim baxa bilər: sifarişi verən istifadəçi, mağazanın **add-product(2)+** səviyyəli əməkdaşı, ya da administrator.

## Ödəniş

Sifariş uğurla yaranan kimi [payment-service](../payment-service)-ə `POST /payments` göndərilir (sinxron, amma uğursuz olsa sifarişi pozmur — sadəcə loglanır) — "müştəri ödəniş edir" anının qarşılığı. Bu, mağazanın müvəqqəti bakiyəsini artırır; mağaza öz ödənişlərini payment-service üzərindən izləyir.

## Stack
- Go 1.26 + chi router
- PostgreSQL (`pgx`) — `orders`, `order_items` cədvəllərinin sahibi + `users`/`shops`/`products`/`product_items`-ı oxuyur

## Setup

```bash
cp .env.example .env
go mod tidy
go run ./cmd/api
```

Default port: **8089**. Swagger UI: http://localhost:8089/swagger/index.html

## Endpoints

| Method | Path                              | Auth                                        | Body |
|--------|-------------------------------------|------------------------------------------------|------|
| GET    | /health                           | -                                                 | - |
| POST   | /api/v1/orders                    | Bearer (istənilən login istifadəçi)              | `{shop_id, items:[{product_item_id, quantity}]}` |
| GET    | /api/v1/orders                    | Bearer (istənilən login istifadəçi)              | - (öz sifarişləri) |
| GET    | /api/v1/orders/{id}                | Bearer (sifarişi verən / mağazanın add-product(2)+ / admin) | - |
| GET    | /api/v1/shops/{shop_id}/orders     | Bearer (mağazanın add-product(2)+ əməkdaşı / admin) | - |

Bax [../ARCHITECTURE.md](../ARCHITECTURE.md) tam servislərarası axın üçün.
