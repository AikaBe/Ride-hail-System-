package service

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"ride-hail/internal/user/handler/dto"
	token "ride-hail/internal/user/jwt"
	"ride-hail/internal/user/model"
	"ride-hail/pkg/uuid"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user model.User) (model.User, error)
	CreateDriver(ctx context.Context, driver model.Driver) (model.Driver, error)
	GetByEmail(ctx context.Context, email string) (model.User, error)
}
type AuthService struct {
	userRepo     UserRepository
	tokenManager *token.Manager
}

func NewAuthService(userRepo UserRepository) *AuthService {
	return &AuthService{userRepo: userRepo}
}

func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest) (model.User, error) {
	if err := req.Validate(); err != nil {
		return model.User{}, fmt.Errorf("validation error: %w", err)
	}

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

	createdUser, err := s.userRepo.CreateUser(ctx, user)
	if err != nil {
		return model.User{}, err
	}

	if req.Role == "DRIVER" {
		var vehicleAttrs map[string]any
		if len(req.VehicleAttrs) > 0 {
			_ = json.Unmarshal(req.VehicleAttrs, &vehicleAttrs)
		}

		driver := model.Driver{
			ID:            createdUser.ID,
			LicenseNumber: req.LicenseNumber,
			VehicleType:   req.VehicleType,
			VehicleAttrs:  vehicleAttrs,
			Status:        model.DriverStatusOffline,
			IsVerified:    false,
		}

		_, err := s.userRepo.CreateDriver(ctx, driver)
		if err != nil {
			return model.User{}, err
		}
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

	access, refresh, err := s.tokenManager.GenerateTokens(string(user.ID), string(user.Role))
	if err != nil {
		return "", "", err
	}

	return access, refresh, nil
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
