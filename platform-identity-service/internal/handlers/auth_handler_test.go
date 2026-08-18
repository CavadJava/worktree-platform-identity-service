package handlers

import (
	"bytes"
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
	"platform-identity-service/internal/repository"
	"platform-identity-service/internal/service"
	"platform-identity-service/internal/service/roleassign"
)

func testRouter(t *testing.T) (chi.Router, *sql.DB) {
	t.Helper()
	dsn := os.Getenv("TEST_DSN")
	if dsn == "" {
		dsn = "host=localhost port=5433 user=postgres password=1 dbname=postgres sslmode=disable"
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	projectRepo := repository.NewProjectRepository(db)
	userRepo := repository.NewUserRepository(db)
	jwtManager := auth.NewJWTManager("handler-test-secret", 60)

	projectSvc := service.NewProjectService(projectRepo)
	authSvc := service.NewAuthService(projectRepo, userRepo, jwtManager)
	userSvc := service.NewUserService(userRepo, roleassign.NewSameProjectAdmin())

	projectHandler := NewProjectHandler(projectSvc)
	authHandler := NewAuthHandler(authSvc)
	userHandler := NewUserHandler(userSvc)

	r := chi.NewRouter()
	r.Post("/api/v1/projects", projectHandler.Create)
	r.Post("/api/v1/auth/register", authHandler.Register)
	r.Post("/api/v1/auth/login", authHandler.Login)
	r.Group(func(r chi.Router) {
		r.Use(appmiddleware.RequireAuth(jwtManager))
		r.Get("/api/v1/users/{id}", userHandler.Get)
		r.Post("/api/v1/users/{id}/role", userHandler.SetRole)
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

	// Create a project
	rec := doJSON(t, r, http.MethodPost, "/api/v1/projects", "", map[string]string{"name": "Flow Test " + uuid.NewString()})
	if rec.Code != http.StatusCreated {
		t.Fatalf("create project: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var projectEnv struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&projectEnv)
	projectID := projectEnv.Data.ID

	// Register the first user (should become admin)
	username := "flow-" + uuid.NewString()
	rec = doJSON(t, r, http.MethodPost, "/api/v1/auth/register", "", map[string]string{
		"project_id": projectID, "name": "Flow User", "username": username,
		"email": uuid.NewString() + "@example.com", "password": "password123",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var userEnv struct {
		Data struct {
			ID   string `json:"id"`
			Role string `json:"role"`
		} `json:"data"`
	}
	json.NewDecoder(rec.Body).Decode(&userEnv)
	if userEnv.Data.Role != "admin" {
		t.Fatalf("expected first user role 'admin', got %q", userEnv.Data.Role)
	}
	userID := userEnv.Data.ID

	// Login
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

	// Get own user with the token
	rec = doJSON(t, r, http.MethodGet, "/api/v1/users/"+userID, loginEnv.Data.Token, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get user: expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Get without a token should be unauthorized
	rec = doJSON(t, r, http.MethodGet, "/api/v1/users/"+userID, "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("get user without token: expected 401, got %d", rec.Code)
	}
}
