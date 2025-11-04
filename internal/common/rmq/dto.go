package rmq

import (
	"time"

	usermodel "ride-hail-system/internal/user/model"
)

type RideRequestedMessage struct {
	RideID              string                `json:"ride_id"`
	RideNumber          string                `json:"ride_number"`
	PickupLocation      Location              `json:"pickup_location"`
	DestinationLocation Location              `json:"destination_location"`
	RideType            usermodel.VehicleType `json:"ride_type"`
	EstimatedFare       float64               `json:"estimated_fare"`
	MaxDistanceKm       float64               `json:"max_distance_km"`
	TimeoutSeconds      int                   `json:"timeout_seconds"`
	CorrelationID       string                `json:"correlation_id"`
}

type Location struct {
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	Address string  `json:"address"`
}

type LocationUpdateMessage struct {
	DriverID  string    `json:"driver_id"`
	RideID    string    `json:"ride_id"`
	Location  LatLng    `json:"location"`
	SpeedKmh  float64   `json:"speed_kmh"`
	Heading   float64   `json:"heading_degrees"`
	Timestamp time.Time `json:"timestamp"`
}

type LatLng struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type DriverResponseMessage struct {
	RideID                  string     `json:"ride_id"`
	OfferID                 string     `json:"offer_id"`
	DriverID                string     `json:"driver_id"`
	Accepted                bool       `json:"accepted"`
	EstimatedArrivalMinutes int        `json:"estimated_arrival_minutes"`
	DriverLocation          LatLng     `json:"driver_location"`
	DriverInfo              DriverInfo `json:"driver_info"`
	EstimatedArrival        time.Time  `json:"estimated_arrival"`
	RespondedAt             time.Time  `json:"responded_at"`
}

type DriverInfo struct {
	Rating  float64 `json:"rating"`
	Vehicle Vehicle `json:"vehicle"`
}

type Vehicle struct {
	Year  int    `json:"year"`
	Model string `json:"uuid"`
	Color string `json:"color"`
	Brand string `json:"brand"`
}

type DriverLocationUpdateMessage struct {
	Type             string    `json:"type"`
	RideID           string    `json:"ride_id"`
	DriverLocation   LatLng    `json:"driver_location"`
	EstimatedArrival time.Time `json:"estimated_arrival"`
	DistanceToPickup float64   `json:"distance_to_pickup_km"`
}

type RideStatusUpdateMessage struct {
	Type    string `json:"type"`
	RideID  string `json:"ride_id"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type PassiNFO struct {
	Type           string         `json:"type"`            // тип сообщения, например "ride_details"
	RideID         string         `json:"ride_id"`         // ID поездки
	PassengerName  string         `json:"passenger_name"`  // имя пассажира
	PassengerPhone string         `json:"passenger_phone"` // телефон пассажира
	PickupLocation PickupLocation `json:"pickup_location"` // место посадки
}

type PickupLocation struct {
	Latitude  float64 `json:"latitude"`  // широта
	Longitude float64 `json:"longitude"` // долгота
	Address   string  `json:"address"`   // адрес
	Notes     string  `json:"notes"`     // примечания (например, "у главного входа")
}
