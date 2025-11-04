package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"ride-hail-system/internal/common/logger"
	"ride-hail-system/internal/user/handler/dto"
	"ride-hail-system/internal/user/model"
	"ride-hail-system/pkg/uuid"

	token "ride-hail-system/internal/user/jwt"

	"github.com/jackc/pgx/v5"
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
	action := "register_user"
	requestID := ctx.Value("request_id")
	if requestID == nil {
		requestID = "none"
	}

	logger.Info(action, "registration process started", fmt.Sprint(requestID), "")

	if err := req.Validate(); err != nil {
		logger.Warn(action, "validation failed", fmt.Sprint(requestID), "", err.Error())
		return model.User{}, fmt.Errorf("validation error: %w", err)
	}

	tx, err := s.userRepo.BeginTx(ctx)
	if err != nil {
		logger.Error(action, "failed to start transaction", fmt.Sprint(requestID), "", err.Error())
		return model.User{}, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
			logger.Warn(action, "rollback failed", fmt.Sprint(requestID), "", err.Error())
		}
	}()

	hash, err := hashPassword(req.Password)
	if err != nil {
		logger.Error(action, "failed to hash password", fmt.Sprint(requestID), "", err.Error())
		return model.User{}, err
	}

	userID, err := uuid.NewUUID()
	if err != nil {
		logger.Error(action, "failed to generate UUID", fmt.Sprint(requestID), "", err.Error())
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
		logger.Error(action, "failed to create user", fmt.Sprint(requestID), "", err.Error())
		return model.User{}, err
	}

	if req.Role == model.RoleDriver {
		logger.Debug(action, "creating driver profile", fmt.Sprint(requestID), string(createdUser.ID))

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
			logger.Error(action, "failed to create driver profile", fmt.Sprint(requestID), string(createdUser.ID), err.Error())
			return model.User{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		logger.Error(action, "failed to commit transaction", fmt.Sprint(requestID), string(createdUser.ID), err.Error())
		return model.User{}, fmt.Errorf("failed to commit transaction: %w", err)
	}

	logger.Info(action, "user successfully registered", fmt.Sprint(requestID), string(createdUser.ID))
	return createdUser, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, string, error) {
	action := "login_user"
	requestID := ctx.Value("request_id")
	if requestID == nil {
		requestID = "none"
	}

	logger.Info(action, fmt.Sprintf("login attempt for user: %s", email), fmt.Sprint(requestID), "")

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		logger.Error(action, "user not found", fmt.Sprint(requestID), "", err.Error())
		return "", "", fmt.Errorf("user not found: %w", err)
	}

	if !checkPassword(user.PasswordHash, password) {
		logger.Warn(action, "invalid credentials", fmt.Sprint(requestID), string(user.ID), "")
		return "", "", fmt.Errorf("invalid credentials")
	}

	access, refresh, err := s.jwtManager.GenerateTokens(string(user.ID), string(user.Role))
	if err != nil {
		logger.Error(action, "failed to generate tokens", fmt.Sprint(requestID), string(user.ID), err.Error())
		return "", "", err
	}

	logger.Info(action, "user successfully logged in", fmt.Sprint(requestID), string(user.ID))
	return access, refresh, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, req dto.RefreshTokenRequest) (dto.RefreshTokenResponse, error) {
	action := "refresh_token"
	requestID := ctx.Value("request_id")
	if requestID == nil {
		requestID = "none"
	}

	logger.Info(action, "refresh token process started", fmt.Sprint(requestID), "")

	claims, err := s.jwtManager.ParseToken(req.RefreshToken)
	if err != nil {
		logger.Error(action, "invalid refresh token", fmt.Sprint(requestID), "", err.Error())
		return dto.RefreshTokenResponse{}, fmt.Errorf("invalid refresh token: %w", err)
	}

	if claims.Type != "refresh" {
		logger.Warn(action, "provided token is not a refresh token", fmt.Sprint(requestID), claims.UserID, "")
		return dto.RefreshTokenResponse{}, fmt.Errorf("provided token is not a refresh token")
	}

	accessToken, err := s.jwtManager.GenerateAccessToken(claims.UserID, claims.Role)
	if err != nil {
		logger.Error(action, "failed to generate access token", fmt.Sprint(requestID), claims.UserID, err.Error())
		return dto.RefreshTokenResponse{}, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.jwtManager.GenerateRefreshToken(claims.UserID, claims.Role)
	if err != nil {
		logger.Error(action, "failed to generate refresh token", fmt.Sprint(requestID), claims.UserID, err.Error())
		return dto.RefreshTokenResponse{}, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	logger.Info(action, "tokens successfully refreshed", fmt.Sprint(requestID), claims.UserID)

	return dto.RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func hashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(append(salt, []byte(password)...))

	return fmt.Sprintf("%x:%x", salt, hash[:]), nil
}

func checkPassword(hashPassword, password string) bool {
	var saltHex, hashHex string
	_, err := fmt.Sscanf(hashPassword, "%32s:%64s", &saltHex, &hashHex)
	if err != nil {
		return false
	}

	salt, err := hex.DecodeString(saltHex)
	if err != nil {
		return false
	}
	storedHash, err := hex.DecodeString(hashHex)
	if err != nil {
		return false
	}

	hash := sha256.Sum256(append(salt, []byte(password)...))

	return subtle.ConstantTimeCompare(storedHash, hash[:]) == 1
}
