package model

import "encoding/json"

type RegisterRequest struct {
	Name          string          `json:"name"`
	Email         string          `json:"email"`
	Password      string          `json:"password"`
	Role          string          `json:"role"`
	LicenseNumber string          `json:"license_number,omitempty"`
	VehicleType   string          `json:"vehicle_type,omitempty"`
	VehicleAttrs  json.RawMessage `json:"vehicle_attrs,omitempty"`
}

type RegisterResponse struct {
	UserID string `json:"user_id"`
	Token  string `json:"token"`
}
