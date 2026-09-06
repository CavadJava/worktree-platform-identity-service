package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"

	"platform-identity-service/internal/auth"
	"platform-identity-service/internal/database"
	appmiddleware "platform-identity-service/internal/middleware"
	"platform-identity-service/internal/models"
	"platform-identity-service/internal/repository"
	"platform-identity-service/internal/service"
)

func testProductRouter(t *testing.T) (chi.Router, *sql.DB, *auth.JWTManager) {
	t.Helper()
	dsn := os.Getenv("TEST_DSN")
	if dsn == "" {
		dsn = "host=localhost port=5433 user=postgres password=1 dbname=platform_identity sslmode=disable"
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if _, err := db.Exec(`CREATE SCHEMA IF NOT EXISTS test_handlers_product`); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	if _, err := db.Exec(`SET search_path TO test_handlers_product`); err != nil {
		t.Fatalf("set search_path: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	productRepo := repository.NewProductRepository(db)
	subRepo := repository.NewSubscriptionRepository(db)
	subprojRepo := repository.NewSubprojectRepository(db)
	jwtManager := auth.NewJWTManager("product-handler-test-secret", 60)

	authService := service.NewAuthService(userRepo, jwtManager)
	productService := service.NewProductService(productRepo, subRepo, subprojRepo, authService)
	productHandler := NewProductHandler(productService)

	r := chi.NewRouter()
	r.Get("/api/v1/products", productHandler.List)
	r.Get("/api/v1/products/{id}/subprojects", productHandler.ListSubprojects)
	r.Group(func(r chi.Router) {
		r.Use(appmiddleware.RequireAuth(jwtManager, userRepo))
		r.Post("/api/v1/products", productHandler.Create)
		r.Post("/api/v1/products/{id}/users", productHandler.CreateUserAndSubscribe)
		r.Post("/api/v1/products/{id}/profile", productHandler.UpdateProfile)
		r.Post("/api/v1/products/{id}/subprojects", productHandler.AddSubproject)
		r.Delete("/api/v1/products/{id}/subprojects/{subId}", productHandler.RemoveSubproject)
	})

	t.Cleanup(func() { db.Close() })
	return r, db, jwtManager
}

func createTestSuperadmin(t *testing.T, db *sql.DB, jwtManager *auth.JWTManager) string {
	t.Helper()
	userRepo := repository.NewUserRepository(db)
	u := &models.User{
		ID: uuid.NewString(), Name: "Super", Username: "super-" + uuid.NewString(), Email: "super-" + uuid.NewString() + "@example.com",
		PasswordHash: "x", SystemRoleID: 1, Status: models.UserStatusActive,
	}
	if err := userRepo.Create(context.Background(), u); err != nil {
		t.Fatalf("create superadmin: %v", err)
	}
	token, _, err := jwtManager.Generate(u.ID, models.SystemRoleSuperadmin)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	return token
}

func TestProductHandler_CreateUserAndSubscribe(t *testing.T) {
	r, db, jwtManager := testProductRouter(t)
	token := createTestSuperadmin(t, db, jwtManager)

	rec := doJSON(t, r, http.MethodPost, "/api/v1/products", token, map[string]string{"name": "ESound"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create product: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var createdEnv struct {
		Data productResponse `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&createdEnv)
	product := createdEnv.Data

	rec = doJSON(t, r, http.MethodPost, "/api/v1/products/"+product.ID+"/users", token, map[string]string{
		"name": "Product User", "username": "produser-" + product.ID, "email": "produser-" + product.ID + "@example.com", "password": "password123",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create user and subscribe: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var subEnv struct {
		Data subscriptionResponse `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&subEnv)
	if subEnv.Data.ProductID != product.ID || !subEnv.Data.Subscripted {
		t.Fatalf("expected subscribed subscription for product, got %+v", subEnv.Data)
	}
}

func TestProductHandler_ProfileAndSubprojects(t *testing.T) {
	r, db, jwtManager := testProductRouter(t)
	token := createTestSuperadmin(t, db, jwtManager)

	rec := doJSON(t, r, http.MethodPost, "/api/v1/products", token, map[string]string{"name": "Teslahubs"})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create product: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var createdEnv struct {
		Data productResponse `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&createdEnv)
	created := createdEnv.Data

	rec = doJSON(t, r, http.MethodPost, "/api/v1/products/"+created.ID+"/profile", token, map[string]string{
		"description": "A Tesla platform", "tech_stack": "React, Go, Postgres",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("update profile: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var profileEnv struct {
		Data productResponse `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&profileEnv)
	profile := profileEnv.Data
	if profile.Description != "A Tesla platform" || profile.TechStack != "React, Go, Postgres" {
		t.Fatalf("expected updated profile in response, got %+v", profile)
	}

	rec = doJSON(t, r, http.MethodPost, "/api/v1/products/"+created.ID+"/subprojects", token, map[string]string{
		"name": "auth-service", "description": "handles login",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("add subproject: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var subEnv struct {
		Data subprojectResponse `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&subEnv)
	sub := subEnv.Data
	if sub.Name != "auth-service" {
		t.Fatalf("expected subproject name auth-service, got %+v", sub)
	}

	rec = doJSON(t, r, http.MethodGet, "/api/v1/products/"+created.ID+"/subprojects", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list subprojects: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var listEnv struct {
		Data []subprojectResponse `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&listEnv)
	if len(listEnv.Data) != 1 {
		t.Fatalf("expected 1 subproject, got %d", len(listEnv.Data))
	}

	rec = doJSON(t, r, http.MethodDelete, "/api/v1/products/"+created.ID+"/subprojects/"+sub.ID, token, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("remove subproject: expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}
