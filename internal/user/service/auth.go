package service

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
	"ride-hail/internal/user/handler/dto"
	token "ride-hail/internal/user/jwt"
	"ride-hail/internal/user/model"
	"ride-hail/pkg/uuid"
)

type UserRepository interface {
	CreateUser(ctx context.Context, tx pgx.Tx, user model.User) (model.User, error)
	CreateDriver(ctx context.Context, tx pgx.Tx, driver model.Driver) (model.Driver, error)
	GetByEmail(ctx context.Context, email string) (model.User, error)

	BeginTx(ctx context.Context) (pgx.Tx, error)
}
type AuthService struct {
	userRepo   UserRepository
	jwtManager *token.Manager
}

func NewAuthService(userRepo UserRepository, tokenManager *token.Manager) *AuthService {
	return &AuthService{userRepo: userRepo, jwtManager: tokenManager}
}

func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest) (model.User, error) {
	if err := req.Validate(); err != nil {
		return model.User{}, fmt.Errorf("validation error: %w", err)
	}

	tx, err := s.userRepo.BeginTx(ctx)
	if err != nil {
		return model.User{}, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
			fmt.Printf("rollback failed: %v\n", err)
		}
	}()

	hash, _ := hashPassword(req.Password)
	userID, err := uuid.NewUUID()
	if err != nil {
		return model.User{}, err
	}
	user := model.User{
		ID:           uuid.UUID(userID),
		Email:        req.Email,
		PasswordHash: hash,
		Role:         req.Role,
		Status:       model.UserActive,
		Attrs:        json.RawMessage(fmt.Sprintf(`{"name":"%s"}`, req.Name)),
	}

	createdUser, err := s.userRepo.CreateUser(ctx, tx, user)
	if err != nil {
		return model.User{}, err
	}

	if req.Role == model.RoleDriver {
		var vehicleAttrs map[string]any
		if len(req.VehicleAttrs) > 0 {
			_ = json.Unmarshal(req.VehicleAttrs, &vehicleAttrs)
		}

		driver := model.Driver{
			ID:            createdUser.ID,
			LicenseNumber: req.LicenseNumber,
			VehicleType:   req.VehicleType,
			VehicleAttrs:  vehicleAttrs,
			Rating:        5,
			Status:        model.DriverStatusOffline,
			IsVerified:    false,
		}

		_, err := s.userRepo.CreateDriver(ctx, tx, driver)
		if err != nil {
			return model.User{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return model.User{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return createdUser, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, string, error) {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", "", fmt.Errorf("user not found: %w", err)
	}

	if !checkPassword(user.PasswordHash, password) {
		return "", "", fmt.Errorf("invalid credentials")
	}

	access, refresh, err := s.jwtManager.GenerateTokens(string(user.ID), string(user.Role))
	if err != nil {
		return "", "", err
	}

	return access, refresh, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, req dto.RefreshTokenRequest) (dto.RefreshTokenResponse, error) {
	claims, err := s.jwtManager.ParseToken(req.RefreshToken)
	if err != nil {
		return dto.RefreshTokenResponse{}, fmt.Errorf("invalid refresh token: %w", err)
	}

	if claims.Type != "refresh" {
		return dto.RefreshTokenResponse{}, fmt.Errorf("provided token is not a refresh token")
	}

	accessToken, err := s.jwtManager.GenerateAccessToken(claims.UserID, claims.Role)
	if err != nil {
		return dto.RefreshTokenResponse{}, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(claims.UserID, claims.Role)
	if err != nil {
		return dto.RefreshTokenResponse{}, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return dto.RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func checkPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
