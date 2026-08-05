# shop-chat-service

İstifadəçi ilə mağaza arasında yazışma. `conversations` və `messages` cədvəllərinin sxem sahibidir.

Hər `(shop_id, user_id)` cütü üçün **bir** söhbət mövcuddur — `POST /conversations {shop_id}` mövcud olanı tapıb qaytarır, yoxdursa yaradır. Bu, `chat(1)` hierarxiya səviyyəsinin ilk konkret funksional istifadəsidir (əvvəllər yalnız modeldə nəzərdə tutulmuşdu).

## Kim yaza/oxuya bilər

- Söhbətin sahibi olan **müştəri** (`conversation.user_id == caller.user_id`).
- Mağazanın **chat(1)+** səviyyəli əməkdaşı (`caller.shop_id == conversation.shop_id && caller.shop_role_level >= 1`).
- Sistem **administratoru** (bypass).

Mesaj göndərən tərəf avtomatik müəyyən olunur: göndərən şəxs söhbətin `user_id`-si ilə eynidirsə `sender_role=user`, əks halda `sender_role=shop`.

## Stack
- Go 1.26 + chi router
- PostgreSQL (`pgx`) — `conversations`, `messages` cədvəllərinin sahibi

## Setup

```bash
cp .env.example .env
go mod tidy
go run ./cmd/api
```

Default port: **8088**. Swagger UI: http://localhost:8088/swagger/index.html

## Endpoints

| Method | Path                                   | Auth                                        | Body |
|--------|------------------------------------------|------------------------------------------------|------|
| GET    | /health                                 | -                                                 | - |
| POST   | /api/v1/conversations                   | Bearer (istənilən login istifadəçi)              | `{shop_id}` |
| GET    | /api/v1/conversations                   | Bearer (istənilən login istifadəçi)              | - (öz söhbətləri, müştəri kimi) |
| GET    | /api/v1/shops/{shop_id}/conversations   | Bearer (mağazanın chat(1)+ əməkdaşı / admin)    | - |
| GET    | /api/v1/conversations/{id}/messages     | Bearer (iştirakçı / admin)                       | - |
| POST   | /api/v1/conversations/{id}/messages     | Bearer (iştirakçı / admin)                       | `{body}` |

Bax [../ARCHITECTURE.md](../ARCHITECTURE.md) tam servislərarası axın üçün.
