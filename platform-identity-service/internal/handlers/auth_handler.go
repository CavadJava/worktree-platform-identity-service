package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"platform-identity-service/internal/service"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

type registerRequest struct {
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Username   string `json:"username"`
	Email      string `json:"email"`
	SystemRole string `json:"system_role"`
	Status     string `json:"status"`
}

// Register godoc
// @Summary      Register a Teslahubs account
// @Description  Always creates system role 'user' with no shop membership — shop_role sahəsi qəbul edilmir.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body registerRequest true "Register payload"
// @Success      201 {object} userResponse
// @Failure      400 {object} map[string]string
// @Failure      409 {object} map[string]string
// @Router       /auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Username == "" || req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username, email and password are required")
		return
	}

	u, err := h.svc.Register(r.Context(), service.RegisterInput{
		Name: req.Name, Username: req.Username, Email: req.Email, Password: req.Password,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUsernameTaken):
			writeError(w, http.StatusConflict, "username already registered")
		case errors.Is(err, service.ErrEmailTaken):
			writeError(w, http.StatusConflict, "email already registered")
		default:
			writeError(w, http.StatusInternalServerError, "failed to register user")
		}
		return
	}

	writeJSON(w, http.StatusCreated, userResponse{
		ID: u.ID, Name: u.Name, Username: u.Username, Email: u.Email,
		SystemRole: u.SystemRoleName, Status: u.Status,
	})
}

type loginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type loginResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
}

// Login godoc
// @Summary      Log in with username or email
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body loginRequest true "Login payload"
// @Success      200 {object} loginResponse
// @Failure      401 {object} map[string]string
// @Router       /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	token, expiresAt, err := h.svc.Login(r.Context(), service.LoginInput{Identifier: req.Identifier, Password: req.Password})
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid username/email or password")
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{Token: token, ExpiresAt: expiresAt.Format(timeFormat)})
}
