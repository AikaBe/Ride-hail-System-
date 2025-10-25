package dto

import (
	usermodel "ride-hail/internal/user/model"
	"ride-hail/pkg/uuid"
)

type OnlineRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type OnlineResponse struct {
	Status    usermodel.DriverStatus `json:"status"`
	SessionID string                 `json:"session_id"`
	Message   string                 `json:"message"`
}

type OfflineResponse struct {
	Status         usermodel.DriverStatus `json:"status"`
	SessionID      string                 `json:"session_id"`
	SessionSummary SessionSummary         `json:"session_summary"`
	Message        string                 `json:"message"`
}

type SessionSummary struct {
	DurationHours  float64 `json:"duration_hours"`
	RidesCompleted int     `json:"rides_completed"`
	Earnings       float64 `json:"earnings"`
}

type LocationRequest struct {
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	AccuracyMeters float64 `json:"accuracy_meters"`
	SpeedKmh       float64 `json:"speed_kmh"`
	HeadingDegrees float64 `json:"heading_degrees"`
}

type LocationResponse struct {
	CoordinateID string `json:"coordinate_id"`
	UpdatedAt    string `json:"updated_at"`
}

type StartRequest struct {
	RideID         string `json:"ride_id"`
	DriverLocation struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"driver_location"`
}
type StartResponse struct {
	RideID    string                 `json:"ride_id"`
	Status    usermodel.DriverStatus `json:"status"`
	StartedAt string                 `json:"started_at"`
	Message   string                 `json:"message"`
}

type CompleteRequest struct {
	RideID        uuid.UUID `json:"ride_id"`
	FinalLocation struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"final_location"`
	ActualDistanceKm   float64 `json:"actual_distance_km"`
	ActualDurationMins float64 `json:"actual_duration_minutes"`
}

type CompleteResponse struct {
	RideID        string                 `json:"ride_id"`
	Status        usermodel.DriverStatus `json:"status"`
	CompletedAt   string                 `json:"completed_at"`
	DriverEarning float64                `json:"driver_earning"`
	Message       string                 `json:"message"`
}
