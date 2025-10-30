package dto

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"ride-hail/internal/user/model"
)

type RegisterRequest struct {
	Name          string            `json:"name"`
	Email         string            `json:"email"`
	Password      string            `json:"password"`
	Role          model.Role        `json:"role"`
	LicenseNumber string            `json:"license_number,omitempty"`
	VehicleType   model.VehicleType `json:"vehicle_type,omitempty"`
	VehicleAttrs  json.RawMessage   `json:"vehicle_attrs,omitempty"`
}

type RegisterResponse struct {
	UserID string `json:"user_id"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func (r *RegisterRequest) Validate() error {
	if strings.TrimSpace(r.Email) == "" {
		return errors.New("email is required")
	}
	if !isValidEmail(r.Email) {
		return errors.New("invalid email format")
	}

	if len(r.Password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}

	switch r.Role {
	case model.RoleDriver:
		if r.LicenseNumber == "" {
			return errors.New("license_number is required for drivers")
		}
	case model.RolePassenger, model.RoleAdmin:
		// ok
	default:
		return fmt.Errorf("unknown role: %s", r.Role)
	}

	return nil
}

func isValidEmail(email string) bool {
	re := regexp.MustCompile(`^[\w._%+\-]+@[\w.\-]+\.[A-Za-z]{2,}$`)
	return re.MatchString(email)
}
