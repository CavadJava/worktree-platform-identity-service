package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"notification-service/internal/client"
	appconfig "notification-service/internal/config"
	"notification-service/internal/database"
	_ "notification-service/internal/docs"
	"notification-service/internal/handlers"
	"notification-service/internal/logclient"
	appmiddleware "notification-service/internal/middleware"
	"notification-service/internal/repository"
)

// @title           Notification Service API
// @version         1.0
// @description     Bildirişləri qəbul edib (mock) göndərən REST API. Real provider inteqrasiyası yoxdur, konsola loglayır.
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

	notificationRepo := repository.NewNotificationRepository(db)
	authClient := client.NewAuthorizationClient(cfg.AuthorizationBaseURL)
	notificationHandler := handlers.NewNotificationHandler(notificationRepo)

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
	r.Use(logclient.RequestLogger(logclient.New(cfg.LogServiceURL, "notification-service")))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Get("/swagger/*", httpSwagger.WrapHandler)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/notifications", notificationHandler.Send)

		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.Auth(authClient))
			r.Get("/notifications", notificationHandler.List)
			r.Get("/notifications/unread-count", notificationHandler.UnreadCount)
			r.Put("/notifications/{id}/read", notificationHandler.MarkRead)
			r.Put("/notifications/read-all", notificationHandler.MarkAllRead)
		})
	})

	log.Printf("notification-service listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
