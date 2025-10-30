package model

import (
	"time"

	"ride-hail/pkg/uuid"
)

type DriverStatus string

const (
	DriverStatusOffline   DriverStatus = "OFFLINE"
	DriverStatusAvailable DriverStatus = "AVAILABLE"
	DriverStatusBusy      DriverStatus = "BUSY"
	DriverStatusEnRoute   DriverStatus = "EN_ROUTE"
)

type VehicleType string

const (
	VehicleEconomy VehicleType = "ECONOMY"
	VehiclePremium VehicleType = "PREMIUM"
	VehicleXL      VehicleType = "XL"
)

type Driver struct {
	ID            uuid.UUID      `db:"id" json:"id"`
	CreatedAt     time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt     time.Time      `db:"updated_at" json:"updated_at"`
	LicenseNumber string         `db:"license_number" json:"license_number"`
	VehicleType   VehicleType    `db:"vehicle_type" json:"vehicle_type,omitempty"`
	VehicleAttrs  map[string]any `db:"vehicle_attrs" json:"vehicle_attrs,omitempty"` // jsonb
	Rating        float64        `db:"rating" json:"rating"`
	TotalRides    int            `db:"total_rides" json:"total_rides"`
	TotalEarnings float64        `db:"total_earnings" json:"total_earnings"`
	Status        DriverStatus   `db:"status" json:"status,omitempty"`
	IsVerified    bool           `db:"is_verified" json:"is_verified"`
}
