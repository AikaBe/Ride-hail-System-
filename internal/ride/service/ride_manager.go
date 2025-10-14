package service

import (
	"context"
	"fmt"
	"math"
	"ride-hail/internal/ride/model"
	"ride-hail/internal/ride/repository"

	"time"
)

type RideRepository interface {
	InsertRide(ctx context.Context, rideNumber, passengerID, rideType string, fare float64, pickupCoordID, destCoordID string) (string, error)
	InsertCoordinate(ctx context.Context, entityID, entityType, address string, latitude, longitude float64) (string, error)
	CancelRide(ctx context.Context, rideID, reason string) (*repository.CancelRideResponse, error)
}

type RideManager struct {
	Repo RideRepository
}

func NewRideManager(repo RideRepository) *RideManager {
	return &RideManager{Repo: repo}
}

func (m *RideManager) CreateRide(ctx context.Context, req model.RideRequest) (*model.RideResponse, error) {
	// Validate required fields
	if req.PassengerID == "" {
		return nil, fmt.Errorf("missing passenger_id")
	}
	if err := validateCoordinates(req.PickupLatitude, req.PickupLongitude); err != nil {
		return nil, fmt.Errorf("invalid pickup coordinates: %w", err)
	}
	if err := validateCoordinates(req.DestinationLatitude, req.DestinationLongitude); err != nil {
		return nil, fmt.Errorf("invalid destination coordinates: %w", err)
	}
	if req.PickupAddress == "" || req.DestinationAddress == "" {
		return nil, fmt.Errorf("pickup and destination addresses are required")
	}

	// Calculate fare
	distanceKm, durationMin, err := calculateRoute(req.PickupLatitude, req.PickupLongitude, req.DestinationLatitude, req.DestinationLongitude)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate route: %w", err)
	}

	estimatedFare, err := calculateFare(req.RideType, distanceKm, durationMin)
	if err != nil {
		return nil, err
	}

	// Generate ride number
	rideNumber := fmt.Sprintf("RIDE_%s", time.Now().Format("20060102_150405"))

	// Create coordinates first
	pickupCoordID, err := m.Repo.InsertCoordinate(ctx, "temp_pickup", "ride_pickup", req.PickupAddress, req.PickupLatitude, req.PickupLongitude)
	if err != nil {
		return nil, fmt.Errorf("failed to create pickup coordinate: %w", err)
	}

	destCoordID, err := m.Repo.InsertCoordinate(ctx, "temp_dest", "ride_destination", req.DestinationAddress, req.DestinationLatitude, req.DestinationLongitude)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination coordinate: %w", err)
	}

	// Insert ride with coordinates
	rideID, err := m.Repo.InsertRide(ctx, rideNumber, req.PassengerID, req.RideType, estimatedFare, pickupCoordID, destCoordID)
	if err != nil {
		return nil, fmt.Errorf("failed to create ride: %w", err)
	}

	// Update coordinates with actual ride ID
	// This would typically be done in a more sophisticated way, but for now we'll proceed

	return &model.RideResponse{
		RideID:                   rideID,
		RideNumber:               rideNumber,
		Status:                   "REQUESTED",
		EstimatedFare:            estimatedFare,
		EstimatedDurationMinutes: durationMin,
		EstimatedDistanceKm:      distanceKm,
	}, nil
}

func (m *RideManager) CancelRide(ctx context.Context, rideID, reason string) (*repository.CancelRideResponse, error) {
	if rideID == "" {
		return nil, fmt.Errorf("ride_id is required")
	}
	if reason == "" {
		return nil, fmt.Errorf("cancellation reason is required")
	}

	return m.Repo.CancelRide(ctx, rideID, reason)
}

func validateCoordinates(lat, lng float64) error {
	if lat < -90 || lat > 90 {
		return fmt.Errorf("latitude must be between -90 and 90")
	}
	if lng < -180 || lng > 180 {
		return fmt.Errorf("longitude must be between -180 and 180")
	}
	return nil
}

func calculateRoute(pickupLat, pickupLng, destLat, destLng float64) (distanceKm, durationMin float64, err error) {
	// Simplified calculation using Haversine formula for demonstration
	// In production, you would use a proper routing service
	const earthRadiusKm = 6371.0

	lat1 := pickupLat * (3.141592653589793 / 180.0)
	lng1 := pickupLng * (3.141592653589793 / 180.0)
	lat2 := destLat * (3.141592653589793 / 180.0)
	lng2 := destLng * (3.141592653589793 / 180.0)

	dlat := lat2 - lat1
	dlng := lng2 - lng1

	a := math.Sin(dlat/2)*math.Sin(dlat/2) + math.Cos(lat1)*math.Cos(lat2)*math.Sin(dlng/2)*math.Sin(dlng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	distanceKm = earthRadiusKm * c

	// Assume average speed of 30 km/h in urban areas
	durationMin = (distanceKm / 30.0) * 60.0

	return distanceKm, durationMin, nil
}

func calculateFare(rideType string, distanceKm, durationMin float64) (float64, error) {
	var baseFare, perKm, perMin float64

	switch rideType {
	case "ECONOMY":
		baseFare, perKm, perMin = 500, 100, 50
	case "PREMIUM":
		baseFare, perKm, perMin = 800, 120, 60
	case "XL":
		baseFare, perKm, perMin = 1000, 150, 75
	default:
		return 0, fmt.Errorf("invalid ride_type: %s", rideType)
	}

	return baseFare + (distanceKm * perKm) + (durationMin * perMin), nil
}
