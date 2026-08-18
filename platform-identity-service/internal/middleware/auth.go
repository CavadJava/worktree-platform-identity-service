package middleware

import (
	"context"
	"net/http"
	"strings"

	"platform-identity-service/internal/auth"
	"platform-identity-service/internal/service/roleassign"
)

type contextKey string

const callerKey contextKey = "caller"

// RequireAuth verifies the token locally against this service's own
// JWTManager — no remote authorization-service call, since this service
// is meant to be a self-contained identity provider for new projects.
func RequireAuth(jwt *auth.JWTManager) func(http.Handler) http.Handler {
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

			caller := roleassign.Caller{UserID: claims.UserID, ProjectID: claims.ProjectID, Role: claims.Role}
			ctx := context.WithValue(r.Context(), callerKey, caller)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func CallerFromContext(ctx context.Context) (roleassign.Caller, bool) {
	caller, ok := ctx.Value(callerKey).(roleassign.Caller)
	return caller, ok
}

func writeAuthError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(`{"success":false,"error":{"code":"` + code + `","message":"` + message + `"}}`))
}
