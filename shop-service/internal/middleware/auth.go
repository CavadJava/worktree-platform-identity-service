package middleware

import (
	"context"
	"errors"
	"net/http"

	"shop-service/internal/client"
	"shop-service/internal/roles"
)

type contextKey string

const identityKey contextKey = "identity"

type authorizer interface {
	Authorize(ctx context.Context, authHeader string) (*client.Identity, error)
}

// RequireAuth verifies the token via authorization-service and stores the
// caller's identity in the request context. Fine-grained checks (can this
// identity create/edit/delete this specific shop) happen in the service
// layer, since they depend on the resource being acted on.
func RequireAuth(authClient authorizer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity, err := authorize(r, authClient)
			if err != nil {
				writeAuthErr(w, err)
				return
			}
			ctx := context.WithValue(r.Context(), identityKey, identity)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireAdministrator additionally rejects anyone whose base role isn't
// "administrator" — used for shop-application review/approval and direct
// shop creation.
func RequireAdministrator(authClient authorizer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity, err := authorize(r, authClient)
			if err != nil {
				writeAuthErr(w, err)
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

func authorize(r *http.Request, authClient authorizer) (*client.Identity, error) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return nil, client.ErrUnauthorized
	}
	return authClient.Authorize(r.Context(), header)
}

func writeAuthErr(w http.ResponseWriter, err error) {
	if errors.Is(err, client.ErrUnauthorized) {
		writeAuthError(w, http.StatusUnauthorized, "unauthorized", "missing, invalid or expired token")
		return
	}
	writeAuthError(w, http.StatusServiceUnavailable, "service_unavailable", "authorization service unavailable")
}

func writeAuthError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(`{"success":false,"error":{"code":"` + code + `","message":"` + message + `"}}`))
}

func IdentityFromContext(ctx context.Context) (*client.Identity, bool) {
	identity, ok := ctx.Value(identityKey).(*client.Identity)
	return identity, ok
}
