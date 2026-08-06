package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"payment-service/internal/client"
	appconfig "payment-service/internal/config"
	"payment-service/internal/database"
	_ "payment-service/internal/docs"
	"payment-service/internal/handlers"
	"payment-service/internal/logclient"
	appmiddleware "payment-service/internal/middleware"
	"payment-service/internal/repository"
	"payment-service/internal/service"
)

// @title           Payment Service API
// @version         1.0
// @description     Sifariş ödənişlərinin qeydiyyatı və mağazaların müvəqqəti bakiyəsi. shop-order-service hər sifariş yaradılandan sonra bura POST /payments göndərir; mağaza öz ödənişlərini və bakiyəsini burdan izləyir.
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

	paymentRepo := repository.NewPaymentRepository(db)

	authClient := client.NewAuthorizationClient(cfg.AuthorizationBaseURL)
	paymentService := service.NewPaymentService(paymentRepo)
	paymentHandler := handlers.NewPaymentHandler(paymentService)

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
	r.Use(logclient.RequestLogger(logclient.New(cfg.LogServiceURL, "payment-service")))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Get("/swagger/*", httpSwagger.WrapHandler)

	r.Route("/api/v1", func(r chi.Router) {
		// Internal, service-to-service — no auth (same convention as
		// notification-service's POST /notifications).
		r.Post("/payments", paymentHandler.Create)

		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.RequireAuth(authClient))
			r.Get("/shops/{id}/payments", paymentHandler.ListByShop)
			r.Get("/shops/{id}/balance", paymentHandler.GetBalance)
		})
	})

	log.Printf("payment-service listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
