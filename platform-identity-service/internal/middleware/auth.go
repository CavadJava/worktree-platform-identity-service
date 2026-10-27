package middleware

import (
	"context"
	"net/http"
	"strings"

	"platform-identity-service/internal/auth"
	"platform-identity-service/internal/models"
	"platform-identity-service/internal/repository"
	"platform-identity-service/internal/service/shopassign"
)

type contextKey string

const callerKey contextKey = "caller"
const claimsKey contextKey = "claims"

// RequireAuth verifies the token locally, then re-checks the caller's
// current status in the database — a status change (e.g. an admin
// deactivating a user) must take effect on the user's very next request,
// not just at their next login, so this cannot rely on the JWT's claims
// alone.
func RequireAuth(jwt *auth.JWTManager, userRepo *repository.UserRepository, revocationRepo *repository.TokenRevocationRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			token := strings.TrimPrefix(header, "Bearer ")
			if header == "" || token == header {
				writeAuthError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid authorization header")
				return
			}

			claims, err := jwt.Verify(token)
			if err != nil {
				writeAuthError(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
				return
			}

			revoked, err := revocationRepo.IsRevoked(r.Context(), claims.JTI)
			if err != nil {
				writeAuthError(w, http.StatusInternalServerError, "internal_error", "failed to verify token")
				return
			}
			if revoked {
				writeAuthError(w, http.StatusUnauthorized, "unauthorized", "token has been revoked")
				return
			}

			user, err := userRepo.GetByID(r.Context(), claims.UserID)
			if err != nil {
				writeAuthError(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
				return
			}
			if user.Status != models.UserStatusActive {
				writeAuthError(w, http.StatusUnauthorized, "unauthorized", "account is deactivated")
				return
			}

			caller := shopassign.Caller{UserID: claims.UserID, SystemRole: claims.SystemRole}
			ctx := context.WithValue(r.Context(), callerKey, caller)
			ctx = context.WithValue(ctx, claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func CallerFromContext(ctx context.Context) (shopassign.Caller, bool) {
	caller, ok := ctx.Value(callerKey).(shopassign.Caller)
	return caller, ok
}

func ClaimsFromContext(ctx context.Context) (*auth.Claims, bool) {
	claims, ok := ctx.Value(claimsKey).(*auth.Claims)
	return claims, ok
}

func writeAuthError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(`{"success":false,"error":{"code":"` + code + `","message":"` + message + `"}}`))
}
