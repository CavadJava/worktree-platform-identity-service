package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	appconfig "log-service/internal/config"
	"log-service/internal/database"
	_ "log-service/internal/docs"
	"log-service/internal/handlers"
	"log-service/internal/repository"
)

// @title           Log Service API
// @version         1.0
// @description     Mərkəzi log anbarı — bütün servislər öz request/error loglarını buraya göndərir; servis, səviyyə və tarix üzrə filtrlə izləmək mümkündür.
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

	logRepo := repository.NewLogRepository(db)
	logHandler := handlers.NewLogHandler(logRepo)

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

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Get("/swagger/*", httpSwagger.WrapHandler)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/logs", logHandler.Create)
		r.Get("/logs", logHandler.List)
		r.Get("/logs/services", logHandler.ListServices)
	})

	log.Printf("log-service listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
