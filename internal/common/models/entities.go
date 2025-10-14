package model

import "time"

type Ride struct {
	ID                     string     `json:"ride_id" db:"id"`
	RideNumber             string     `json:"ride_number" db:"ride_number"`
	PassengerID            string     `json:"passenger_id" db:"passenger_id"`
	DriverID               *string    `json:"driver_id,omitempty" db:"driver_id"`
	VehicleType            string     `json:"ride_type" db:"vehicle_type"`
	Status                 string     `json:"status" db:"status"`
	RequestedAt            time.Time  `json:"requested_at" db:"requested_at"`
	CancelledAt            *time.Time `json:"cancelled_at,omitempty" db:"cancelled_at"`
	CancellationReason     *string    `json:"cancellation_reason,omitempty" db:"cancellation_reason"`
	EstimatedFare          float64    `json:"estimated_fare" db:"estimated_fare"`
	EstimatedDistanceKm    float64    `json:"estimated_distance_km" db:"-"` 
	EstimatedDurationMin   float64    `json:"estimated_duration_minutes" db:"-"` 
	PickupCoordinateID     string     `json:"pickup_coordinate_id" db:"pickup_coordinate_id"`
	DestinationCoordinateID string    `json:"destination_coordinate_id" db:"destination_coordinate_id"`
	CreatedAt              time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at" db:"updated_at"`
}
type Coordinate struct {
	ID          string    `json:"id" db:"id"`
	EntityID    string    `json:"entity_id" db:"entity_id"`
	EntityType  string    `json:"entity_type" db:"entity_type"`
	Address     string    `json:"address" db:"address"`
	Latitude    float64   `json:"latitude" db:"latitude"`
	Longitude   float64   `json:"longitude" db:"longitude"`
	IsCurrent   bool      `json:"is_current" db:"is_current"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}
type CreateRideRequest struct {
	PassengerID          string  `json:"passenger_id"`
	PickupLatitude       float64 `json:"pickup_latitude"`
	PickupLongitude      float64 `json:"pickup_longitude"`
	PickupAddress        string  `json:"pickup_address"`
	DestinationLatitude  float64 `json:"destination_latitude"`
	DestinationLongitude float64 `json:"destination_longitude"`
	DestinationAddress   string  `json:"destination_address"`
	RideType             string  `json:"ride_type"`
}

// --- Запрос при создании поездки ---
type RideRequest struct {
	PassengerID          string  `json:"passenger_id"`
	PickupLatitude       float64 `json:"pickup_latitude"`
	PickupLongitude      float64 `json:"pickup_longitude"`
	PickupAddress        string  `json:"pickup_address"`
	DestinationLatitude  float64 `json:"destination_latitude"`
	DestinationLongitude float64 `json:"destination_longitude"`
	DestinationAddress   string  `json:"destination_address"`
	RideType             string  `json:"ride_type"`
}

// --- Ответ при успешном создании поездки ---
type RideResponse struct {
	RideID                   string  `json:"ride_id"`
	RideNumber               string  `json:"ride_number"`
	Status                   string  `json:"status"`
	EstimatedFare            float64 `json:"estimated_fare"`
	EstimatedDurationMinutes float64 `json:"estimated_duration_minutes"`
	EstimatedDistanceKm      float64 `json:"estimated_distance_km"`
}

// --- Запрос на отмену поездки ---
type CancelRideRequest struct {
	Reason string `json:"reason"`
}

// --- Ответ после отмены поездки ---
type CancelRideResponse struct {
	RideID      string `json:"ride_id"`
	Status      string `json:"status"`
	CancelledAt string `json:"cancelled_at"`
	Message     string `json:"message"`
}
