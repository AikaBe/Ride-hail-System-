package auth

//
//import (
//	"context"
//	"net/http"
//	"ride-hail/internal/common/logger"
//)
//
//type contextKey string
//
//const ClaimsContextKey = contextKey("claims")
//
//func AuthMiddleware(next http.Handler) http.Handler {
//	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		requestID := r.Header.Get("X-Request-ID")
//		// Получаем токен из заголовка Authorization
//		token := r.Header.Get("Authorization")
//		if token == "" {
//			logger.Info("auth_missing_token", "Authorization header required", requestID, "")
//			http.Error(w, "Authorization header required", http.StatusUnauthorized)
//			return
//		}
//
//		claims, err := ValidateToken(token)
//		if err != nil {
//			logger.Error("auth_invalid_token", "Invalid token", requestID, "", err.Error(), "")
//			http.Error(w, "invalid token", http.StatusUnauthorized)
//			return
//		}
//		logger.Debug("auth_token_validated", "Token successfully validated", requestID, jwt.UserID)
//
//		ctx := context.WithValue(r.Context(), ClaimsContextKey, claims)
//		next.ServeHTTP(w, r.WithContext(ctx))
//	})
//}
//
//func FromContext(r *http.Request) *Claims {
//	claims, ok := r.Context().Value(ClaimsContextKey).(*Claims)
//	if !ok {
//		return nil
//	}
//	return claims
//}
