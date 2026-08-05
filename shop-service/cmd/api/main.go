package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"shop-service/internal/client"
	appconfig "shop-service/internal/config"
	"shop-service/internal/database"
	_ "shop-service/internal/docs"
	"shop-service/internal/handlers"
	appmiddleware "shop-service/internal/middleware"
	"shop-service/internal/repository"
	"shop-service/internal/service"
)

// @title           Shop Service API
// @version         1.0
// @description     Mağazaların CRUD idarəetməsi + mağaza açma müraciəti/təsdiq axını. Birbaşa yaratma yalnız administrator üçündür; adi istifadəçilər /shop-applications ilə müraciət edir, administrator formu göndərəndə (send-form) müvəqqəti mağaza yaranır və müraciət sahibi onun admin(4) səviyyəli sahibi olur, yekun approve ilə mağaza daimi olur.
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

	shopRepo := repository.NewShopRepository(db)
	appRepo := repository.NewShopApplicationRepository(db)
	userAssignRepo := repository.NewUserAssignmentRepository(db)

	authClient := client.NewAuthorizationClient(cfg.AuthorizationBaseURL)
	notificationClient := client.NewNotificationClient(cfg.NotificationBaseURL)

	shopService := service.NewShopService(shopRepo)
	appService := service.NewShopApplicationService(appRepo, shopRepo, userAssignRepo, notificationClient)

	shopHandler := handlers.NewShopHandler(shopService)
	appHandler := handlers.NewShopApplicationHandler(appService)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Get("/swagger/*", httpSwagger.WrapHandler)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/shops", shopHandler.List)
		r.Get("/shops/{id}", shopHandler.Get)

		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.RequireAdministrator(authClient))
			r.Post("/shops", shopHandler.Create)
		})

		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.RequireAuth(authClient))
			r.Put("/shops/{id}", shopHandler.Update)
			r.Delete("/shops/{id}", shopHandler.Delete)

			r.Post("/shop-applications", appHandler.Submit)
			r.Get("/shop-applications/{id}", appHandler.Get)
		})

		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.RequireAdministrator(authClient))
			r.Get("/shop-applications", appHandler.List)
			r.Post("/shop-applications/{id}/send-form", appHandler.SendForm)
			r.Post("/shop-applications/{id}/approve", appHandler.Approve)
			r.Post("/shop-applications/{id}/reject", appHandler.Reject)
		})
	})

	log.Printf("shop-service listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
