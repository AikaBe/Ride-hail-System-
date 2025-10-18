package handler

import (
	"fmt"
	"ride-hail/internal/common/model"
)

// MapRideRequestToEntities maps raw HTTP request data into domain entities.
// Performs only syntactic validation (field presence, coordinate ranges).
// Does NOT fill domain-specific fields like RideNumber, RequestedAt, or Status.
func MapRideRequestToEntities(req RideRequest) (model.Ride, model.Coordinate, model.Coordinate, error) {
	if req.PassengerID == "" {
		return model.Ride{}, model.Coordinate{}, model.Coordinate{}, fmt.Errorf("passenger_id is required")
	}
	if req.PickupAddress == "" || req.DestinationAddress == "" {
		return model.Ride{}, model.Coordinate{}, model.Coordinate{}, fmt.Errorf("pickup and destination addresses are required")
	}
	if req.PickupLatitude < -90 || req.PickupLatitude > 90 || req.DestinationLatitude < -90 || req.DestinationLatitude > 90 {
		return model.Ride{}, model.Coordinate{}, model.Coordinate{}, fmt.Errorf("latitude out of range (-90..90)")
	}
	if req.PickupLongitude < -180 || req.PickupLongitude > 180 || req.DestinationLongitude < -180 || req.DestinationLongitude > 180 {
		return model.Ride{}, model.Coordinate{}, model.Coordinate{}, fmt.Errorf("longitude out of range (-180..180)")
	}

	pickup := model.Coordinate{
		EntityID:  model.UUID(req.PassengerID),
		Address:   req.PickupAddress,
		Latitude:  req.PickupLatitude,
		Longitude: req.PickupLongitude,
	}

	destination := model.Coordinate{
		EntityID:  model.UUID(req.PassengerID),
		Address:   req.DestinationAddress,
		Latitude:  req.DestinationLatitude,
		Longitude: req.DestinationLongitude,
	}

	vehicleType := model.VehicleType(req.RideType)

	ride := model.Ride{
		PassengerID: model.UUID(req.PassengerID),
		VehicleType: &vehicleType,
	}

	return ride, pickup, destination, nil
}
