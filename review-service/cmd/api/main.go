package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"review-service/internal/client"
	appconfig "review-service/internal/config"
	"review-service/internal/database"
	_ "review-service/internal/docs"
	"review-service/internal/handlers"
	"review-service/internal/logclient"
	appmiddleware "review-service/internal/middleware"
	"review-service/internal/repository"
	"review-service/internal/service"
)

// @title           Review Service API
// @version         1.0
// @description     Məhsullara reyting/mətn rəyi və şəkil/video əlavə etmək üçün REST API. Fayllar diskdə saxlanılır, /media/* altında verilir.
// @BasePath        /api/v1
func main() {
	cfg := appconfig.Load()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("database error: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(db); err != nil {
		log.Fatalf("migration error: %v", err)
	}

	reviewRepo := repository.NewReviewRepository(db)
	reviewSvc := service.NewReviewService(reviewRepo)
	authClient := client.NewAuthorizationClient(cfg.AuthorizationBaseURL)
	reviewHandler := handlers.NewReviewHandler(reviewSvc, cfg.UploadDir, cfg.MaxUploadBytes)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSAllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	r.Use(logclient.RequestLogger(logclient.New(cfg.LogServiceURL, "review-service")))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Get("/swagger/*", httpSwagger.WrapHandler)

	// Uploaded review photos/clips — raw file serving, no JSON envelope, so
	// it sits outside /api/v1 (same as /health and /swagger).
	r.Handle("/media/*", http.StripPrefix("/media/", http.FileServer(http.Dir(cfg.UploadDir))))

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/products/{product_id}/reviews", reviewHandler.List)

		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.Auth(authClient))
			r.Post("/products/{product_id}/reviews", reviewHandler.Create)
			r.Delete("/reviews/{id}", reviewHandler.Delete)
			r.Post("/reviews/{id}/media", reviewHandler.AddMedia)
		})
	})

	log.Printf("review-service listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
