package model

import (
	"encoding/json"
	"time"

	"ride-hail/pkg/uuid"
)

type EntityType string

const (
	EntityTypeDriver    EntityType = "driver"
	EntityTypePassenger EntityType = "passenger"
)

type Role string

const (
	RolePassenger Role = "PASSENGER"
	RoleDriver    Role = "DRIVER"
	RoleAdmin     Role = "ADMIN"
)

type UserStatus string

const (
	UserActive   UserStatus = "ACTIVE"
	UserInactive UserStatus = "INACTIVE"
	UserBanned   UserStatus = "BANNED"
)

type User struct {
	ID           uuid.UUID       `json:"id" db:"id"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at" db:"updated_at"`
	Email        string          `json:"email" db:"email"`
	Role         Role            `json:"role" db:"role"`
	Status       UserStatus      `json:"status" db:"status"`
	PasswordHash string          `json:"password_hash" db:"password_hash"`
	Attrs        json.RawMessage `json:"attrs" db:"attrs"`
}
