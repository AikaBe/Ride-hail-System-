package model

import "time"

type SystemOverview struct {
	Timestamp          time.Time      `json:"timestamp"`
	Metrics            SystemMetrics  `json:"metrics"`
	DriverDistribution map[string]int `json:"driver_distribution"`
	Hotspots           []Hotspot      `json:"hotspots"`
}

type SystemMetrics struct {
	ActiveRides                int     `json:"active_rides"`
	AvailableDrivers           int     `json:"available_drivers"`
	BusyDrivers                int     `json:"busy_drivers"`
	TotalRidesToday            int     `json:"total_rides_today"`
	TotalRevenueToday          float64 `json:"total_revenue_today"`
	AverageWaitTimeMinutes     float64 `json:"average_wait_time_minutes"`
	AverageRideDurationMinutes float64 `json:"average_ride_duration_minutes"`
	CancellationRate           float64 `json:"cancellation_rate"`
}

type Hotspot struct {
	Location       string `json:"location"`
	ActiveRides    int    `json:"active_rides"`
	WaitingDrivers int    `json:"waiting_drivers"`
}

type ActiveRidesResponse struct {
	Rides      []ActiveRide `json:"rides"`
	TotalCount int          `json:"total_count"`
	Page       int          `json:"page"`
	PageSize   int          `json:"page_size"`
}

type ActiveRide struct {
	RideID                string    `json:"ride_id"`
	RideNumber            string    `json:"ride_number"`
	Status                string    `json:"status"`
	PassengerID           string    `json:"passenger_id"`
	DriverID              *string   `json:"driver_id,omitempty"`
	PickupAddress         string    `json:"pickup_address"`
	DestinationAddress    string    `json:"destination_address"`
	StartedAt             string    `json:"started_at,omitempty"`
	CompletedAt           string    `json:"completed_at,omitempty"`
	CurrentDriverLocation *Location `json:"current_driver_location,omitempty"`
	DistanceCompletedKm   float64   `json:"distance_completed_km,omitempty"`
	DistanceRemainingKm   float64   `json:"distance_remaining_km,omitempty"`
}

type OnlineDriver struct {
	DriverID        string    `json:"driver_id"`
	Email           string    `json:"email"`
	Rating          float64   `json:"rating"`
	Status          string    `json:"status"`
	VehicleType     string    `json:"vehicle_type"`
	CurrentLocation *Location `json:"current_location,omitempty"`
	CurrentAddress  string    `json:"current_address,omitempty"`
	TotalRides      int       `json:"total_rides"`
	TotalEarnings   float64   `json:"total_earnings"`
}

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
