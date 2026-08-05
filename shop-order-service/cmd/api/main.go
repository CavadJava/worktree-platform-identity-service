package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"shop-order-service/internal/client"
	appconfig "shop-order-service/internal/config"
	"shop-order-service/internal/database"
	_ "shop-order-service/internal/docs"
	"shop-order-service/internal/handlers"
	"shop-order-service/internal/logclient"
	appmiddleware "shop-order-service/internal/middleware"
	"shop-order-service/internal/repository"
	"shop-order-service/internal/service"
)

// @title           Shop Order Service API
// @version         1.0
// @description     İstifadəçilərin mağazalardan məhsul seçib sifariş yaratması. Sifariş nömrəsi "U<istifadəçi_seq>-S<mağaza_seq>-<n>" formatındadır (<n> — istifadəçinin neçənci sifarişi olduğu).
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

	orderRepo := repository.NewOrderRepository(db)
	lookupRepo := repository.NewLookupRepository(db)

	authClient := client.NewAuthorizationClient(cfg.AuthorizationBaseURL)
	orderService := service.NewOrderService(orderRepo, lookupRepo)
	orderHandler := handlers.NewOrderHandler(orderService)

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
	r.Use(logclient.RequestLogger(logclient.New(cfg.LogServiceURL, "shop-order-service")))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Get("/swagger/*", httpSwagger.WrapHandler)

	r.Route("/api/v1", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.RequireAuth(authClient))
			r.Post("/orders", orderHandler.Create)
			r.Get("/orders", orderHandler.ListMine)
			r.Get("/orders/{id}", orderHandler.Get)
			r.Get("/shops/{shop_id}/orders", orderHandler.ListForShop)
		})
	})

	log.Printf("shop-order-service listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
