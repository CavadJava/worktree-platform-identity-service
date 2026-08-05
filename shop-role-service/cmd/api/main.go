package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"shop-role-service/internal/client"
	appconfig "shop-role-service/internal/config"
	"shop-role-service/internal/database"
	_ "shop-role-service/internal/docs"
	"shop-role-service/internal/handlers"
	appmiddleware "shop-role-service/internal/middleware"
	"shop-role-service/internal/repository"
	"shop-role-service/internal/service"
)

// @title           Shop Role Service API
// @version         1.0
// @description     İstifadəçini bir mağazaya hierarxik səviyyə ilə təyin edir: admin(4) > review(3) > add-product(2) > chat(1). Sistem administratoru istənilən mağazaya, mağazanın öz admin(4)-ü isə yalnız öz mağazasına əməkdaş təyin edə bilər.
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

	roleRepo := repository.NewRoleRepository(db)
	authClient := client.NewAuthorizationClient(cfg.AuthorizationBaseURL)
	roleService := service.NewRoleService(roleRepo)
	roleHandler := handlers.NewRoleHandler(roleService)

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
		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.RequireAuth(authClient))
			r.Post("/roles/assign", roleHandler.Assign)
			r.Post("/roles/revoke", roleHandler.Revoke)
			r.Get("/roles/{user_id}", roleHandler.Get)
			r.Get("/shops/{shop_id}/staff", roleHandler.ListStaff)
		})
	})

	log.Printf("shop-role-service listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
