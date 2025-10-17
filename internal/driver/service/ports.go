package service

import (
	"context"
	"ride-hail/internal/common/model"
)

type DriverRepository interface {
	SetOnline(ctx context.Context, driverID string, lat, lon float64) (model.OnlineResponse, error)
	SetOffline(ctx context.Context, driverID string) (model.OfflineResponse, error)
	Location(ctx context.Context, driverID string, req model.LocationRequest) (model.LocationResponse, error)
	Start(ctx context.Context, driverID string, rideID string, req model.Location) (model.StartResponse, error)
	Complete(ctx context.Context, driverID string, driverEarning float64, req model.CompleteRequest, location model.Location) (model.CompleteResponse, error)
	GetRideStatus(ctx context.Context, driverID, rideID string) (string, error)
	GetDriverStatus(ctx context.Context, driverID string) (string, error)
}
