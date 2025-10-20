package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"ride-hail/internal/common/logger"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte("super-secret-key")

type Claims struct {
	UserID string `json:"sub"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func GetTokenHandler() http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")

		if r.Method != http.MethodPost {
			logger.Info("invalid_method", "Only POST allowed", requestID, "")
			http.Error(w, "only POST method allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			UserID string `json:"user_id"`
			Role   string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("decode_failed", "Failed to decode request body", requestID, "", err.Error(), "")
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.UserID == "" {
			logger.Info("missing_user_id", "user_id is required", requestID, "")
			http.Error(w, "user_id is required", http.StatusBadRequest)
			return
		}
		if req.Role == "" {
			req.Role = "PASSENGER"
		}

		token, err := GenerateToken(req.UserID, req.Role)
		if err != nil {
			logger.Error("token_generation_failed", "Failed to generate token", requestID, req.UserID, err.Error(), "")
			http.Error(w, "failed to generate token", http.StatusInternalServerError)
			return
		}
		logger.Info("token_generated", "Token successfully generated", requestID, req.UserID)

		json.NewEncoder(w).Encode(map[string]string{
			"token": token,
		})
	}
}

func GenerateToken(userID, role string) (string, error) {
	claims := Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)), // 1 час
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func ValidateToken(tokenString string) (*Claims, error) {
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
