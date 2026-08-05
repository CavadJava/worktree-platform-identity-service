package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"registration-service/internal/auth"
	"registration-service/internal/client"
	appconfig "registration-service/internal/config"
	"registration-service/internal/database"
	_ "registration-service/internal/docs"
	"registration-service/internal/handlers"
	"registration-service/internal/repository"
	"registration-service/internal/service"
)

// @title           Registration Service API
// @version         1.0
// @description     İstifadəçi qeydiyyatı və login üçün REST API. Uğurlu qeydiyyatdan sonra notification-service-ə mock bildiriş göndərilir.
// @BasePath        /api/v1
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

	if err := database.Migrate(db); err != nil {
		log.Fatalf("migration error: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTTTLMinutes)
	notificationClient := client.NewNotificationClient(cfg.NotificationBaseURL)
	userService := service.NewUserService(userRepo, jwtManager, notificationClient)

	authHandler := handlers.NewAuthHandler(userService)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Get("/swagger/*", httpSwagger.WrapHandler)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
	})

	log.Printf("registration-service listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
