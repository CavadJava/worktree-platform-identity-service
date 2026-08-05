package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"localization-service/internal/client"
	appconfig "localization-service/internal/config"
	"localization-service/internal/database"
	_ "localization-service/internal/docs"
	"localization-service/internal/handlers"
	"localization-service/internal/logclient"
	appmiddleware "localization-service/internal/middleware"
	"localization-service/internal/repository"
	"localization-service/internal/service"
)

// @title           Localization Service API
// @version         1.0
// @description     Sistemdəki bütün mətnlərin/xəta mesajlarının çoxdilli (az/en/ru) idarəetməsi. Oxumaq (list/map/lookup/locales) public-dir; yaratma/yeniləmə/silmə yalnız administrator üçündür.
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
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

	translationRepo := repository.NewTranslationRepository(db)
	authClient := client.NewAuthorizationClient(cfg.AuthorizationBaseURL)
	translationService := service.NewTranslationService(translationRepo, cfg.DefaultLocale)
	translationHandler := handlers.NewTranslationHandler(translationService)

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
	r.Use(logclient.RequestLogger(logclient.New(cfg.LogServiceURL, "localization-service")))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Get("/swagger/*", httpSwagger.WrapHandler)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/translations", translationHandler.List)
		r.Get("/translations/map", translationHandler.Map)
		r.Get("/translations/lookup", translationHandler.Lookup)
		r.Get("/locales", translationHandler.ListLocales)

		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.RequireAdministrator(authClient))
			r.Post("/translations", translationHandler.Upsert)
			r.Delete("/translations/{id}", translationHandler.Delete)
		})
	})

	log.Printf("localization-service listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
