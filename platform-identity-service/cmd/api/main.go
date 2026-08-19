package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"platform-identity-service/internal/auth"
	appconfig "platform-identity-service/internal/config"
	"platform-identity-service/internal/database"
	_ "platform-identity-service/internal/docs"
	"platform-identity-service/internal/handlers"
	"platform-identity-service/internal/logclient"
	appmiddleware "platform-identity-service/internal/middleware"
	"platform-identity-service/internal/repository"
	"platform-identity-service/internal/service"
)

// @title           Teslahubs Identity Service API
// @version         2.0
// @description     Teslahubs-un mərkəzi identity modulu: istifadəçilər Shop-lara (many-to-many, shop-admin/shop-user rolları ilə) üzv ola bilər və Product-lara (manual subscription) abunə ola bilər. Sistem rolları: superadmin/admin/user.
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
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

	superadminHash, err := auth.HashPassword(cfg.SuperadminPassword)
	if err != nil {
		log.Fatalf("failed to hash superadmin password: %v", err)
	}
	if err := database.SeedSuperadmin(db, uuid.NewString(), cfg.SuperadminUsername, superadminHash); err != nil {
		log.Fatalf("failed to seed superadmin: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	shopRepo := repository.NewShopRepository(db)
	membershipRepo := repository.NewShopMembershipRepository(db)
	productRepo := repository.NewProductRepository(db)
	subscriptionRepo := repository.NewSubscriptionRepository(db)
	systemRoleRepo := repository.NewSystemRoleRepository(db)
	shopRoleRepo := repository.NewShopRoleRepository(db)
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTTTLMinutes)

	authService := service.NewAuthService(userRepo, jwtManager)
	userService := service.NewUserService(userRepo)
	shopService := service.NewShopService(shopRepo)
	membershipService := service.NewShopMembershipService(membershipRepo, userRepo, shopRepo, authService)
	productService := service.NewProductService(productRepo, subscriptionRepo)
	systemRoleService := service.NewSystemRoleService(systemRoleRepo)
	shopRoleService := service.NewShopRoleService(shopRoleRepo)

	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userService, membershipService)
	shopHandler := handlers.NewShopHandler(shopService)
	membershipHandler := handlers.NewShopMembershipHandler(membershipService)
	productHandler := handlers.NewProductHandler(productService)
	systemRoleHandler := handlers.NewSystemRoleHandler(systemRoleService)
	shopRoleHandler := handlers.NewShopRoleHandler(shopRoleService)

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
	r.Use(logclient.RequestLogger(logclient.New(cfg.LogServiceURL, "platform-identity-service")))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Get("/swagger/*", httpSwagger.WrapHandler)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/shops", shopHandler.Create)
		r.Get("/shops", shopHandler.List)
		r.Get("/shops/{id}", shopHandler.Get)
		r.Get("/system-roles", systemRoleHandler.List)
		r.Get("/shop-roles", shopRoleHandler.List)
		r.Get("/products", productHandler.List)

		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.RequireAuth(jwtManager, userRepo))

			r.Get("/users/{id}", userHandler.Get)
			r.Get("/users", userHandler.ListAll)
			r.Post("/users", authHandler.CreateUser)
			r.Post("/users/{id}/system-role", userHandler.SetSystemRole)
			r.Post("/users/{id}/status", userHandler.SetStatus)
			r.Get("/users/{id}/shops", userHandler.ListMyShops)

			r.Post("/shops/{id}/members", membershipHandler.AddMember)
			r.Post("/shops/{id}/members/new", membershipHandler.AddNewMember)
			r.Get("/shops/{id}/members", membershipHandler.ListMembers)
			r.Post("/shops/{id}/members/{userId}/role", membershipHandler.SetMemberRole)

			r.Post("/products", productHandler.Create)
			r.Get("/products/{id}/access", productHandler.CheckAccess)
			r.Post("/users/{userId}/products/{productId}/subscribe", productHandler.SetSubscription)
		})
	})

	log.Printf("platform-identity-service listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
