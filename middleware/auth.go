package middleware

import (
	"context"
	"net/http"
	"strings"
)

// ContextKey type for context keys
type ContextKey string

const (
	// UserIDKey is the context key for user ID
	UserIDKey ContextKey = "user_id"
	// UsernameKey is the context key for username
	UsernameKey ContextKey = "username"
)

// AuthService interface for auth validation
type AuthService interface {
	ValidateToken(token string) (userID int64, username string, err error)
}

// Auth middleware validates JWT tokens
func Auth(authService AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Missing authorization header", http.StatusUnauthorized)
				return
			}

			// Check Bearer prefix
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
				return
			}

			token := parts[1]

			// Validate token
			userID, username, err := authService.ValidateToken(token)
			if err != nil {
				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			// Add user info to context
			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			ctx = context.WithValue(ctx, UsernameKey, username)

			// Call next handler
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuth middleware validates JWT tokens but doesn't reject requests without tokens
func OptionalAuth(authService AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 && parts[0] == "Bearer" {
					token := parts[1]
					userID, username, err := authService.ValidateToken(token)
					if err == nil {
						ctx := context.WithValue(r.Context(), UserIDKey, userID)
						ctx = context.WithValue(ctx, UsernameKey, username)
						r = r.WithContext(ctx)
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetUserID extracts user ID from request context
func GetUserID(r *http.Request) (int64, bool) {
	userID, ok := r.Context().Value(UserIDKey).(int64)
	return userID, ok
}

// GetUsername extracts username from request context
func GetUsername(r *http.Request) (string, bool) {
	username, ok := r.Context().Value(UsernameKey).(string)
	return username, ok
}
