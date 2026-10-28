package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func testRouter(t *testing.T) (chi.Router, *sql.DB) {
	t.Helper()
	dsn := os.Getenv("TEST_DSN")
	if dsn == "" {
		dsn = "host=localhost port=5433 user=postgres password=1 dbname=platform_identity sslmode=disable"
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if _, err := db.Exec(`CREATE SCHEMA IF NOT EXISTS test_handlers`); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	if _, err := db.Exec(`SET search_path TO test_handlers`); err != nil {
		t.Fatalf("set search_path: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	shopRepo := repository.NewShopRepository(db)
	membershipRepo := repository.NewShopMembershipRepository(db)
	tokenRevocationRepo := repository.NewTokenRevocationRepository(db)
	jwtManager := auth.NewJWTManager("handler-test-secret", 60)

	authSvc := service.NewAuthService(userRepo, jwtManager)
	userSvc := service.NewUserService(userRepo)
	shopSvc := service.NewShopService(shopRepo)
	membershipSvc := service.NewShopMembershipService(membershipRepo, userRepo, shopRepo, authSvc, userSvc)

	authHandler := NewAuthHandler(authSvc, tokenRevocationRepo)
	userHandler := NewUserHandler(userSvc, membershipSvc)
	shopHandler := NewShopHandler(shopSvc)
	membershipHandler := NewShopMembershipHandler(membershipSvc)

	r := chi.NewRouter()
	r.Post("/api/v1/auth/register", authHandler.Register)
	r.Post("/api/v1/auth/login", authHandler.Login)
	r.Post("/api/v1/shops", shopHandler.Create)
	r.Group(func(r chi.Router) {
		r.Use(appmiddleware.RequireAuth(jwtManager, userRepo, tokenRevocationRepo))
		r.Get("/api/v1/users/{id}", userHandler.Get)
		r.Get("/api/v1/users", userHandler.ListAll)
		r.Post("/api/v1/shops/{id}/members", membershipHandler.AddMember)
		r.Post("/api/v1/auth/logout", authHandler.Logout)
	})

	t.Cleanup(func() { db.Close() })
	return r, db
}

func doJSON(t *testing.T, r chi.Router, method, path, token string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestFullFlow_RegisterLoginGetUser(t *testing.T) {
	r, _ := testRouter(t)

	username := "flow-" + uuid.NewString()
	rec := doJSON(t, r, http.MethodPost, "/api/v1/auth/register", "", map[string]string{
		"name": "Flow User", "username": username, "email": uuid.NewString() + "@example.com", "password": "password123",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var userEnv struct {
		Data struct {
			ID         string `json:"id"`
			SystemRole string `json:"system_role"`
		} `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&userEnv)
	if userEnv.Data.SystemRole != models.SystemRoleUser {
		t.Fatalf("expected system_role 'user', got %q", userEnv.Data.SystemRole)
	}
	userID := userEnv.Data.ID

	rec = doJSON(t, r, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"identifier": username, "password": "password123",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var loginEnv struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&loginEnv)
	if loginEnv.Data.Token == "" {
		t.Fatal("expected non-empty token")
	}

	rec = doJSON(t, r, http.MethodGet, "/api/v1/users/"+userID, loginEnv.Data.Token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get user: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = doJSON(t, r, http.MethodGet, "/api/v1/users/"+userID, "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("get user without token: expected 401, got %d", rec.Code)
	}
}

func TestFullFlow_RegistrationCreatesNoShopMembership(t *testing.T) {
	r, db := testRouter(t)

	username := "noshop-" + uuid.NewString()
	rec := doJSON(t, r, http.MethodPost, "/api/v1/auth/register", "", map[string]string{
		"name": "No Shop", "username": username, "email": uuid.NewString() + "@example.com", "password": "password123",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var userEnv struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&userEnv)

	rec = doJSON(t, r, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"identifier": username, "password": "password123",
	})
	var loginEnv struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&loginEnv)
	if loginEnv.Data.Token == "" {
		t.Fatal("expected non-empty token")
	}

	membershipRepo := repository.NewShopMembershipRepository(db)
	memberships, err := membershipRepo.ListByUser(context.Background(), userEnv.Data.ID)
	if err != nil {
		t.Fatalf("list memberships: %v", err)
	}
	if len(memberships) != 0 {
		t.Fatalf("expected registration to create no shop memberships, got %d", len(memberships))
	}
}

// TestRequireAuth_RejectsTokenAfterUserDeactivated proves the core point of
// the RequireAuth rewrite: a user's status is re-checked from the database
// on every request, not just at login. A token issued while the user was
// ACTIVE must stop working the moment an admin flips them to IN_ACTIVE,
// without the user needing to log in again.
func TestRequireAuth_RejectsTokenAfterUserDeactivated(t *testing.T) {
	r, db := testRouter(t)

	username := "deactivate-" + uuid.NewString()
	rec := doJSON(t, r, http.MethodPost, "/api/v1/auth/register", "", map[string]string{
		"name": "Deactivate Me", "username": username, "email": uuid.NewString() + "@example.com", "password": "password123",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var userEnv struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&userEnv)
	userID := userEnv.Data.ID

	rec = doJSON(t, r, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"identifier": username, "password": "password123",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("login: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var loginEnv struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&loginEnv)
	token := loginEnv.Data.Token
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	// Sanity check: the token works while the user is still ACTIVE.
	rec = doJSON(t, r, http.MethodGet, "/api/v1/users/"+userID, token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get user while active: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Deactivate the user directly at the repository layer.
	userRepo := repository.NewUserRepository(db)
	if err := userRepo.SetStatus(context.Background(), userID, models.UserStatusInActive); err != nil {
		t.Fatalf("set status: %v", err)
	}

	// The SAME token, issued while the user was ACTIVE, must now be rejected.
	rec = doJSON(t, r, http.MethodGet, "/api/v1/users/"+userID, token, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("get user after deactivation: expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestLogout_RevokesTokenForThisSessionOnly proves the core point of this
// endpoint: logging out one token must reject that SAME token on its next
// use, while leaving a second, independently-issued token for the same
// user completely unaffected (e.g. the same account logged in on a second
// browser).
func TestLogout_RevokesTokenForThisSessionOnly(t *testing.T) {
	r, _ := testRouter(t)

	username := "logout-" + uuid.NewString()
	rec := doJSON(t, r, http.MethodPost, "/api/v1/auth/register", "", map[string]string{
		"name": "Logout Me", "username": username, "email": uuid.NewString() + "@example.com", "password": "password123",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var userEnv struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&userEnv)
	userID := userEnv.Data.ID

	login := func() string {
		rec := doJSON(t, r, http.MethodPost, "/api/v1/auth/login", "", map[string]string{
			"identifier": username, "password": "password123",
		})
		if rec.Code != http.StatusOK {
			t.Fatalf("login: expected 200, got %d: %s", rec.Code, rec.Body.String())
		}
		var loginEnv struct {
			Data struct {
				Token string `json:"token"`
			} `json:"data"`
		}
		json.NewDecoder(rec.Body).Decode(&loginEnv)
		if loginEnv.Data.Token == "" {
			t.Fatal("expected non-empty token")
		}
		return loginEnv.Data.Token
	}

	// Two independent logins, standing in for the same account signed in on
	// two separate browsers/devices.
	tokenA := login()
	tokenB := login()

	// Both work before either is logged out.
	rec = doJSON(t, r, http.MethodGet, "/api/v1/users/"+userID, tokenA, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("tokenA before logout: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	rec = doJSON(t, r, http.MethodGet, "/api/v1/users/"+userID, tokenB, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("tokenB before logout: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Log out tokenA only.
	rec = doJSON(t, r, http.MethodPost, "/api/v1/auth/logout", tokenA, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("logout: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// tokenA is now rejected...
	rec = doJSON(t, r, http.MethodGet, "/api/v1/users/"+userID, tokenA, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("tokenA after logout: expected 401, got %d: %s", rec.Code, rec.Body.String())
	}

	// ...but tokenB, the OTHER session for the same user, is untouched.
	rec = doJSON(t, r, http.MethodGet, "/api/v1/users/"+userID, tokenB, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("tokenB after tokenA's logout: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// A second logout call with the now-already-revoked tokenA fails at
	// RequireAuth itself (401) before Logout's own handler body ever runs —
	// there is nothing left to revoke, and the row already exists from the
	// first call.
	rec = doJSON(t, r, http.MethodPost, "/api/v1/auth/logout", tokenA, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("second logout with revoked token: expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}
