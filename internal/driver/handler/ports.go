package handler

import (
	"context"
	"ride-hail/internal/common/model"
)

type DriverService interface {
	GoOnline(ctx context.Context, driverID string, lat, lon float64) (model.OnlineResponse, error)
	GoOffline(ctx context.Context, driverID string) (model.OfflineResponse, error)
	Location(ctx context.Context, driverID string, req model.LocationRequest) (model.LocationResponse, error)
	Start(ctx context.Context, driverID string, rideId string, location model.Location) (model.StartResponse, error)
	Complete(ctx context.Context, driverID string, req model.CompleteRequest, location model.Location) (model.CompleteResponse, error)
}
