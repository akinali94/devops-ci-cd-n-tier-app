package middleware

import (
	"context"
	"encoding/json"
	"net/http"
)

type contextKey string

const UserIDKey contextKey = "user_id"

// AuthClient validates JWT tokens by calling auth-service.
type AuthClient struct {
	authServiceURL string
	httpClient     *http.Client
}

func NewAuthClient(authServiceURL string) *AuthClient {
	return &AuthClient{
		authServiceURL: authServiceURL,
		httpClient:     &http.Client{},
	}
}

// Authenticate is an HTTP middleware that validates the Bearer token by
// calling auth-service GET /validate and injects user_id into the context.
func (a *AuthClient) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
			return
		}

		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, a.authServiceURL+"/validate", nil)
		if err != nil {
			http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			return
		}
		req.Header.Set("Authorization", authHeader)

		resp, err := a.httpClient.Do(req)
		if err != nil {
			http.Error(w, `{"error":"auth service unavailable"}`, http.StatusServiceUnavailable)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var payload struct {
			UserID string `json:"user_id"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil || payload.UserID == "" {
			http.Error(w, `{"error":"invalid auth response"}`, http.StatusInternalServerError)
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, payload.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// UserIDFromContext extracts the user_id injected by the Authenticate middleware.
func UserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(UserIDKey).(string)
	return id, ok
}
