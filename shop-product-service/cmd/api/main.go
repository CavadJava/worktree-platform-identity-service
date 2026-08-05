package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"shop-product-service/internal/client"
	appconfig "shop-product-service/internal/config"
	"shop-product-service/internal/database"
	_ "shop-product-service/internal/docs"
	"shop-product-service/internal/handlers"
	appmiddleware "shop-product-service/internal/middleware"
	"shop-product-service/internal/repository"
	"shop-product-service/internal/service"
)

// @title           Shop Product Service API
// @version         1.0
// @description     Mağaza məhsullarının CRUD idarəetməsi. Yaratma/yeniləmə/silmə üçün mağazaya həmin istifadəçi aid olmalı (JWT-də shop_id/shop_role_level ilə) və lazımi səviyyəyə sahib olmalıdır; və ya administrator.
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

	productRepo := repository.NewProductRepository(db)
	favoriteRepo := repository.NewFavoriteRepository(db)
	itemRepo := repository.NewProductItemRepository(db)
	authClient := client.NewAuthorizationClient(cfg.AuthorizationBaseURL)
	productService := service.NewProductService(productRepo)
	favoriteService := service.NewFavoriteService(favoriteRepo, productRepo)
	itemService := service.NewProductItemService(itemRepo, productRepo)
	productHandler := handlers.NewProductHandler(productService)
	favoriteHandler := handlers.NewFavoriteHandler(favoriteService)
	itemHandler := handlers.NewProductItemHandler(itemService)

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
		r.Get("/products", productHandler.List)
		r.Get("/products/{id}", productHandler.Get)
		r.Get("/products/{product_id}/items", itemHandler.List)
		r.Get("/product-items/{id}", itemHandler.Get)

		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.RequireAuth(authClient))
			r.Post("/products", productHandler.Create)
			r.Put("/products/{id}", productHandler.Update)
			r.Delete("/products/{id}", productHandler.Delete)

			r.Post("/products/{id}/favorite", favoriteHandler.Add)
			r.Delete("/products/{id}/favorite", favoriteHandler.Remove)
			r.Get("/favorites", favoriteHandler.List)

			r.Post("/products/{product_id}/items", itemHandler.Create)
			r.Put("/product-items/{id}", itemHandler.Update)
			r.Delete("/product-items/{id}", itemHandler.Delete)
		})
	})

	log.Printf("shop-product-service listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
