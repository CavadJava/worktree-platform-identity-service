package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"user-service/internal/client"
	appconfig "user-service/internal/config"
	"user-service/internal/database"
	_ "user-service/internal/docs"
	"user-service/internal/handlers"
	appmiddleware "user-service/internal/middleware"
	"user-service/internal/repository"
	"user-service/internal/service"
)

// @title           User Service API
// @version         1.0
// @description     İstifadəçi profilinin oxunması/yenilənməsi üçün REST API. Auth doğrulaması authorization-service-ə həvalə edilir.
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Bearer token formatında JWT (registration-service /login cavabından).
func main() {
	cfg, err := appconfig.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("database error: %v", err)
	}
	defer db.Close()

	userRepo := repository.NewUserRepository(db)
	authClient := client.NewAuthorizationClient(cfg.AuthorizationBaseURL)
	profileService := service.NewProfileService(userRepo)
	profileHandler := handlers.NewProfileHandler(profileService)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Get("/swagger/*", httpSwagger.WrapHandler)

	r.Route("/api/v1", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.Auth(authClient))
			r.Get("/profile", profileHandler.Get)
			r.Put("/profile", profileHandler.Update)
		})
	})

	log.Printf("user-service listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
