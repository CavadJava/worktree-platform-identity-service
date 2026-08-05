package handlers

import (
	"net/http"
	"strings"

	"authorization-service/internal/auth"
)

type AuthorizeHandler struct {
	jwt *auth.JWTManager
}

func NewAuthorizeHandler(jwt *auth.JWTManager) *AuthorizeHandler {
	return &AuthorizeHandler{jwt: jwt}
}

type authorizeResponse struct {
	UserID        string  `json:"user_id"`
	Role          string  `json:"role"`
	ShopID        *string `json:"shop_id,omitempty"`
	ShopRoleLevel int     `json:"shop_role_level"`
}

// Authorize godoc
// @Summary      Verify a bearer token
// @Description  Digər servislərin (məs. user-service) qorunan endpoint-lərə gələn sorğuları doğrulamaq üçün çağırdığı endpoint. Authorization başlığındakı JWT-ni yoxlayır və sahibinin user_id-sini qaytarır.
// @Tags         authorize
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} authorizeResponse
// @Failure      401 {object} map[string]string
// @Router       /authorize [post]
func (h *AuthorizeHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	header := r.Header.Get("Authorization")
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		writeError(w, http.StatusUnauthorized, "missing or invalid authorization header")
		return
	}

	claims, err := h.jwt.Verify(parts[1])
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired token")
		return
	}

	writeJSON(w, http.StatusOK, authorizeResponse{
		UserID:        claims.UserID,
		Role:          claims.Role,
		ShopID:        claims.ShopID,
		ShopRoleLevel: claims.ShopRoleLevel,
	})
}
