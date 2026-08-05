package middleware

import (
	"context"
	"errors"
	"net/http"

	"notification-service/internal/client"
)

type contextKey string

const userIDKey contextKey = "userID"

type authorizer interface {
	Authorize(ctx context.Context, authHeader string) (string, error)
}

// Auth delegates token verification to authorization-service instead of
// checking the JWT locally — this service never sees the signing secret.
func Auth(authClient authorizer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				writeAuthError(w, http.StatusUnauthorized, "unauthorized", "missing or invalid authorization header")
				return
			}

			userID, err := authClient.Authorize(r.Context(), header)
			if err != nil {
				if errors.Is(err, client.ErrUnauthorized) {
					writeAuthError(w, http.StatusUnauthorized, "unauthorized", "invalid or expired token")
					return
				}
				writeAuthError(w, http.StatusServiceUnavailable, "service_unavailable", "authorization service unavailable")
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDKey).(string)
	return userID, ok
}

func writeAuthError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(`{"success":false,"error":{"code":"` + code + `","message":"` + message + `"}}`))
}
