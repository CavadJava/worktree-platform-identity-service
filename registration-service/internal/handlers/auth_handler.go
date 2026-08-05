package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	_ "registration-service/internal/models" // referenced by swag annotations
	"registration-service/internal/service"
)

type AuthHandler struct {
	svc *service.UserService
}

func NewAuthHandler(svc *service.UserService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
	Phone    string `json:"phone"`
}

// Register godoc
// @Summary      Register a new user
// @Description  Yeni istifadəçi qeydiyyatı. Email unikal olmalıdır.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body registerRequest true "Registration payload"
// @Success      201 {object} models.User
// @Failure      400 {object} map[string]string
// @Failure      409 {object} map[string]string "email already registered"
// @Router       /register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.FullName = strings.TrimSpace(req.FullName)

	if req.Email == "" || !strings.Contains(req.Email, "@") {
		writeError(w, http.StatusBadRequest, "valid email is required")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}
	if req.FullName == "" {
		writeError(w, http.StatusBadRequest, "full_name is required")
		return
	}

	user, err := h.svc.Register(r.Context(), service.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
		Phone:    req.Phone,
	})
	if err != nil {
		if errors.Is(err, service.ErrEmailTaken) {
			writeError(w, http.StatusConflict, "email already registered")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to register user")
		return
	}

	writeJSON(w, http.StatusCreated, user)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

// Login godoc
// @Summary      Login
// @Description  Email/password ilə login olub JWT token alır.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body loginRequest true "Login payload"
// @Success      200 {object} loginResponse
// @Failure      400 {object} map[string]string
// @Failure      401 {object} map[string]string "invalid email or password"
// @Router       /login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to login")
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}
