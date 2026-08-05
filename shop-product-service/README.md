# shop-product-service

Mağaza məhsullarının CRUD idarəetməsi. `products`, `product_items`, `product_favorites`, `product_types` və `product_subtypes` cədvəllərinin sxem sahibidir.

İcazə yoxlaması **tamamilə JWT-dən** qərarlaşdırılır — cross-service çağırış yoxdur. Cari istifadəçinin tokenindəki `shop_id`/`shop_role_level` hədəf məhsulun `shop_id`-si ilə müqayisə olunur:

- **Yaratmaq/yeniləmək**: `add-product(2)`+ səviyyəli mağaza əməkdaşı və ya administrator.
- **Silmək**: `review(3)`+ səviyyəli mağaza əməkdaşı və ya administrator.
- **Oxumaq/siyahı**: public.

Hierarxiya: `admin(4) > review(3) > add-product(2) > chat(1)` — yüksək səviyyə aşağı səviyyələrin bütün icazələrini əhatə edir.

## Məhsul itemləri (variantlar)

Bir `Product` (məs. "Bayraq") özünün müxtəlif ölçü/say **variantlarına** bölünə bilər — məs. "30x60 1 qat" (15 AZN, 200 stok) və "100x100 2 qat" (45 AZN, endirimli 35 AZN, 50 stok). Hər item öz `price`/`stock`/`is_discounted`/`discount_price`-ına sahibdir; `Product`-un özündəki `price`/`stock` sadə (variantsız) məhsullar üçün qalır.

- `is_discounted=true` olduqda `discount_price` **məcburidir** və `price`-dan **kiçik olmalıdır** — əks halda `400 bad_request`.
- Yaratmaq/yeniləmək: `add-product(2)`+ səviyyəli mağaza əməkdaşı (məhsulun aid olduğu mağazada) və ya administrator.
- Silmək: `review(3)`+ səviyyəli mağaza əməkdaşı və ya administrator.
- Oxumaq (siyahı/tək item): public.

## Məhsul növləri / alt növləri

Mağaza sahibi öz mağazası üçün **növ** (məs. "Ölçü") və hər növün altında **alt növlər** (məs. "En", "Uzunluq") təyin edə bilər. `ProductType` mağaza-scoped-dir (`shop_id`); `ProductSubtype` valideyn `ProductType`-a bağlıdır (`ON DELETE CASCADE`). `Product`-un opsional `product_type_id` sahəsi ilə hər hansı bir məhsul bir növə etiketlənə bilər — məhsul yaradılanda/yenilənəndə göstərilən `product_type_id` mütləq **eyni mağazaya** aid olmalıdır, əks halda `400 bad_request`.

- Yaratmaq/yeniləmək (növ və alt növ): `add-product(2)`+ səviyyəli mağaza əməkdaşı (öz mağazasında) və ya administrator.
- Silmək (növ və alt növ): `review(3)`+ səviyyəli mağaza əməkdaşı və ya administrator.
- Oxumaq (siyahı/tək): public.

## Favoritlər

İstənilən login olmuş istifadəçi bir məhsulu favoritə əlavə edə bilər (sahiblik/səviyyə tələb olunmur) — `product_favorites (user_id, product_id)`, idempotent (`ON CONFLICT DO NOTHING`).

## Stack
- Go 1.26 + chi router
- PostgreSQL (`pgx`) — `products`, `product_items`, `product_favorites`, `product_types`, `product_subtypes` cədvəllərinin sahibi

## Setup

```bash
cp .env.example .env
go mod tidy
go run ./cmd/api
```

Default port: **8087**. Swagger UI: http://localhost:8087/swagger/index.html

## Endpoints

| Method | Path                 | Auth                                     | Body |
|--------|----------------------|--------------------------------------------|------|
| GET    | /health              | -                                            | - |
| POST   | /api/v1/products     | Bearer (add-product(2)+ / administrator)    | `{shop_id, name, description, price, stock, product_type_id?}` |
| GET    | /api/v1/products     | -                                             | - (opt. `?shop_id=`) |
| GET    | /api/v1/products/{id}| -                                             | - |
| PUT    | /api/v1/products/{id}| Bearer (add-product(2)+ / administrator)    | `{name, description, price, stock, product_type_id?}` |
| DELETE | /api/v1/products/{id}| Bearer (review(3)+ / administrator)         | - |
| POST   | /api/v1/products/{id}/favorite | Bearer (istənilən login istifadəçi) | - |
| DELETE | /api/v1/products/{id}/favorite | Bearer (istənilən login istifadəçi) | - |
| GET    | /api/v1/favorites     | Bearer (istənilən login istifadəçi)          | - (öz favoritləri) |
| POST   | /api/v1/products/{product_id}/items | Bearer (add-product(2)+ / administrator) | `{name, price, stock, is_discounted, discount_price}` |
| GET    | /api/v1/products/{product_id}/items | -                                  | - (məhsulun bütün variantları) |
| GET    | /api/v1/product-items/{id}          | -                                  | - |
| PUT    | /api/v1/product-items/{id}          | Bearer (add-product(2)+ / administrator) | `{name, price, stock, is_discounted, discount_price}` |
| DELETE | /api/v1/product-items/{id}          | Bearer (review(3)+ / administrator) | - |
| POST   | /api/v1/product-types                    | Bearer (add-product(2)+ / administrator) | `{shop_id, name}` |
| GET    | /api/v1/product-types                    | -                                          | - (`?shop_id=` tələb olunur) |
| GET    | /api/v1/product-types/{id}               | -                                          | - |
| PUT    | /api/v1/product-types/{id}               | Bearer (add-product(2)+ / administrator) | `{name}` |
| DELETE | /api/v1/product-types/{id}               | Bearer (review(3)+ / administrator)      | - |
| POST   | /api/v1/product-types/{type_id}/subtypes | Bearer (add-product(2)+ / administrator) | `{name}` |
| GET    | /api/v1/product-types/{type_id}/subtypes | -                                          | - |
| PUT    | /api/v1/product-subtypes/{id}            | Bearer (add-product(2)+ / administrator) | `{name}` |
| DELETE | /api/v1/product-subtypes/{id}            | Bearer (review(3)+ / administrator)      | - |

Bax [../ARCHITECTURE.md](../ARCHITECTURE.md) tam servislərarası axın üçün.
