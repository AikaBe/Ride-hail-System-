package model

import (
	"encoding/json"
	"time"
)

type User struct {
	ID           UUID            `json:"id" db:"id"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at" db:"updated_at"`
	Email        string          `json:"email" db:"email"`
	Role         Role            `json:"role" db:"role"`
	Status       UserStatus      `json:"status" db:"status"`
	PasswordHash string          `json:"password_hash" db:"password_hash"`
	Attrs        json.RawMessage `json:"attrs" db:"attrs"`
}

type Coordinate struct {
	ID             UUID       `json:"id" db:"id"`
	CreatedAt      time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at" db:"updated_at"`
	EntityID       UUID       `json:"entity_id" db:"entity_id"`
	EntityType     EntityType `json:"entity_type" db:"entity_type"`
	Address        string     `json:"address" db:"address"`
	Latitude       float64    `json:"latitude" db:"latitude"`
	Longitude      float64    `json:"longitude" db:"longitude"`
	FareAmount     *float64   `json:"fare_amount,omitempty" db:"fare_amount"`
	DistanceKm     *float64   `json:"distance_km,omitempty" db:"distance_km"`
	DurationMinute *int       `json:"duration_minutes,omitempty" db:"duration_minutes"`
	IsCurrent      bool       `json:"is_current" db:"is_current"`
}

type Ride struct {
	ID                      UUID         `json:"id" db:"id"`
	CreatedAt               time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt               time.Time    `json:"updated_at" db:"updated_at"`
	RideNumber              string       `json:"ride_number" db:"ride_number"`
	PassengerID             UUID         `json:"passenger_id" db:"passenger_id"`
	DriverID                *UUID        `json:"driver_id,omitempty" db:"driver_id"`
	VehicleType             *VehicleType `json:"vehicle_type,omitempty" db:"vehicle_type"`
	Status                  *RideStatus  `json:"status,omitempty" db:"status"`
	Priority                int          `json:"priority" db:"priority"`
	RequestedAt             time.Time    `json:"requested_at" db:"requested_at"`
	MatchedAt               *time.Time   `json:"matched_at,omitempty" db:"matched_at"`
	ArrivedAt               *time.Time   `json:"arrived_at,omitempty" db:"arrived_at"`
	StartedAt               *time.Time   `json:"started_at,omitempty" db:"started_at"`
	CompletedAt             *time.Time   `json:"completed_at,omitempty" db:"completed_at"`
	CancelledAt             *time.Time   `json:"cancelled_at,omitempty" db:"cancelled_at"`
	CancellationReason      *string      `json:"cancellation_reason,omitempty" db:"cancellation_reason"`
	EstimatedFare           *float64     `json:"estimated_fare,omitempty" db:"estimated_fare"`
	FinalFare               *float64     `json:"final_fare,omitempty" db:"final_fare"`
	PickupCoordinateID      *UUID        `json:"pickup_coordinate_id,omitempty" db:"pickup_coordinate_id"`
	DestinationCoordinateID *UUID        `json:"destination_coordinate_id,omitempty" db:"destination_coordinate_id"`
}

type RideEvent struct {
	ID        UUID            `json:"id" db:"id"`
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
	RideID    string          `json:"ride_id" db:"ride_id"`
	EventType RideEventType   `json:"event_type" db:"event_type"`
	EventData json.RawMessage `json:"event_data" db:"event_data"`
}
