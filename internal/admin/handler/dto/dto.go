package dto

// Response DTOs for admin endpoints
// These mirror the model structures but provide a clean API contract

type SystemOverviewResponse struct {
	Timestamp          string                `json:"timestamp"`
	Metrics            SystemMetricsResponse `json:"metrics"`
	DriverDistribution map[string]int        `json:"driver_distribution"`
	Hotspots           []HotspotResponse     `json:"hotspots"`
}

type SystemMetricsResponse struct {
	ActiveRides                int     `json:"active_rides"`
	AvailableDrivers           int     `json:"available_drivers"`
	BusyDrivers                int     `json:"busy_drivers"`
	TotalRidesToday            int     `json:"total_rides_today"`
	TotalRevenueToday          float64 `json:"total_revenue_today"`
	AverageWaitTimeMinutes     float64 `json:"average_wait_time_minutes"`
	AverageRideDurationMinutes float64 `json:"average_ride_duration_minutes"`
	CancellationRate           float64 `json:"cancellation_rate"`
}

type HotspotResponse struct {
	Location       string `json:"location"`
	ActiveRides    int    `json:"active_rides"`
	WaitingDrivers int    `json:"waiting_drivers"`
}

type ActiveRidesResponse struct {
	Rides      []ActiveRideResponse `json:"rides"`
	TotalCount int                  `json:"total_count"`
	Page       int                  `json:"page"`
	PageSize   int                  `json:"page_size"`
}

type ActiveRideResponse struct {
	RideID                string            `json:"ride_id"`
	RideNumber            string            `json:"ride_number"`
	Status                string            `json:"status"`
	PassengerID           string            `json:"passenger_id"`
	DriverID              *string           `json:"driver_id,omitempty"`
	PickupAddress         string            `json:"pickup_address"`
	DestinationAddress    string            `json:"destination_address"`
	StartedAt             string            `json:"started_at,omitempty"`
	CompletedAt           string            `json:"completed_at,omitempty"`
	CurrentDriverLocation *LocationResponse `json:"current_driver_location,omitempty"`
}

type OnlineDriverResponse struct {
	DriverID        string            `json:"driver_id"`
	Email           string            `json:"email"`
	Rating          float64           `json:"rating"`
	Status          string            `json:"status"`
	VehicleType     string            `json:"vehicle_type"`
	CurrentLocation *LocationResponse `json:"current_location,omitempty"`
	CurrentAddress  string            `json:"current_address,omitempty"`
	TotalRides      int               `json:"total_rides"`
	TotalEarnings   float64           `json:"total_earnings"`
}

type LocationResponse struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
