# review-service

Məhsullara **reyting + mətn rəyi** yazmaq, üstünə **şəkil/video** əlavə etmək üçün REST API. `reviews` və `review_media` cədvəllərinin sxem sahibidir.

- Rəy istənilən authenticated istifadəçi tərəfindən yazıla bilər (rating 1-5, mətn opsional).
- Media faylı ayrı endpoint-lə, `multipart/form-data` ilə, artıq yaradılmış rəyə əlavə olunur — bir rəyin bir neçə şəkil/video-su ola bilər.
- Fayllar diskdə saxlanılır (`Config.UploadDir`, default `./uploads`), `review_media.url` isə ora göstərən **nisbi** yoldur (`/media/reviews/<review_id>/<fayl>`) — client öz HOST-unu özü qoşur, eynən API çağırışlarında olduğu kimi.
- `/media/*` JSON zərfindən kənar, xam fayl serving-idir (`/health`, `/swagger` kimi).
- Yalnız rəyin sahibi onu silə/media əlavə edə bilər (admin override yoxdur — bu, çox mürəkkəb icazə modelinə ehtiyac duymayan sadə "own-only" sxemdir, [user-service](../user-service)-in address-lər üçün istifadə etdiyi ilə eyni).
- Rəy silinəndə media sətirləri `ON DELETE CASCADE` ilə silinir, disk üzərindəki fayllar isə handler-də best-effort silinir (`os.RemoveAll` bütöv rəy qovluğunu).

## Stack
- Go 1.26 + chi router
- PostgreSQL (`pgx`) — `reviews`, `review_media` cədvəllərinin sahibi
- authorization-service ilə token yoxlanılır (istifadəçi ID-si üçün — rol yoxlanmır, çünki bütün authenticated istifadəçilər rəy yaza bilər)

## Setup

```bash
cp .env.example .env
go mod tidy
go run ./cmd/api
```

Default port: **8093**. Swagger UI: http://localhost:8093/swagger/index.html

## Endpoints

| Method | Path                                   | Auth           | Body / Qeyd |
|--------|-----------------------------------------|----------------|-------------|
| GET    | /health                                 | -              | - |
| GET    | /api/v1/products/{product_id}/reviews   | -              | rəylər + media, ən yenidən köhnəyə |
| POST   | /api/v1/products/{product_id}/reviews   | Bearer         | `{rating: 1-5, text?}` |
| POST   | /api/v1/reviews/{id}/media              | Bearer (sahib) | multipart, `file` sahəsi — jpeg/png/webp/heic şəkil, mp4/mov video, max 25MB |
| DELETE | /api/v1/reviews/{id}                    | Bearer (sahib) | - |
| GET    | /media/{path}                           | -              | yüklənmiş fayl (xam, JSON zərfsiz) |

## Nümunə

```bash
curl -X POST http://localhost:8093/api/v1/products/$PRODUCT_ID/reviews \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"rating":5,"text":"Cox rahatdir!"}'
# → {"id":"...","rating":5,"text":"Cox rahatdir!","media":[],...}

curl -X POST http://localhost:8093/api/v1/reviews/$REVIEW_ID/media \
  -H "Authorization: Bearer $TOKEN" -F "file=@photo.jpg;type=image/jpeg"
# → {"media_type":"image","url":"/media/reviews/<id>/<uuid>.jpg",...}
```

Bax [../ARCHITECTURE.md](../ARCHITECTURE.md) tam servislərarası axın üçün.
