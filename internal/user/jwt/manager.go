package token

import (
	"github.com/golang-jwt/jwt/v5"
	"time"
)

type Manager struct {
	secretKey  string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewManager(secret string, accessTTL, refreshTTL time.Duration) *Manager {
	return &Manager{
		secretKey:  secret,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	Type   string `json:"type"` // "access" или "refresh"
	jwt.RegisteredClaims
}

func (m *Manager) GenerateAccessToken(userID, role string) (string, error) {
	claims := &Claims{
		UserID: userID,
		Role:   role,
		Type:   "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.accessTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(m.secretKey))
}

func (m *Manager) GenerateRefreshToken(userID, role string) (string, error) {
	claims := &Claims{
		UserID: userID,
		Role:   role,
		Type:   "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.refreshTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(m.secretKey))
}

func (m *Manager) GenerateTokens(userID, role string) (accessToken, refreshToken string, err error) {
	accessToken, err = m.GenerateAccessToken(userID, role)
	if err != nil {
		return "", "", err
	}

	refreshToken, err = m.GenerateRefreshToken(userID, role)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (m *Manager) ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(m.secretKey), nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrSignatureInvalid
}
