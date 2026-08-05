package middleware

import (
	"context"
	"errors"
	"net/http"

	"registration-service/internal/client"
	"registration-service/internal/roles"
)

type contextKey string

const identityKey contextKey = "identity"

type authorizer interface {
	Authorize(ctx context.Context, authHeader string) (*client.Identity, error)
}

// RequireAdministrator verifies the token via authorization-service and
// rejects anyone whose base role isn't "administrator" — used to gate the
// admin-only user listing endpoints.
func RequireAdministrator(authClient authorizer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				writeAuthError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid authorization header")
				return
			}

			identity, err := authClient.Authorize(r.Context(), header)
			if err != nil {
				if errors.Is(err, client.ErrUnauthorized) {
					writeAuthError(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
					return
				}
				writeAuthError(w, http.StatusServiceUnavailable, "service_unavailable", "authorization service unavailable")
				return
			}

			if identity.Role != roles.RoleAdministrator {
				writeAuthError(w, http.StatusForbidden, "forbidden", "administrator role required")
				return
			}

			ctx := context.WithValue(r.Context(), identityKey, identity)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func IdentityFromContext(ctx context.Context) (*client.Identity, bool) {
	identity, ok := ctx.Value(identityKey).(*client.Identity)
	return identity, ok
}

func writeAuthError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(`{"success":false,"error":{"code":"` + code + `","message":"` + message + `"}}`))
}
