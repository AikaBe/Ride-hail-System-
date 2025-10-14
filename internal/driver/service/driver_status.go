package service

import (
	"context"
	"ride-hail/internal/common/model"
	"ride-hail/internal/driver/repository"
)

type DriverService struct {
	repo *repository.DriverRepository
}

func NewDriverService(repo *repository.DriverRepository) *DriverService {
	return &DriverService{repo: repo}
}

func (s *DriverService) GoOnline(ctx context.Context, driverID string, lat, lon float64) (model.OnlineResponse, error) {
	return s.repo.SetOnline(ctx, driverID, lat, lon)
}

func (s *DriverService) GoOffline(ctx context.Context, driverID string) (model.OfflineResponse, error) {
	return s.repo.SetOffline(ctx, driverID)
}

func (s *DriverService) Location(ctx context.Context, driverID string, req model.LocationRequest) (model.LocationResponse, error) {
	return s.repo.Location(ctx, driverID, req)
}

func (s *DriverService) Start(ctx context.Context, driverID string, req model.StartRequest) (model.StartResponse, error) {
	return s.repo.Start(ctx, driverID, req)
}
