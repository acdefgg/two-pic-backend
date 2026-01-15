package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"sync-photo-backend/internal/services"
)

type contextKey string

const userIDKey contextKey = "user_id"

// AuthMiddleware creates a middleware for JWT authentication
func AuthMiddleware(userService *services.UserService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondError(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				respondError(w, "Invalid authorization header format", http.StatusUnauthorized)
				return
			}

			token := parts[1]
			userID, err := userService.ValidateJWT(token)
			if err != nil {
				respondError(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID extracts user ID from context
func GetUserID(ctx context.Context) string {
	userID, ok := ctx.Value(userIDKey).(string)
	if !ok {
		return ""
	}
	return userID
}

// respondError sends an error response
func respondError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write([]byte(`{"error":"` + message + `"}`))
}

// ValidateWebSocketToken validates JWT token from WebSocket query parameter
func ValidateWebSocketToken(token string, userService *services.UserService) (string, error) {
	if token == "" {
		return "", fmt.Errorf("token required")
	}
	return userService.ValidateJWT(token)
}
