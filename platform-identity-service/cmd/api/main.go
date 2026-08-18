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
	"platform-identity-service/internal/service/roleassign"
)

// @title           Platform Identity Service API
// @version         1.0
// @description     Gələcək layihələr üçün mərkəzi User/Admin qeydiyyat modulu. Hər layihənin ilk qeydiyyatdan keçən useri avtomatik admin olur, sonrakılar user. Öz JWT-sini özü verir/yoxlayır.
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

	projectRepo := repository.NewProjectRepository(db)
	userRepo := repository.NewUserRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	jwtManager := auth.NewJWTManager(cfg.JWTSecret, cfg.JWTTTLMinutes)

	projectService := service.NewProjectService(projectRepo)
	authService := service.NewAuthService(projectRepo, userRepo, jwtManager)
	userService := service.NewUserService(userRepo, roleassign.NewSuperadminOrSameProjectAdmin())
	roleService := service.NewRoleService(roleRepo)

	projectHandler := handlers.NewProjectHandler(projectService)
	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userService)
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
	r.Use(logclient.RequestLogger(logclient.New(cfg.LogServiceURL, "platform-identity-service")))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Get("/swagger/*", httpSwagger.WrapHandler)

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/projects", projectHandler.Create)
		r.Get("/projects", projectHandler.List)
		r.Get("/projects/{id}", projectHandler.Get)
		r.Get("/roles", roleHandler.List)

		r.Post("/auth/register", authHandler.Register)
		r.Post("/auth/login", authHandler.Login)

		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.RequireAuth(jwtManager))
			r.Get("/users/{id}", userHandler.Get)
			r.Post("/users/{id}/role", userHandler.SetRole)
			r.Get("/projects/{id}/users", userHandler.ListByProject)
			r.Get("/users", userHandler.ListAll)
		})
	})

	log.Printf("platform-identity-service listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
