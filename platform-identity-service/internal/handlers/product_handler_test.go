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

	reqRepo := repository.NewProductAdminRequestRepository(db)
	authService := service.NewAuthService(userRepo, jwtManager)
	productService := service.NewProductService(productRepo, subRepo, subprojRepo, reqRepo, userRepo, authService)
	productHandler := NewProductHandler(productService)
	authHandler := NewAuthHandler(authService)

	r := chi.NewRouter()
	r.Get("/api/v1/products", productHandler.List)
	r.Get("/api/v1/products/{id}/subprojects", productHandler.ListSubprojects)
	r.Post("/api/v1/auth/login", authHandler.Login)
	r.Group(func(r chi.Router) {
		r.Use(appmiddleware.RequireAuth(jwtManager, userRepo))
		r.Post("/api/v1/products", productHandler.Create)
		r.Post("/api/v1/products/{id}/users", productHandler.CreateUserAndSubscribe)
		r.Post("/api/v1/products/{id}/profile", productHandler.UpdateProfile)
		r.Post("/api/v1/products/{id}/subprojects", productHandler.AddSubproject)
		r.Delete("/api/v1/products/{id}/subprojects/{subId}", productHandler.RemoveSubproject)
		r.Get("/api/v1/products/mine", productHandler.ListMine)
		r.Get("/api/v1/products/browse", productHandler.ListBrowse)
		r.Post("/api/v1/products/{id}/admin-requests", productHandler.RequestAdmin)
		r.Get("/api/v1/products/{id}/admin-requests", productHandler.ListAdminRequests)
		r.Post("/api/v1/products/{id}/admin-requests/{requestId}/decide", productHandler.DecideAdminRequest)
		r.Post("/api/v1/products/{id}/admin", productHandler.PromoteAdmin)
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

func loginAs(t *testing.T, r chi.Router, username, password string) string {
	t.Helper()
	rec := doJSON(t, r, http.MethodPost, "/api/v1/auth/login", "", map[string]string{"identifier": username, "password": password})
	if rec.Code != http.StatusOK {
		t.Fatalf("login as %s: expected 200, got %d: %s", username, rec.Code, rec.Body.String())
	}
	var env struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&env)
	return env.Data.Token
}

func TestProductHandler_ScopedListAndAdminRequestFlow(t *testing.T) {
	r, db, jwtManager := testProductRouter(t)
	superToken := createTestSuperadmin(t, db, jwtManager)

	// Create two products; only subscribe the admin to the first.
	rec := doJSON(t, r, http.MethodPost, "/api/v1/products", superToken, map[string]string{"name": "Teslahubs"})
	var p1Env struct {
		Data productResponse `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&p1Env)
	p1 := p1Env.Data

	rec = doJSON(t, r, http.MethodPost, "/api/v1/products", superToken, map[string]string{"name": "ESound"})
	var p2Env struct {
		Data productResponse `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&p2Env)
	p2 := p2Env.Data

	// Create a plain user via the product-user endpoint, requesting the
	// 'admin' role directly (new-account exemption from request/approval).
	rec = doJSON(t, r, http.MethodPost, "/api/v1/products/"+p1.ID+"/users", superToken, map[string]string{
		"name": "Admin User", "username": "adminuser-" + p1.ID, "email": "adminuser-" + p1.ID + "@example.com",
		"password": "password123", "system_role": "admin",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create product user with admin role: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var adminSubEnv struct {
		Data subscriptionResponse `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&adminSubEnv)
	adminUserID := adminSubEnv.Data.UserID

	adminToken := loginAs(t, r, "adminuser-"+p1.ID, "password123")

	// The admin's scoped list only includes p1, not p2.
	rec = doJSON(t, r, http.MethodGet, "/api/v1/products/mine", adminToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list mine: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var mineEnv struct {
		Data []productResponse `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&mineEnv)
	if len(mineEnv.Data) != 1 || mineEnv.Data[0].ID != p1.ID {
		t.Fatalf("expected admin to see only p1, got %+v", mineEnv.Data)
	}

	// Browse list shows both, read-only.
	rec = doJSON(t, r, http.MethodGet, "/api/v1/products/browse", adminToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("browse: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Superadmin creates a plain user subscribed to p2 only (not p1), so
	// the user has no existing access to p1 — the admin then requests
	// admin access for that user first on p2 (which the admin does NOT
	// manage, must fail) and then on p1 (which the user still has no
	// access to, and the admin does manage, must succeed).
	rec = doJSON(t, r, http.MethodPost, "/api/v1/products/"+p2.ID+"/users", superToken, map[string]string{
		"name": "Plain User", "username": "plainuser-" + p1.ID, "email": "plainuser-" + p1.ID + "@example.com",
		"password": "password123", "system_role": "user",
	})
	var plainSubEnv struct {
		Data subscriptionResponse `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&plainSubEnv)
	plainUserID := plainSubEnv.Data.UserID

	rec = doJSON(t, r, http.MethodPost, "/api/v1/products/"+p2.ID+"/admin-requests", adminToken, map[string]string{"subject_user_id": plainUserID})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("request admin for unmanaged product: expected 403, got %d: %s", rec.Code, rec.Body.String())
	}

	// Requesting on p1 (which the admin does manage) succeeds.
	rec = doJSON(t, r, http.MethodPost, "/api/v1/products/"+p1.ID+"/admin-requests", adminToken, map[string]string{"subject_user_id": plainUserID})
	if rec.Code != http.StatusCreated {
		t.Fatalf("request admin: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var reqEnv struct {
		Data productAdminRequestResponse `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&reqEnv)

	// The admin cannot decide their own request.
	rec = doJSON(t, r, http.MethodPost, "/api/v1/products/"+p1.ID+"/admin-requests/"+reqEnv.Data.ID+"/decide", adminToken, map[string]bool{"approve": true})
	if rec.Code != http.StatusForbidden {
		t.Fatalf("admin deciding own request: expected 403, got %d: %s", rec.Code, rec.Body.String())
	}

	// Superadmin approves it.
	rec = doJSON(t, r, http.MethodPost, "/api/v1/products/"+p1.ID+"/admin-requests/"+reqEnv.Data.ID+"/decide", superToken, map[string]bool{"approve": true})
	if rec.Code != http.StatusOK {
		t.Fatalf("superadmin decide: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Superadmin can also promote directly, with no request at all.
	rec = doJSON(t, r, http.MethodPost, "/api/v1/products/"+p2.ID+"/admin", superToken, map[string]string{"subject_user_id": adminUserID})
	if rec.Code != http.StatusOK {
		t.Fatalf("promote directly: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}
