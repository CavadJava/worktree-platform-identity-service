package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"shop-category-service/internal/client"
	appconfig "shop-category-service/internal/config"
	"shop-category-service/internal/database"
	_ "shop-category-service/internal/docs"
	"shop-category-service/internal/handlers"
	"shop-category-service/internal/logclient"
	appmiddleware "shop-category-service/internal/middleware"
	"shop-category-service/internal/repository"
	"shop-category-service/internal/service"
)

// @title           Shop Category Service API
// @version         1.0
// @description     Sistem-səviyyəli kataloq: kateqoriyalar və alt-kateqoriyalar. Oxumaq public-dir; yaratma/yeniləmə/silmə yalnız administrator üçündür. İlk açılışda default kataloq idempotent seed edilir.
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

	categoryRepo := repository.NewCategoryRepository(db)
	authClient := client.NewAuthorizationClient(cfg.AuthorizationBaseURL)
	categoryService := service.NewCategoryService(categoryRepo)
	categoryHandler := handlers.NewCategoryHandler(categoryService)

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
	r.Use(logclient.RequestLogger(logclient.New(cfg.LogServiceURL, "shop-category-service")))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Get("/swagger/*", httpSwagger.WrapHandler)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/categories", categoryHandler.ListCategories)
		r.Get("/categories/{id}", categoryHandler.GetCategory)
		r.Get("/categories/{id}/subcategories", categoryHandler.ListSubcategories)
		r.Get("/subcategories", categoryHandler.ListAllSubcategories)

		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.RequireAdministrator(authClient))
			r.Post("/categories", categoryHandler.CreateCategory)
			r.Put("/categories/{id}", categoryHandler.UpdateCategory)
			r.Delete("/categories/{id}", categoryHandler.DeleteCategory)
			r.Post("/categories/{id}/subcategories", categoryHandler.CreateSubcategory)
			r.Put("/subcategories/{id}", categoryHandler.UpdateSubcategory)
			r.Delete("/subcategories/{id}", categoryHandler.DeleteSubcategory)
		})
	})

	log.Printf("shop-category-service listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
