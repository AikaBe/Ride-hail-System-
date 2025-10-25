package dto

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

type RideResponse struct {
	RideID                   string  `json:"ride_id"`
	RideNumber               string  `json:"ride_number"`
	Status                   string  `json:"status"`
	EstimatedFare            float64 `json:"estimated_fare"`
	EstimatedDurationMinutes int     `json:"estimated_duration_minutes"`
	EstimatedDistanceKm      float64 `json:"estimated_distance_km"`
}

type CancelRideRequest struct {
	Reason string `json:"reason"`
}
