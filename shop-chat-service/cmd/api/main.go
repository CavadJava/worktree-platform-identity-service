package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	"shop-chat-service/internal/client"
	appconfig "shop-chat-service/internal/config"
	"shop-chat-service/internal/database"
	_ "shop-chat-service/internal/docs"
	"shop-chat-service/internal/handlers"
	"shop-chat-service/internal/logclient"
	appmiddleware "shop-chat-service/internal/middleware"
	"shop-chat-service/internal/repository"
	"shop-chat-service/internal/service"
)

// @title           Shop Chat Service API
// @version         1.0
// @description     İstifadəçi ilə mağaza arasında yazışma. Hər (mağaza, istifadəçi) cütü üçün bir söhbət mövcuddur. Söhbətdə yaza bilənlər: söhbətin sahibi olan müştəri, mağazanın chat(1)+ səviyyəli əməkdaşı, və ya sistem administratoru.
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

	conversationRepo := repository.NewConversationRepository(db)
	messageRepo := repository.NewMessageRepository(db)

	authClient := client.NewAuthorizationClient(cfg.AuthorizationBaseURL)
	chatService := service.NewChatService(conversationRepo, messageRepo)
	chatHandler := handlers.NewChatHandler(chatService)

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
	r.Use(logclient.RequestLogger(logclient.New(cfg.LogServiceURL, "shop-chat-service")))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Get("/swagger/*", httpSwagger.WrapHandler)

	r.Route("/api/v1", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(appmiddleware.RequireAuth(authClient))
			r.Post("/conversations", chatHandler.StartConversation)
			r.Get("/conversations", chatHandler.ListMyConversations)
			r.Get("/conversations/{id}/messages", chatHandler.ListMessages)
			r.Post("/conversations/{id}/messages", chatHandler.SendMessage)
			r.Get("/shops/{shop_id}/conversations", chatHandler.ListShopConversations)
		})
	})

	log.Printf("shop-chat-service listening on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
