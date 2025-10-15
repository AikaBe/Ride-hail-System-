package auth

import (
	"context"
	"net/http"
)

type contextKey string

const ClaimsContextKey = contextKey("claims")

// AuthMiddleware проверяет JWT на каждом запросе
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Получаем токен из заголовка Authorization
		token := r.Header.Get("Authorization")
		if token == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		claims, err := ValidateToken(token)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func FromContext(r *http.Request) *Claims {
	claims, ok := r.Context().Value(ClaimsContextKey).(*Claims)
	if !ok {
		return nil
	}
	return claims
}
