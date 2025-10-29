package model

import (
	"ride-hail/pkg/uuid"
	"time"
)

type DriverSession struct {
	ID            uuid.UUID  `db:"id" json:"id"`
	DriverID      uuid.UUID  `db:"driver_id" json:"driver_id"`
	StartedAt     time.Time  `db:"started_at" json:"started_at"`
	EndedAt       *time.Time `db:"ended_at" json:"ended_at,omitempty"`
	TotalRides    int        `db:"total_rides" json:"total_rides"`
	TotalEarnings float64    `db:"total_earnings" json:"total_earnings"`
}

type LocationHistory struct {
	ID             uuid.UUID `db:"id" json:"id"`
	CoordinateID   uuid.UUID `db:"coordinate_id" json:"coordinate_id,omitempty"`
	DriverID       uuid.UUID `db:"driver_id" json:"driver_id,omitempty"`
	Latitude       float64   `db:"latitude" json:"latitude"`
	Longitude      float64   `db:"longitude" json:"longitude"`
	AccuracyMeters float64   `db:"accuracy_meters" json:"accuracy_meters,omitempty"`
	SpeedKmh       float64   `db:"speed_kmh" json:"speed_kmh,omitempty"`
	HeadingDegrees float64   `db:"heading_degrees" json:"heading_degrees,omitempty"`
	RecordedAt     time.Time `db:"recorded_at" json:"recorded_at"`
	RideID         uuid.UUID `db:"ride_id" json:"ride_id,omitempty"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type DriverNearby struct {
	ID        string  `json:"id"`
	Email     string  `json:"email"`
	Rating    float64 `json:"rating"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Distance  float64 `json:"distance_km"`
}
type DriverResponceWS struct {
	Type            string `json:"type"`     // тип сообщения (например, "ride_response")
	OfferID         string `json:"offer_id"` // ID предложения
	RideID          string `json:"ride_id"`  // ID поездки
	DriverID        string `json:"driver_id"`
	Accepted        bool   `json:"accepted"` // принял ли водитель заказ
	CurrentLocation struct {
		Latitude  float64 `json:"latitude"`  // широта
		Longitude float64 `json:"longitude"` // долгота
	} `json:"current_location"`
}

type DriverInfo struct {
	Rating  float64 `json:"rating"`
	Vehicle Vehicle `json:"vehicle"`
}

type Vehicle struct {
	Year  int    `json:"year"`
	Model string `json:"model"`
	Color string `json:"color"`
	Brand string `json:"brand"`
}
