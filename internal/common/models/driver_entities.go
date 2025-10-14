package models

type OnlineRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type OnlineResponse struct {
	Status    string `json:"status"`
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

type OfflineResponse struct {
	Status         string         `json:"status"`
	SessionID      string         `json:"session_id"`
	SessionSummary SessionSummary `json:"session_summary"`
	Message        string         `json:"message"`
}

type SessionSummary struct {
	DurationHours  float64 `json:"duration_hours"`
	RidesCompleted float64 `json:"rides_completed"`
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

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type StartRequest struct {
	RideID         string   `json:"ride_id"`
	DriverLocation Location `json:"driver_location"`
}

type StartResponse struct {
	RideID    string `json:"ride_id"`
	Status    string `json:"status"`
	StartedAt string `json:"started_at"`
	Message   string `json:"message"`
}

type CompleteRequest struct {
	RideID             string   `json:"ride_id"`
	FinalLocation      Location `json:"final_location"`
	ActualDistanceKm   float64  `json:"actual_distance_km"`
	ActualDurationMins float64  `json:"actual_duration_minutes"`
}

type CompleteResponse struct {
	RideID        string  `json:"ride_id"`
	Status        string  `json:"status"`
	CompletedAt   string  `json:"completed_at"`
	DriverEarning float64 `json:"driver_earning"`
	Message       string  `json:"message"`
}
