package service

import (
	"fmt"
	"math"
	"ride-hail/internal/common/model"
)

func (s *RideService) validateRideRequest(ride model.Ride) error {
	if ride.PassengerID == "" {
		return fmt.Errorf("passenger_id is required")
	}

	if ride.VehicleType == nil {
		return fmt.Errorf("vehicle_type is required")
	}

	switch *ride.VehicleType {
	case model.VehicleEconomy, model.VehiclePremium, model.VehicleXL:
	default:
		return fmt.Errorf("invalid vehicle_type: %s", *ride.VehicleType)
	}

	if ride.Status != nil && *ride.Status != model.RideRequested {
		return fmt.Errorf("invalid status for new ride: must be REQUESTED or nil")
	}

	if ride.DriverID != nil ||
		ride.StartedAt != nil ||
		ride.CompletedAt != nil ||
		ride.CancelledAt != nil {
		return fmt.Errorf("unexpected lifecycle fields for new ride")
	}

	return nil
}

func (s *RideService) validateCoordinates(pickup, destination model.Coordinate) error {
	if pickup.EntityID == "" || destination.EntityID == "" {
		return fmt.Errorf("entity_id is required for pickup and destination")
	}

	if len(pickup.Address) < 3 {
		return fmt.Errorf("pickup address is too short")
	}
	if len(destination.Address) < 3 {
		return fmt.Errorf("destination address is too short")
	}

	if pickup.EntityType != "" && pickup.EntityType != model.EntityTypePassenger {
		return fmt.Errorf("pickup.entity_type must be 'passenger'")
	}
	if destination.EntityType != "" && destination.EntityType != model.EntityTypePassenger {
		return fmt.Errorf("destination.entity_type must be 'passenger'")
	}

	if err := validateLatLon(pickup.Latitude, pickup.Longitude); err != nil {
		return fmt.Errorf("invalid pickup coordinates: %w", err)
	}
	if err := validateLatLon(destination.Latitude, destination.Longitude); err != nil {
		return fmt.Errorf("invalid destination coordinates: %w", err)
	}

	if areCoordinatesEqual(pickup, destination) {
		return fmt.Errorf("pickup and destination cannot be the same location")
	}

	if pickup.DistanceKm != nil || destination.DistanceKm != nil {
		return fmt.Errorf("distance_km should not be provided manually")
	}
	if pickup.DurationMinute != nil || destination.DurationMinute != nil {
		return fmt.Errorf("duration_minutes should not be provided manually")
	}
	if pickup.FareAmount != nil || destination.FareAmount != nil {
		return fmt.Errorf("fare_amount should not be provided manually")
	}

	return nil
}

func validateLatLon(lat, lon float64) error {
	if lat < -90 || lat > 90 {
		return fmt.Errorf("latitude out of range (-90..90)")
	}
	if lon < -180 || lon > 180 {
		return fmt.Errorf("longitude out of range (-180..180)")
	}
	return nil
}

func areCoordinatesEqual(a, b model.Coordinate) bool {
	const epsilon = 0.000001
	return math.Abs(a.Latitude-b.Latitude) < epsilon && math.Abs(a.Longitude-b.Longitude) < epsilon
}
